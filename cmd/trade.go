package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/ztrade/base/common"
	"github.com/ztrade/ztrade/pkg/ctl"
	"github.com/ztrade/ztrade/pkg/report"

	"github.com/spf13/cobra"
)

// tradeCmd represents the trade command
var tradeCmd = &cobra.Command{
	Use:   "trade",
	Short: "trade with script",
	Long:  `trade with script`,
	Run:   runTrade,
}

var (
	recentDay int
)

func init() {
	rootCmd.AddCommand(tradeCmd)
	tradeCmd.PersistentFlags().StringVar(&scriptFile, "script", "", "script file to backtest")
	tradeCmd.PersistentFlags().StringVarP(&rptFile, "report", "o", "report.html", "output report html file path")
	tradeCmd.PersistentFlags().StringVarP(&binSize, "binSize", "b", "1m", "binSize: 1m,5m,15m,1h,1d")
	tradeCmd.PersistentFlags().StringVar(&symbol, "symbol", "XBTUSD", "symbol")
	tradeCmd.PersistentFlags().StringVar(&exchangeName, "exchange", "bitmex", "exchage name, only support bitmex current now")
	tradeCmd.PersistentFlags().IntVarP(&recentDay, "recent", "r", 1, "load recent (n) day datas,default 1")
	tradeCmd.PersistentFlags().StringVar(&param, "param", "", "param json string")
}

func runTrade(cmd *cobra.Command, args []string) {
	if scriptFile == "" {
		log.Fatal("strategy file can't be empty")
		return
	}
	var gracefulStop = make(chan os.Signal, 1)
	signal.Notify(gracefulStop, syscall.SIGTERM, syscall.SIGINT)
	real, err := ctl.NewTrade(exchangeName, symbol)
	if err != nil {
		log.Fatal("trade error:", err.Error())
		return
	}
	if recentDay != 0 {
		real.SetLoadRecent(time.Duration(recentDay) * time.Hour * 24)
	}
	r := report.NewReportSimple()
	tStart := time.Now()
	real.SetReporter(r)
	paramData := make(map[string]interface{})
	if param != "" {
		err = json.Unmarshal([]byte(param), &paramData)
		if err != nil {
			log.Fatal("param error:", err.Error())
		}
	}
	err = real.AddScript(filepath.Base(scriptFile), scriptFile, param)
	if err != nil {
		fmt.Println("AddScript failed:", err.Error())
		return
	}
	// real.SetScript(scriptFile)
	go func() {
		sig := <-gracefulStop
		fmt.Printf("caught sig: %+v", sig)
		real.Stop()
	}()
	err = real.Start()
	if err != nil {
		log.Fatal("trade error:", err.Error())
	}
	real.Wait()
	fmt.Println("begin to geneate report to ", rptFile)
	r.SetTimeRange(tStart, time.Now())
	err = r.GenRPT(rptFile)
	if err != nil {
		return
	}
	fmt.Println("open report ", rptFile)
	err = common.OpenURL(rptFile)
	if err != nil {
		log.Fatal("open report failed:", err.Error())
	}
	return
}
