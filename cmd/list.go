package cmd

import (
	"fmt"
	"os"

	"github.com/olekukonko/tablewriter"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/ztrade/ztrade/pkg/ctl"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "list local datas",
	Long:  `list local datas`,
	Run:   runList,
}

func init() {
	rootCmd.AddCommand(listCmd)

}
func runList(cmd *cobra.Command, args []string) {
	cfg := viper.GetViper()
	db, err := initDB(cfg)
	if err != nil {
		fmt.Println("init db failed:", err.Error())
		log.Fatal("init db failed:", err.Error())
	}
	l, err := ctl.NewLocalData(db)
	if err != nil {
		fmt.Println("init localdata failed:", err.Error())
		log.Fatal("init localdata failed:", err.Error())
	}
	infos, err := l.ListAll()
	if err != nil {
		fmt.Println("localdata listAll failed:", err.Error())
		log.Fatal("localdata listAll failed:", err.Error())
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.Header([]string{"Exchange", "Symbol", "Binsize", "Start", "End"})

	for _, v := range infos {
		table.Append([]string{v.Exchange, v.Symbol, v.BinSize, v.Start.String(), v.End.String()})
	}
	table.Render()
}
