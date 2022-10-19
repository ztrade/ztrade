package gateio

import (
	"testing"

	"github.com/ztrade/trademodel"
	"github.com/ztrade/ztrade/pkg/core"
)

func TestSpotDepthWS(t *testing.T) {
	data := testClt.datas
	err := testClt.Watch(core.WatchParam{Type: core.EventDepth, Extra: "BTC_USDT"})
	if err != nil {
		t.Fatal(err.Error())
	}

	for v := range data {
		t.Logf("%s %v", v.Symbol, v.Data.Data.(*trademodel.Depth))
	}
}

func TestSpotTradeWS(t *testing.T) {
	data := testClt.datas
	err := testClt.Watch(core.WatchParam{Type: core.EventTradeMarket, Extra: "BTC_USDT"})
	if err != nil {
		t.Fatal(err.Error())
	}

	for v := range data {
		t.Logf("%s %v", v.Symbol, v.Data.Data.(*trademodel.Trade))
	}
}
