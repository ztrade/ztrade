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

	"github.com/ztrade/base/common"
	"golang.org/x/mod/modfile"
	"golang.org/x/mod/module"
)

var (
	buildInfo *debug.BuildInfo

	//go:embed tmpl/define.go
	defineGo string
	//go:embed tmpl/export.go
	exportGo string

	// func (s *DemoStrategy) Init(engine Engine, params ParamData) (err error) {
	nameReg = regexp.MustCompile(`func\s+\([^)]+\s+\*?(\w+)\)`)
)

type Builder struct {
	source    string
	output    string
	debugMode bool
	keepTemp  bool
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

func (b *Builder) Build() (err error) {
	if b.source == "" {
		err = errors.New("strategy file can't be empty")
		return
	}
	if b.output == "" {
		b.output = strings.Replace(b.source, ".go", ".so", 1)
	}
	baseName := filepath.Base(b.source)
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
	err = common.CopyWithMainPkg(filepath.Join(tempDir, baseName), b.source)
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
	if b.keepTemp {
		fmt.Println("temp dir:", tempDir)
	}
	dst, _ := filepath.Abs(b.output)
	runGoGet := true
	var output []byte
	for i := 0; i != 2; i++ {
		if i == 1 {
			runGoGet, err = b.fixGoMod(tempDir)
			if err != nil {
				err = fmt.Errorf("fixGoMod failed: %w", err)
				return
			}
			if !runGoGet {
				break
			}
		}

		eBuildGet := exec.Command("go", "get", "-v")
		eBuildGet.Dir = tempDir
		output, err = eBuildGet.CombinedOutput()
		if err != nil {
			err = fmt.Errorf("run go get command failed: %w, %s", err, string(output))
			return
		}
	}

	eBuild := exec.Command("go", "build", "--buildmode=plugin", "-o", dst)
	eBuild.Dir = tempDir
	output, err = eBuild.CombinedOutput()
	if err != nil {
		err = fmt.Errorf("run build command failed: %w, %s", err, string(output))
		return
	}
	return
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
		if b.debugMode {
			fmt.Println("fix path version:", v.Path, v.Version)
		}
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
