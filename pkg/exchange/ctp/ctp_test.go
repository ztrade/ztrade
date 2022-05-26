package ctp

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/spf13/viper"
	. "github.com/ztrade/trademodel"
	. "github.com/ztrade/ztrade/pkg/core"
)

func TestOrder(t *testing.T) {
	cfgFile := "../../../dist/configs/ztrade.yaml"
	viper.SetConfigFile(cfgFile)
	viper.ReadInConfig()
	api, err := NewCtpExchange(viper.GetViper(), "ctp")
	if err != nil {
		t.Fatal()
	}
	var act TradeAction
	act.Action = OpenLong
	act.Price = 2000
	act.Amount = 1
	act.Symbol = "DCE.c2207"
	go func() {
		for {
			<-api.datas
		}
	}()
	ret, err := api.ProcessOrder(act)
	if err != nil {
		t.Fatal()
	}
	t.Log(ret)
	time.Sleep(time.Minute)
}

func TestMarketData(t *testing.T) {
	cfgFile := "../../../dist/configs/ztrade.yaml"
	viper.SetConfigFile(cfgFile)
	viper.ReadInConfig()
	api, err := NewCtpExchange(viper.GetViper(), "ctp")
	if err != nil {
		t.Fatal()
	}
	cp := CandleParam{Exchange: "ctp", Symbol: "DCE.c2207"}
	param := NewWatchCandle(&cp)
	api.Watch(*param)
	var buf []byte
	go func() {
		for data := range api.GetDataChan() {
			buf, _ = json.Marshal(data)
			fmt.Println(string(buf))
		}
	}()
	time.Sleep(time.Minute)
}

func TestSymbols(t *testing.T) {
	cfgFile := "../../../dist/configs/ztrade.yaml"
	viper.SetConfigFile(cfgFile)
	viper.ReadInConfig()
	time.Sleep(time.Second * 10)
	api, err := NewCtpExchange(viper.GetViper(), "ctp")
	if err != nil {
		t.Fatal(err.Error())
	}
	symbols, err := api.GetSymbols()
	if err != nil {
		t.Fatal(err.Error())
	}
	for _, v := range symbols {
		t.Log(v)
	}
}
