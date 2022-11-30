package binance

import (
	"log"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/ztrade/trademodel"
	"github.com/ztrade/ztrade/pkg/core"
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
		Price:  19000,
		Time:   time.Now(),
		Symbol: "BTCUSDT",
	}
	ret, err := testSpotClt.ProcessOrder(act)
	if err != nil {
		t.Fatal(err.Error())
	}
	t.Log(*ret)
}

func TestSpotProcessOrderStop(t *testing.T) {
	testSpotClt.GetSymbols()
	act := trademodel.TradeAction{
		Action: trademodel.StopLong,
		Amount: 0.001,
		Price:  20410.45,
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
	testClt.GetSymbols()
	orders, err := testSpotClt.CancelAllOrders()
	if err != nil {
		t.Fatal(err.Error())
	}
	for _, v := range orders {
		t.Log(v)
	}
}

func TestSpotBalance(t *testing.T) {
	err := testSpotClt.fetchBalance()
	if err != nil {
		t.Fatal(err.Error())
	}
	closeCh := make(chan int)
	go time.AfterFunc(time.Second*3, func() { close(closeCh) })
	dataCh := testSpotClt.GetDataChan()
	var v *core.ExchangeData
	for {
		select {
		case <-closeCh:
			return
		case v = <-dataCh:
			switch v.Data.Type {
			case core.EventBalance:
				bl := v.Data.Data.(*trademodel.Balance)
				t.Log("balance:", v.Name, v.Symbol, bl.Currency, bl.Available, bl.Balance, bl.Frozen)
			case core.EventPosition:
				pos := v.Data.Data.(*trademodel.Position)
				t.Log("pos:", v.Name, v.Symbol, pos.Symbol, pos.Hold, pos.Price)
			}
		}
	}
}
