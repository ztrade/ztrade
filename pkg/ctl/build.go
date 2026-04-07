package ctl

import (
	_ "embed"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime/debug"
	"strings"
	"text/template"
	"time"

	"github.com/ztrade/base/common"
	"golang.org/x/mod/modfile"
	"golang.org/x/mod/module"
)

var (
	buildInfo           *debug.BuildInfo
	goModTidyRetryDelay = time.Second

	//go:embed tmpl/define.go.tmpl
	defineGo string
	//go:embed tmpl/export.go.tmpl
	exportGo string

	// func (s *DemoStrategy) Init(engine Engine, params ParamData) (err error) {
	nameReg = regexp.MustCompile(`func\s+\([^)]+\s+\*?(\w+)\)`)
)

type Builder struct {
	source                 string
	output                 string
	debugMode              bool
	keepTemp               bool
	moduleRoot             string
	ignoreSourceModuleRoot bool
}

func NewBuilder(src, output string) *Builder {
	b := new(Builder)
	b.source = src
	b.output = output
	return b
}

func (b *Builder) SetKeepTemp(keepTemp bool) {
	b.keepTemp = keepTemp
}

func (b *Builder) SetModuleRoot(moduleRoot string) {
	b.moduleRoot = moduleRoot
}

func (b *Builder) SetIgnoreSourceModuleRoot(ignore bool) {
	b.ignoreSourceModuleRoot = ignore
}

func (b *Builder) Build() (err error) {
	if b.source == "" {
		err = errors.New("strategy file can't be empty")
		return
	}
	sourcePath, err := filepath.Abs(b.source)
	if err != nil {
		err = fmt.Errorf("resolve source path failed: %w", err)
		return
	}
	if b.output == "" {
		b.output = strings.Replace(sourcePath, ".go", ".so", 1)
	}
	baseName := filepath.Base(sourcePath)
	dir := baseName[0 : len(baseName)-len(filepath.Ext(b.source))]
	tempDir, err := os.MkdirTemp("", dir)
	if err != nil {
		err = fmt.Errorf("create temp dir failed: %w", err)
		return
	}
	defer func() {
		if !b.keepTemp {
			os.RemoveAll(tempDir)
		}
	}()
	err = common.CopyWithMainPkg(filepath.Join(tempDir, baseName), sourcePath)
	if err != nil {
		err = fmt.Errorf("copy file failed: %w", err)
		return
	}
	err = os.WriteFile(filepath.Join(tempDir, "define.go"), []byte(defineGo), 0644)
	// err = common.CopyWithMainPkg(filepath.Join(tempDir, "define.go"), filepath.Join(common.GetExecDir(), "tmpl", "define.go"))
	if err != nil {
		err = fmt.Errorf("write tmpl file define.go failed: %w", err)
		return
	}
	// fTmpl := filepath.Join(common.GetExecDir(), "tmpl", "export.go")
	// tmpl, err := template.ParseFiles(fTmpl)
	tmpl, err := template.New("export").Parse(exportGo)
	if err != nil {
		err = fmt.Errorf("export.go error: %w", err)
		return
	}
	fExport, err := os.Create(filepath.Join(tempDir, "export.go"))
	if err != nil {
		err = fmt.Errorf("create export.go error: %w", err)
		return
	}
	name, err := b.getName(b.source)
	if err != nil {
		err = fmt.Errorf("get name error: %w", err)
		return
	}
	fmt.Println("find strategy name: ", name)
	err = tmpl.Execute(fExport, map[string]string{"Name": name})
	if err != nil {
		err = fmt.Errorf("create export.go error: %w", err)
		return
	}
	e := exec.Command("go", "mod", "init", dir)
	e.Dir = tempDir
	err = e.Run()
	if err != nil {
		err = fmt.Errorf("run command failed: %w", err)
		return
	}
	err = b.prepareModuleContext(tempDir, filepath.Dir(sourcePath))
	if err != nil {
		err = fmt.Errorf("prepare module context failed: %w", err)
		return
	}
	err = b.syncGoMod(tempDir)
	if err != nil {
		err = fmt.Errorf("sync go.mod failed: %w", err)
		return
	}
	if b.keepTemp {
		fmt.Println("temp dir:", tempDir)
	}
	dst, _ := filepath.Abs(b.output)
	var output []byte
	eBuild := exec.Command("go", "build", "--buildmode=plugin", "-o", dst)
	eBuild.Dir = tempDir
	output, err = eBuild.CombinedOutput()
	if err != nil {
		err = fmt.Errorf("run build command failed: %w, %s", err, string(output))
		return
	}
	return
}

