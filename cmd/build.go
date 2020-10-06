package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/ztrade/base/common"
	"github.com/ztrade/ztrade/pkg/process/goscript/engine"
)

// buildCmd represents the build command
var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "build script to go plugin",
	Long:  `"build script to go plugin`,
	Run:   runBuild,
}

var (
	output string
)

func init() {
	rootCmd.AddCommand(buildCmd)
	buildCmd.PersistentFlags().StringVar(&scriptFile, "script", "", "script file to backtest")
	buildCmd.PersistentFlags().StringVar(&output, "output", "", "plugin output file")
}

func runBuild(cmd *cobra.Command, args []string) {
	if scriptFile == "" {
		log.Fatal("strategy file can't be empty")
		return
	}
	if output == "" {
		output = strings.Replace(scriptFile, ".go", ".so", 1)
	}
	baseName := filepath.Base(scriptFile)
	dir := baseName[0 : len(baseName)-len(filepath.Ext(scriptFile))]
	tempDir, err := ioutil.TempDir("", dir)
	if err != nil {
		log.Fatal("create temp dir failed:", err.Error())
	}
	defer func() {
		// os.RemoveAll(tempDir)
	}()

	err = common.CopyWithMainPkg(filepath.Join(tempDir, baseName), scriptFile)
	if err != nil {
		log.Fatal("copy file failed:", err.Error())
	}
	err = common.CopyWithMainPkg(filepath.Join(tempDir, "define.go"), filepath.Join(common.GetExecDir(), "tmpl", "define.go"))
	if err != nil {
		log.Fatal("copy tmpl file failed:", err.Error())
	}
	runner, err := engine.NewRunner(scriptFile)
	if err != nil {
		log.Fatal("export.go error:", err.Error())
	}
	fTmpl := filepath.Join(common.GetExecDir(), "tmpl", "export.go")
	tmpl, err := template.ParseFiles(fTmpl)
	if err != nil {
		log.Fatal("export.go error:", err.Error())
	}

	fExport, err := os.Create(filepath.Join(tempDir, "export.go"))
	if err != nil {
		log.Fatal("create export.go error:", err.Error())
	}

	err = tmpl.Execute(fExport, map[string]string{"Name": runner.GetName()})
	if err != nil {
		log.Fatal("create export.go error:", err.Error())
	}

	e := exec.Command("go", "mod", "init", dir)
	e.Dir = tempDir
	err = e.Run()
	if err != nil {
		log.Fatal("run command failed:", err.Error())
	}
	dst, _ := filepath.Abs(output)
	eBuild := exec.Command("go", "build", "--buildmode=plugin", "-o", dst)
	eBuild.Dir = tempDir
	err = eBuild.Run()
	if err != nil {
		log.Fatal("run build command failed:", err.Error())
	}
	// fmt.Println(tempDir)
	fmt.Printf("build success: %s\n", output)
}
