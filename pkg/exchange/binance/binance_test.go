package binance

import (
	"fmt"
	"log"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/ztrade/trademodel"
	"github.com/ztrade/ztrade/pkg/core"
)

var (
	testClt *BinanceTrade
)

func getTestClt() *BinanceTrade {
	cfgFile := "../../../dist/configs/ztrade.yaml"
	cfg := viper.New()
	cfg.SetConfigFile(cfgFile)
	err := cfg.ReadInConfig()
	if err != nil {
		log.Fatal("ReadInConfig failed:" + err.Error())
	}
	testClt, err := NewBinanceTrader(cfg, "binance")
	if err != nil {
		log.Fatal("create client failed:" + err.Error())
	}
	return testClt
}

func TestMain(m *testing.M) {
	testClt = getTestClt()
	m.Run()
}

func TestKlineChan(t *testing.T) {
	end := time.Now()
	start := end.Add(0 - time.Hour)
	datas, errCh := testClt.GetKline("BTCUSDT", "1m", start, end)
	var n int
	for v := range datas {
		n++
		_ = v
		// t.Logf("%v", v)
		// fmt.Printf("%v\n", v)
	}
	err := <-errCh
	if err != nil {
		t.Fatal(err.Error())
	}
	t.Log("total:", n)

}

func TestProcessOrder(t *testing.T) {
	act := trademodel.TradeAction{
		Action: trademodel.OpenLong,
		Amount: 1,
		Price:  0.1,
		Symbol: "EOSUSDT",
		Time:   time.Now(),
	}
	ret, err := testClt.ProcessOrder(act)
	if err != nil {
		t.Fatal(err.Error())
	}
	t.Log(*ret)
}

func TestCancelAllOrders(t *testing.T) {
	orders, err := testClt.CancelAllOrders()
	if err != nil {
		t.Fatal(err.Error())
	}
	for _, v := range orders {
		t.Log(v)
	}
}

func TestWatchKline(t *testing.T) {
	symbol := core.SymbolInfo{Symbol: "BTCUSDT", Resolutions: "1m"}
	datas, stopC, err := testClt.WatchKline(symbol)
	go func() {
		<-time.After(time.Minute * 3)
		stopC <- struct{}{}
	}()
	var n int
	for v := range datas {
		n++
		_ = v
		// t.Logf("%v", v)
		fmt.Printf("%v\n", v)
	}
	if err != nil {
		t.Fatal(err.Error())
	}
	t.Log("total:", n)

}

func TestSymbols(t *testing.T) {
	symbols, err := testClt.GetSymbols()
	if err != nil {
		t.Fatal(err.Error())
	}
	for _, v := range symbols {
		t.Log(v.Symbol, v.Pricescale)
	}
}
