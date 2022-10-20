package gateio

import (
	"log"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/ztrade/trademodel"
)

var (
	testClt *GateIO
)

func TestMain(m *testing.M) {
	testClt = getTestSpotClt()
	m.Run()
}

func getTestSpotClt() *GateIO {
	cfgFile := "../../../dist/configs/ztrade.yaml"
	cfg := viper.New()
	cfg.SetConfigFile(cfgFile)
	err := cfg.ReadInConfig()
	if err != nil {
		log.Fatal("ReadInConfig failed:" + err.Error())
	}
	testClt, err = NewGateIO(cfg, "gateio")
	if err != nil {
		log.Fatal("create client failed:" + err.Error())
	}
	return testClt
}

func TestOrderLong(t *testing.T) {
	order, err := testClt.ProcessOrder(trademodel.TradeAction{
		Symbol: "EOS_USDT",
		Price:  1,
		Amount: 1,
		Action: trademodel.OpenLong,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Log(order)
	ret, err := testClt.CancelOrder(order)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(ret)
}

func TestOrderShort(t *testing.T) {
	order, err := testClt.ProcessOrder(trademodel.TradeAction{
		Symbol: "BTC_USD",
		Price:  19100,
		Amount: 1,
		Action: trademodel.OpenShort,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Log(order)
	ret, err := testClt.CancelOrder(order)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(ret)
}

func TestOrderClose(t *testing.T) {
	order, err := testClt.ProcessOrder(trademodel.TradeAction{
		Symbol: "EOS_USDT",
		Price:  1.033,
		Amount: 1,
		Action: trademodel.CloseShort,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Log(order)
}

func TestOrderStop(t *testing.T) {
	order, err := testClt.ProcessOrder(trademodel.TradeAction{
		Symbol: "EOS_USDT",
		Price:  1.1,
		Amount: 1,
		Action: trademodel.StopShort,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Log(order)

	ret, err := testClt.CancelOrder(order)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(ret)
}

func TestGetAllSymbols(t *testing.T) {
	symbols, err := testClt.GetSymbols()
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range symbols {
		t.Log(v.Symbol)
	}
}

func TestKline(t *testing.T) {
	end := time.Now()
	start := end.Add(-time.Hour)
	data, errCh := testClt.GetKline("BTC_USDT", "1m", start, end)
	for v := range data {
		t.Log(v)
	}
	err := <-errCh
	if err != nil {
		t.Fatal(err.Error())
	}
}
