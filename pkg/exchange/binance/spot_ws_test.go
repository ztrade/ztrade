package binance

import (
	"testing"

	"github.com/ztrade/trademodel"
	"github.com/ztrade/ztrade/pkg/core"
)

func TestSpotDepthWS(t *testing.T) {
	data := testSpotClt.datas
	err := testSpotClt.Watch(core.WatchParam{Type: core.EventDepth, Extra: "BTCUSDT"})
	if err != nil {
		t.Fatal(err.Error())
	}

	for v := range data {
		t.Logf("%s %v", v.Symbol, v.Data.Data.(*trademodel.Depth))
	}
}

func TestSpotTradeWS(t *testing.T) {
	data := testSpotClt.datas
	err := testSpotClt.Watch(core.WatchParam{Type: core.EventTradeMarket, Extra: "BTCUSDT"})
	if err != nil {
		t.Fatal(err.Error())
	}

	for v := range data {
		t.Logf("%s %v", v.Symbol, v.Data.Data.(*trademodel.Trade))
	}
}
