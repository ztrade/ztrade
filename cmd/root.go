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

	"github.com/ztrade/exchange"
	"github.com/ztrade/ztrade/pkg/ctl"
	"github.com/ztrade/ztrade/pkg/process/dbstore"

	homedir "github.com/mitchellh/go-homedir"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	_ "github.com/ztrade/exchange/include"
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
Backtest with golang script/plugin`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	//	Run: func(cmd *cobra.Command, args []string) { },
}

// RootCmd export RootCmd
func RootCmd() *cobra.Command {
	return rootCmd
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
	rootCmd.PersistentFlags().BoolVarP(&runPprof, "pprof", "p", false, "run with pprof mode at :8088")
	rootCmd.PersistentFlags().StringVarP(&logFile, "log", "l", "ztrade.log", "log file")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	rootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		var err error
		if logFile != "" {
			logF, err = os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
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
			if err := http.ListenAndServe("127.0.0.1:8088", nil); err != nil {
				log.Errorf("pprof server error: %s", err.Error())
			}
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
		ctl.SetConfig(exchange.WrapViper(viper.GetViper()))
	}
}

func initTimerange(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVarP(&startStr, "start", "s", "2019-01-01 10:00:00", "start time")
	cmd.PersistentFlags().StringVarP(&endStr, "end", "e", "", "end time")
	cmd.PersistentFlags().StringVar(&symbol, "symbol", "BTCUSDT", "symbol")
	cmd.PersistentFlags().StringVar(&exchangeName, "exchange", "binance", "exchange name, support binance,okex current now")
}

func parseTimerange() (startTime, endTime time.Time, err error) {
	if startStr == "" {
		err = errors.New("start/end time can't be empty")
		return
	}
	startTime, err = time.Parse("2006-01-02 15:04:05", startStr)
	if err != nil {
		err = errors.New("parse start time error")
		return
	}
	if endStr == "" {
		endTime = time.Now()
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
