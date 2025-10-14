package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/ztrade/base/common"
	"github.com/ztrade/ztrade/pkg/ctl"

	log "github.com/sirupsen/logrus"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/ztrade/ztrade/pkg/report"
)

var (
	scriptFile   string
	rptFile      string
	startStr     string
	endStr       string
	binSize      string
	symbol       string
	exchangeName string
	balanceInit  float64
	param        string
	loadOnce     int
	fee          float64
	lever        float64
	simpleReport bool

	rptDB string
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
	backtestCmd.PersistentFlags().Float64VarP(&balanceInit, "balance", "", 100000, "init total balance")
	backtestCmd.PersistentFlags().StringVar(&param, "param", "", "param json string")
	backtestCmd.PersistentFlags().IntVarP(&loadOnce, "load", "", 50000, "load db once limit")
	backtestCmd.PersistentFlags().Float64VarP(&fee, "fee", "", 0.0001, "fee")
	backtestCmd.PersistentFlags().Float64VarP(&lever, "lever", "", 1, "lever")
	backtestCmd.PersistentFlags().BoolVarP(&simpleReport, "console", "", false, "print report to console")
	backtestCmd.PersistentFlags().StringVarP(&rptDB, "reportDB", "d", "", "save all actions to sqlite db")
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

	r := report.NewReportSimple()
	back, err := ctl.NewBacktest(db, exchangeName, symbol, param, startTime, endTime)
	if err != nil {
		log.Fatal("init backtest failed:", err.Error())
	}
	r.SetTimeRange(startTime, endTime)
	back.SetScript(scriptFile)
	back.SetReporter(r)
	back.SetBalanceInit(balanceInit, fee)
	back.SetLoadDBOnce(loadOnce)
	back.SetLever(lever)

	err = back.Run()

	if err != nil {
		fmt.Println("run backtest error", err.Error())
		log.Fatal("run backtest error", err.Error())
	}
	if simpleReport {
		result, err := r.GetResult()
		if err != nil {
			return
		}
		//		for _, v := range result.Actions {
		//			fmt.Println(v.Time, v.Action, v.Amount, v.Price, v.Profit, v.TotalProfit)
		//		}
		buf, err := json.Marshal(result)
		if err != nil {
			return
		}
		fmt.Println(string(buf))
		return
	}
	err = r.GenRPT(rptFile)
	if err != nil {
		return
	}
	if rptDB != "" {
		err = r.ExportToDB(rptDB)
		if err != nil {
			fmt.Println("export to DB failed:", err.Error())
			return
		}
	}
	err = common.OpenURL(rptFile)
	return
}
