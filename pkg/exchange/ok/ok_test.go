package ok

import (
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/ztrade/trademodel"
)

func TestOrder(t *testing.T) {
	viper.SetConfigFile("../../../dist/configs/ztrade.yaml")
	err := viper.ReadInConfig()
	if err != nil {
		t.Fatal(err.Error())
	}
	api, err := NewOkexExchange(viper.GetViper(), "okex", "ETH-USDT-SWAP")
	if err != nil {
		t.Fatal(err.Error())
	}
	order, err := api.ProcessOrder(trademodel.TradeAction{
		Action: trademodel.OpenShort,
		Amount: 1,
		Price:  3100,
		Time:   time.Now(),
	})
	if err != nil {
		t.Fatal(err.Error())
	}
	t.Log(*order)

	order, err = api.ProcessOrder(trademodel.TradeAction{
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
	_, err = api.CancelAllOrders()
	if err != nil {
		t.Fatal(err.Error())
	}
}
