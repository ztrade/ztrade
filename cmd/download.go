package cmd

import (
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

func init() {
	rootCmd.AddCommand(downloadCmd)
	initTimerange(downloadCmd)
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
	down := ctl.NewDataDownload(cfg, db, exchangeName, symbol, binSize, startTime, endTime)
	err = down.Run()
	if err != nil {
		log.Fatal("download data error", err.Error())
	}
}
