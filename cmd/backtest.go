package cmd

import (
	"fmt"
	"ztrade/pkg/ctl"

	log "github.com/sirupsen/logrus"

	"ztrade/pkg/common"
	"ztrade/pkg/process/rpt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	scriptFile   string
	rptFile      string
	startStr     string
	endStr       string
	binSize      string
	symbol       string
	exchangeName string
)

// backtestCmd represents the backtest command
var backtestCmd = &cobra.Command{
	Use:   "backtest",
	Short: "backtest with script",
	Long:  `backtest a script between start and end`,
	Run:   runBacktest,
}

func init() {
	rootCmd.AddCommand(backtestCmd)

	backtestCmd.PersistentFlags().StringVar(&scriptFile, "script", "", "script file to backtest")
	backtestCmd.PersistentFlags().StringVarP(&rptFile, "report", "o", "report.html", "output report html file path")
	initTimerange(backtestCmd)
}

func runBacktest(cmd *cobra.Command, args []string) {
	if scriptFile == "" {
		log.Fatal("strategy file can't be empty")
		return
	}
	startTime, endTime, err := parseTimerange()
	if err != nil {
		log.Fatal(err.Error())
		return
	}
	cfg := viper.GetViper()
	db, err := initDB(cfg)
	if err != nil {
		log.Fatal("init db failed:", err.Error())
	}

	r := rpt.NewRPTProcesser()
	back, err := ctl.NewBacktest(db, exchangeName, symbol, startTime, endTime)
	if err != nil {
		log.Fatal("init backtest failed:", err.Error())
	}
	back.SetScript(scriptFile)
	back.SetReporter(r)
	err = back.Run()

	if err != nil {
		fmt.Println("run backtest error", err.Error())
		log.Fatal("run backtest error", err.Error())
	}
	err = r.GenRPT(rptFile)
	if err != nil {
		return
	}
	err = common.OpenURL(rptFile)
	return
}
