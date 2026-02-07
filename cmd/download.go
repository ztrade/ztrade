package cmd

import (
	"fmt"

	"github.com/ztrade/ztrade/pkg/ctl"

	log "github.com/sirupsen/logrus"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// downloadCmd represents the download command
var downloadCmd = &cobra.Command{
	Use:   "download",
	Short: "download data from exchange",
	Long:  `download data from exchange`,
	Run:   runDownload,
}

var (
	bAuto *bool
)

func init() {
	rootCmd.AddCommand(downloadCmd)
	initTimerange(downloadCmd)
	downloadCmd.PersistentFlags().StringVarP(&binSize, "binSize", "b", "1m", "binSize of kline to download: 1m,5m,15m,1h,1d")
	bAuto = downloadCmd.PersistentFlags().BoolP("auto", "a", false, "auto download")
}

func runDownload(cmd *cobra.Command, args []string) {
	cfg := viper.GetViper()
	startTime, endTime, err := parseTimerange()
	if err != nil {
		log.Fatal(err.Error())
		return
	}
	db, err := initDB(cfg)
	if err != nil {
		log.Fatal("init db failed:", err.Error())
	}
	var down *ctl.DataDownload
	if *bAuto {
		down = ctl.NewDataDownloadAuto(cfg, db, exchangeName, symbol, binSize)
	} else {
		down = ctl.NewDataDownload(cfg, db, exchangeName, symbol, binSize, startTime, endTime)
	}
	err = down.Run()
	if err != nil {
		fmt.Println("download data error", err.Error())
		log.Fatal("download data error", err.Error())
	}
}
