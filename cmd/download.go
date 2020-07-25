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