func (b *Builder) prepareModuleContext(tempDir, sourceDir string) error {
	moduleRoot, explicit, err := b.resolveModuleRoot(sourceDir)
	if err != nil {
		return err
	}
	if moduleRoot == "" {
		return nil
	}
	if explicit {
		fmt.Println("use custom go.mod:", filepath.Join(moduleRoot, "go.mod"))
	} else {
		fmt.Println("use discovered go.mod:", filepath.Join(moduleRoot, "go.mod"))
	}
	if err := b.mergeGoMod(filepath.Join(tempDir, "go.mod"), filepath.Join(moduleRoot, "go.mod"), moduleRoot); err != nil {
		return err
	}
	goSum := filepath.Join(moduleRoot, "go.sum")
	if _, err := os.Stat(goSum); err == nil {
		if err := common.Copy(filepath.Join(tempDir, "go.sum"), goSum); err != nil {
			return err
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func (b *Builder) mergeGoMod(targetGoMod, sourceGoMod, moduleRoot string) error {
	targetBuf, err := os.ReadFile(targetGoMod)
	if err != nil {
		return err
	}
	targetMf, err := modfile.Parse(targetGoMod, targetBuf, nil)
	if err != nil {
		return err
	}
	sourceBuf, err := os.ReadFile(sourceGoMod)
	if err != nil {
		return err
	}
	sourceMf, err := modfile.Parse(sourceGoMod, sourceBuf, nil)
	if err != nil {
		return err
	}
	for _, req := range sourceMf.Require {
		if err := targetMf.AddRequire(req.Mod.Path, req.Mod.Version); err != nil {
			return err
		}
	}
	for _, replace := range sourceMf.Replace {
		newPath := replace.New.Path
		if replace.New.Version == "" && strings.HasPrefix(newPath, ".") && !filepath.IsAbs(newPath) {
			newPath = filepath.Clean(filepath.Join(moduleRoot, newPath))
		}
		if err := targetMf.AddReplace(replace.Old.Path, replace.Old.Version, newPath, replace.New.Version); err != nil {
			return err
		}
	}
	formatted, err := targetMf.Format()
	if err != nil {
		return err
	}
	return os.WriteFile(targetGoMod, formatted, 0644)
}

func (b *Builder) syncGoMod(dir string) error {
	if err := b.runGoModTidy(dir); err != nil {
		return err
	}
	runAgain, err := b.fixGoMod(dir)
	if err != nil {
		return err
	}
	if runAgain {
		if err := b.runGoModTidy(dir); err != nil {
			return err
		}
	}
	return nil
}

func (b *Builder) runGoModTidy(dir string) error {
	var lastErr error
	for attempt := 1; attempt <= 3; attempt++ {
		e := exec.Command("go", "mod", "tidy")
		e.Dir = dir
		output, err := e.CombinedOutput()
		if err == nil {
			return nil
		}
		lastErr = fmt.Errorf("run go mod tidy failed on attempt %d: %w, %s", attempt, err, string(output))
		if attempt < 3 {
			time.Sleep(time.Duration(attempt) * goModTidyRetryDelay)
		}
	}
	return lastErr
}

func (b *Builder) fixGoMod(dir string) (hasFixed bool, err error) {
	gomod := filepath.Join(dir, "go.mod")
	f, err := os.Open(gomod)
	if err != nil {
		return
	}
	buf, err := io.ReadAll(f)
	if err != nil {
		f.Close()
		return
	}
	f.Close()
	mf, err := modfile.Parse("go.mod", buf, nil)
	if err != nil {
		return
	}
	moduleRoot, _, err := b.resolveModuleRoot(filepath.Dir(b.source))
	if err != nil {
		return false, err
	}
	var ver string
	var replaceModPaths []module.Version
	for _, v := range mf.Require {
		ver = fixRequireVersion(v.Mod.Path)
		if ver != "" && ver != v.Mod.Version {
			replaceModPaths = append(replaceModPaths, module.Version{Path: v.Mod.Path, Version: ver})
		}
	}
	for _, v := range replaceModPaths {
		mf.AddRequire(v.Path, v.Version)
		hasFixed = true
		if b.debugMode {
			fmt.Println("fix path version:", v.Path, v.Version)
		}
	}
	type replaceRewrite struct {
		oldPath string
		oldVer  string
		newPath string
		newVer  string
	}
	var rewrites []replaceRewrite
	for _, replace := range mf.Replace {
		if moduleRoot == "" || replace.New.Version != "" || filepath.IsAbs(replace.New.Path) || !strings.HasPrefix(replace.New.Path, ".") {
			continue
		}
		absPath := filepath.Clean(filepath.Join(moduleRoot, replace.New.Path))
		if absPath == replace.New.Path {
			continue
		}
		rewrites = append(rewrites, replaceRewrite{
			oldPath: replace.Old.Path,
			oldVer:  replace.Old.Version,
			newPath: absPath,
			newVer:  replace.New.Version,
		})
		hasFixed = true
	}
	for _, rewrite := range rewrites {
		if err = mf.DropReplace(rewrite.oldPath, rewrite.oldVer); err != nil {
			return false, err
		}
		if err = mf.AddReplace(rewrite.oldPath, rewrite.oldVer, rewrite.newPath, rewrite.newVer); err != nil {
			return false, err
		}
	}
	if !hasFixed {
		return false, nil
	}

	buf, err = mf.Format()
	if err != nil {
		return
	}
	f, err = os.OpenFile(gomod, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.ModePerm)
	if err != nil {
		return
	}
	_, err = f.Write(buf)
	f.Close()
	hasFixed = true
	return
}

func (b *Builder) resolveModuleRoot(sourceDir string) (string, bool, error) {
	if b.moduleRoot != "" {
		root, err := filepath.Abs(b.moduleRoot)
		if err != nil {
			return "", false, err
		}
		if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return "", false, fmt.Errorf("module root %s does not contain go.mod", root)
			}
			return "", false, err
		}
		return root, true, nil
	}
	if b.ignoreSourceModuleRoot {
		return "", false, nil
	}
	root, err := findModuleRoot(sourceDir)
	return root, false, err
}

func findModuleRoot(start string) (string, error) {
	current, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(current, "go.mod")); err == nil {
			return current, nil
		} else if !errors.Is(err, os.ErrNotExist) {
			return "", err
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", nil
		}
		current = parent
	}
}

func (b *Builder) getName(src string) (name string, err error) {
	buf, err := os.ReadFile(src)
	if err != nil {
		return
	}
	lines := strings.Split(string(buf), "\n")
	for _, v := range lines {
		if strings.Contains(v, "Init(") {
			rets := nameReg.FindStringSubmatch(v)
			if len(rets) > 1 {
				name = rets[1]
				return
			}
			return
		}
	}
	return "", errors.New("can't find init function")
}

func fixRequireVersion(modPath string) (ver string) {
	if buildInfo == nil {
		return
	}
	for _, v := range buildInfo.Deps {
		if v.Path == modPath {
			return v.Version
		}
	}
	return
}

func init() {
	var ok bool
	buildInfo, ok = debug.ReadBuildInfo()
	if !ok {
		panic("read build info failed")
	}
}
