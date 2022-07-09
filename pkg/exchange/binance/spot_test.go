package binance

import (
	"log"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/ztrade/trademodel"
)

var (
	testSpotClt *BinanceSpot
)

func getTestSpotClt() *BinanceSpot {
	cfgFile := "../../../dist/configs/ztrade.yaml"
	cfg := viper.New()
	cfg.SetConfigFile(cfgFile)
	err := cfg.ReadInConfig()
	if err != nil {
		log.Fatal("ReadInConfig failed:" + err.Error())
	}
	testSpotClt, err = NewBinanceSpotEx(cfg, "binance_spot")
	if err != nil {
		log.Fatal("create client failed:" + err.Error())
	}
	return testSpotClt
}

func TestSpotProcessOrder(t *testing.T) {
	act := trademodel.TradeAction{
		Action: trademodel.OpenLong,
		Amount: 0.01,
		Price:  20000,
		Time:   time.Now(),
		Symbol: "BTCUSDT",
	}
	ret, err := testSpotClt.ProcessOrder(act)
	if err != nil {
		t.Fatal(err.Error())
	}
	t.Log(*ret)
}

func TestSpotCancelAllOrders(t *testing.T) {
	orders, err := testSpotClt.CancelAllOrders()
	if err != nil {
		t.Fatal(err.Error())
	}
	for _, v := range orders {
		t.Log(v)
	}
}
