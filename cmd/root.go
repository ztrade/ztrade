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
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"net/http"
	_ "net/http/pprof"
	"ztrade/pkg/ctl"
	"ztrade/pkg/process/dbstore"

	homedir "github.com/mitchellh/go-homedir"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

var (
	cfgFile  string
	logFile  string
	debug    bool
	runPprof bool

	logF *os.File
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "ztrade",
	Short: "The last trade system you need",
	Long: `The last trade system you need.
Trade with all popular exchanges.
Backtest with javascrit.
View chanlun chart`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	//	Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	defer func() {
		if logF != nil {
			logF.Close()
		}
	}()
	if err := rootCmd.Execute(); err != nil {
		fmt.Println("run command error:", err.Error())
	}

}

func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file (default is $HOME/.configs/ztrade.yaml or ./configs/ztrade.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&debug, "debug", "P", false, "run debug mode")
	rootCmd.PersistentFlags().BoolVarP(&runPprof, "pprof", "p", false, "run with pprof mode at :8888")
	rootCmd.PersistentFlags().StringVarP(&logFile, "log", "l", "ztrade.log", "log file")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	rootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		var err error
		if logFile != "" {
			logF, err = os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY, 0666)
			if err != nil {
				log.Error("open log file failed:", err.Error())
			} else {
				log.SetOutput(logF)
			}
		}
		if debug {
			log.SetLevel(log.DebugLevel)
		}
		if !runPprof {
			return
		}
		go func() {
			http.ListenAndServe("0.0.0.0:8888", nil)
		}()

	}
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Search config in home directory with name ".ztrade" (without extension).
		viper.AddConfigPath(filepath.Join(home, ".configs"))
		viper.AddConfigPath("./configs")
		ex, err := os.Executable()
		if err != nil {
			panic(err)
		}
		exPath := filepath.Dir(ex)
		viper.AddConfigPath(filepath.Join(exPath, "configs"))
		viper.SetConfigName("ztrade")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
		ctl.SetConfig(viper.GetViper())
	}
}

func initTimerange(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVarP(&startStr, "start", "s", "2019-01-01 10:00:00", "start time")
	cmd.PersistentFlags().StringVarP(&endStr, "end", "e", "2019-01-01 10:00:00", "end time")
	cmd.PersistentFlags().StringVarP(&binSize, "binSize", "b", "1m", "binSize: 1m,5m,15m,1h,1d")
	cmd.PersistentFlags().StringVar(&symbol, "symbol", "XBTUSD", "symbol")
	cmd.PersistentFlags().StringVar(&exchangeName, "exchange", "bitmex", "exchage name, only support bitmex current now")
}

func parseTimerange() (startTime, endTime time.Time, err error) {
	if startStr == "" || endStr == "" {
		err = errors.New("start/end time can't be empty")
		return
	}
	startTime, err = time.Parse("2006-01-02 15:04:05", startStr)
	if err != nil {
		err = errors.New("parse start time error")
		return
	}
	endTime, err = time.Parse("2006-01-02 15:04:05", endStr)
	if err != nil {
		err = errors.New("parse end time error")
		return
	}
	return
}

func initDB(cfg *viper.Viper) (db *dbstore.DBStore, err error) {
	db, err = dbstore.LoadDB(cfg)
	return
}
