package goscript

import (
	"testing"

	. "github.com/SuperGod/trademodel"
)

func TestRunner(t *testing.T) {
	r, err := NewRunner("./strategy.go")
	if err != nil {
		t.Fatal(err.Error())
	}
	param, err := r.Param()
	if err != nil {
		t.Fatal(err.Error())
	}
	t.Log("param:", param)
	err = r.OnCandle(Candle{Open: 10, Close: 20, Low: 1, High: 100})
	if err != nil {
		t.Fatal(err.Error())
	}
	err = r.OnPosition(16)
	if err != nil {
		t.Fatal(err.Error())
	}
	err = r.OnTrade(Trade{ID: "trade"})
	if err != nil {
		t.Fatal(err.Error())
	}
	err = r.OnTradeHistory(Trade{ID: "tradehistory"})
	if err != nil {
		t.Fatal(err.Error())
	}
	err = r.OnDepth(Depth{Buys: []DepthInfo{DepthInfo{Price: 10, Amount: 10}}, Sells: []DepthInfo{DepthInfo{Price: 11, Amount: 10}}})
	if err != nil {
		t.Fatal(err.Error())
	}
}
