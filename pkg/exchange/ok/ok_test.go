package ok

import (
	"fmt"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/ztrade/trademodel"
	"github.com/ztrade/ztrade/pkg/core"
)

var (
	testClt *OkexTrader
)

func getTestClt() *OkexTrader {
	cfgFile := "../../../dist/configs/ztrade.yaml"
	cfg := viper.New()
	cfg.SetConfigFile(cfgFile)
	err := cfg.ReadInConfig()
	if err != nil {
		log.Fatal("ReadInConfig failed:" + err.Error())
	}
	testClt, err = NewOkexTrader(cfg, "okex")
	if err != nil {
		log.Fatal("create client failed:" + err.Error())
	}
	return testClt
}

func TestMain(m *testing.M) {
	testClt = getTestClt()
	m.Run()
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

func TestOrder(t *testing.T) {
	order, err := testClt.ProcessOrder(trademodel.TradeAction{
		Action: trademodel.OpenShort,
		Amount: 1,
		Price:  3100,
		Time:   time.Now(),
	})
	if err != nil {
		t.Fatal(err.Error())
	}
	t.Log(*order)

	order, err = testClt.ProcessOrder(trademodel.TradeAction{
		Action: trademodel.StopShort,
		Amount: 1,
		Price:  3080,
		Time:   time.Now(),
	})
	if err != nil {
		t.Fatal(err.Error())
	}
	t.Log(*order)
	time.Sleep(time.Second)
	_, err = testClt.CancelAllOrders()
	if err != nil {
		t.Fatal(err.Error())
	}
}

func TestCancelAllOrder(t *testing.T) {
	_, err := testClt.CancelAllOrders()
	if err != nil {
		t.Fatal(err.Error())
	}
}

func TestCancelOrder(t *testing.T) {
	act := trademodel.TradeAction{
		Action: trademodel.OpenLong,
		Amount: 1,
		Price:  2000,
		Time:   time.Now(),
	}
	order, err := testClt.ProcessOrder(act)
	if err != nil {
		t.Fatal(err.Error())
	}
	t.Log(*order)
	time.Sleep(time.Second * 5)
	ret, err := testClt.CancelOrder(order)
	if err != nil {
		t.Fatal(err.Error())
	}
	t.Log("ret:", *ret)
}

func TestCancelStopOrder(t *testing.T) {
	act := trademodel.TradeAction{
		Action: trademodel.StopLong,
		Amount: 1,
		Price:  2000,
		Time:   time.Now(),
	}
	order, err := testClt.ProcessOrder(act)
	if err != nil {
		t.Fatal(err.Error())
	}
	t.Log(*order)
	time.Sleep(time.Second * 5)
	ret, err := testClt.CancelOrder(order)
	if err != nil {
		t.Fatal(err.Error())
	}
	t.Log("ret:", *ret)
}

func TestDepth(t *testing.T) {
	param := core.WatchParam{Type: core.EventDepth, Extra: "ETH-USDT-SWAP", Data: map[string]interface{}{"name": "depth"}}
	testClt.Watch(param)
	ch := testClt.GetDataChan()
	for v := range ch {
		fmt.Println(v)
	}
}
