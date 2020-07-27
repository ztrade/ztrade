package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	log "github.com/sirupsen/logrus"
	"github.com/ztrade/ztrade/pkg/common"
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

func init() {
	rootCmd.AddCommand(tradeCmd)
	tradeCmd.PersistentFlags().StringVar(&scriptFile, "script", "", "script file to backtest")
	tradeCmd.PersistentFlags().StringVarP(&rptFile, "report", "o", "report.html", "output report html file path")
	tradeCmd.PersistentFlags().StringVarP(&binSize, "binSize", "b", "1m", "binSize: 1m,5m,15m,1h,1d")
	tradeCmd.PersistentFlags().StringVar(&symbol, "symbol", "XBTUSD", "symbol")
	tradeCmd.PersistentFlags().StringVar(&exchangeName, "exchange", "bitmex", "exchage name, only support bitmex current now")
}

func runTrade(cmd *cobra.Command, args []string) {
	if scriptFile == "" {
		log.Fatal("strategy file can't be empty")
		return
	}
	var err error
	var gracefulStop = make(chan os.Signal)
	signal.Notify(gracefulStop, syscall.SIGTERM)
	signal.Notify(gracefulStop, syscall.SIGINT)
	real, err := ctl.NewTrade(exchangeName, symbol)
	r := report.NewReportSimple()
	real.SetReporter(r)
	real.AddScript(filepath.Base(scriptFile), scriptFile, nil)
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
	err = r.GenRPT(rptFile)
	if err != nil {
		return
	}
	fmt.Println("open report ", rptFile)
	err = common.OpenURL(rptFile)

	return
}
