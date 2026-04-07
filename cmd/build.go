package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/ztrade/ztrade/pkg/ctl"
)

// buildCmd represents the build command
var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "build script to go plugin",
	Long:  `"build script to go plugin`,
	Run:   runBuild,
}

var (
	output                 string
	keepTemp               bool
	moduleRoot             string
	ignoreSourceModuleRoot bool
)

func init() {
	rootCmd.AddCommand(buildCmd)
	buildCmd.PersistentFlags().StringVar(&scriptFile, "script", "", "script file to backtest")
	buildCmd.PersistentFlags().StringVar(&output, "output", "", "plugin output file")
	buildCmd.PersistentFlags().StringVar(&moduleRoot, "moduleRoot", "", "optional module root containing go.mod/go.sum/replace for private dependencies")
	buildCmd.PersistentFlags().BoolVar(&ignoreSourceModuleRoot, "ignoreSourceModuleRoot", false, "ignore go.mod discovered from the script directory or its parent directories")
	buildCmd.PersistentFlags().BoolVarP(&keepTemp, "keep", "k", false, "keep temp dir")
}

func runBuild(cmd *cobra.Command, args []string) {
	b := ctl.NewBuilder(scriptFile, output)
	b.SetKeepTemp(keepTemp)
	b.SetModuleRoot(moduleRoot)
	b.SetIgnoreSourceModuleRoot(ignoreSourceModuleRoot)
	err := b.Build()
	if err != nil {
		fmt.Println("build failed:", err.Error())
		return
	}
	fmt.Printf("build success: %s\n", output)
}
