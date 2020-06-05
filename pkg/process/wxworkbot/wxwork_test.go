package wxworkbot

import (
	"testing"
	"time"

	. "github.com/SuperGod/trademodel"
)

func TestSendTrade(t *testing.T) {
	wx, err := NewWXWork(true)
	if err != nil {
		t.Fatal(err.Error())
	}
	wx.symbol = "XBTUSD"
	trade := Trade{
		ID:     "abcdefg",
		Action: OpenLong,
		Time:   time.Now(),
		Price:  10000.00,
		Amount: 100,
		Side:   "buy",
		Remark: "openlong test",
	}
	err = wx.sendTrade(trade)
	if err != nil {
		t.Fatal(err.Error())
	}

	trade = Trade{
		ID:     "abcdefg",
		Action: StopLong,
		Time:   time.Now(),
		Price:  10000.00,
		Amount: 101,
		Side:   "buy",
		Remark: "openlong test",
	}
	err = wx.sendTrade(trade)
	if err != nil {
		t.Fatal(err.Error())
	}
}

func TestSendPos(t *testing.T) {
	wx, err := NewWXWork(true)
	if err != nil {
		t.Fatal(err.Error())
	}
	wx.symbol = "XBTUSD"
	err = wx.sendPos(10)
	if err != nil {
		t.Fatal(err.Error())
	}
	err = wx.sendPos(-10)
	if err != nil {
		t.Fatal(err.Error())
	}
}
