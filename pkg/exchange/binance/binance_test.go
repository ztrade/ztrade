package binance

import (
	"fmt"
	"log"
	"os/user"
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
	u, err := user.Current()
	if err != nil {
		log.Fatal(err.Error())
	}
	cfgFile := u.HomeDir + "/.config/exchange.json"
	cfg := viper.New()
	cfg.SetConfigFile(cfgFile)
	err = cfg.ReadInConfig()
	if err != nil {
		log.Fatal("ReadInConfig failed:" + err.Error())
	}
	testClt, err := NewBinanceTrade(cfg, "binance")
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
	datas, errCh := testClt.KlineChan(start, end, "BTCUSDT", "1m")
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
		Price:  1,
		Time:   time.Now(),
	}
	testClt.symbol = "EOSUSDT"
	ret, err := testClt.ProcessOrder(act)
	if err != nil {
		t.Fatal(err.Error())
	}
	t.Log(*ret)
}

func TestCancelAllOrders(t *testing.T) {
	testClt.symbol = "EOSUSDT"
	orders, err := testClt.CancelAllOrders()
	if err != nil {
		t.Fatal(err.Error())
	}
	for _, v := range orders {
		t.Log(v)
	}
}

func TestWatchKline(t *testing.T) {
	symbol := core.SymbolInfo{Symbol: "EOSUSDT", Resolutions: "1m"}
	datas, stopC, err := testClt.WatchKline(symbol)
	go func() {
		<-time.After(time.Minute)
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
