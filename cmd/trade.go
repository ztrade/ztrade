/*
Copyright © 2019 NAME HERE <EMAIL ADDRESS>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/ztrade/ztrade/pkg/common"
	"github.com/ztrade/ztrade/pkg/ctl"
	"github.com/ztrade/ztrade/pkg/process/rpt"

	log "github.com/sirupsen/logrus"

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
	r := rpt.NewRPTProcesser()
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
