package common

import (
	"testing"
	"time"

	. "github.com/SuperGod/trademodel"
	"github.com/shopspring/decimal"
)

func calFee(fee decimal.Decimal, trades ...Trade) float64 {
	var cost decimal.Decimal
	for _, v := range trades {
		dec := decimal.NewFromFloat(v.Price).Mul(decimal.NewFromFloat(v.Amount))
		cost = cost.Add(dec)
	}
	realFee, _ := cost.Mul(fee).Float64()
	return realFee
}

func TestLong(t *testing.T) {
	tm := time.Now()
	openTrade := Trade{
		ID:     "1",
		Action: OpenLong,
		Time:   tm,
		Price:  100,
		Amount: 1,
	}
	closeTrade := Trade{
		ID:     "2",
		Action: CloseLong,
		Time:   tm.Add(time.Second),
		Price:  110,
		Amount: 1,
	}
	stopTrade := Trade{
		ID:     "3",
		Action: StopLong,
		Time:   tm.Add(time.Second * 2),
		Price:  90,
		Amount: 1,
	}

	b := NewVBalance()
	b.Set(1000)
	b.AddTrade(openTrade)
	b.AddTrade(closeTrade)
	fee := calFee(b.fee, openTrade, closeTrade)
	if b.Get() != 1010-fee {
		t.Fatal("balance close error:", b.Get(), 1010-fee)
	}
	b.Set(1000)
	b.AddTrade(openTrade)
	b.AddTrade(stopTrade)

	fee = calFee(b.fee, openTrade, stopTrade)
	if b.Get() != 990-fee {
		t.Fatal("balance stop error:", b.Get())
	}
}

func TestMultiLong(t *testing.T) {
	tm := time.Now()
	openTrade := Trade{
		ID:     "1",
		Action: OpenLong,
		Time:   tm,
		Price:  100,
		Amount: 1,
	}
	openTrade2 := Trade{
		ID:     "1",
		Action: OpenLong,
		Time:   tm,
		Price:  105,
		Amount: 1,
	}
	closeTrade := Trade{
		ID:     "2",
		Action: CloseLong,
		Time:   tm.Add(time.Second),
		Price:  110,
		Amount: 2,
	}
	stopTrade := Trade{
		ID:     "3",
		Action: StopLong,
		Time:   tm.Add(time.Second * 2),
		Price:  90,
		Amount: 2,
	}

	b := NewVBalance()
	b.Set(1000)
	b.AddTrade(openTrade)
	b.AddTrade(openTrade2)
	b.AddTrade(closeTrade)
	fee := calFee(b.fee, openTrade, openTrade2, closeTrade)
	if b.Get() != 1015-fee {
		t.Fatal("balance close error:", b.Get(), fee)
	}
	b.Set(1000)
	b.AddTrade(openTrade)
	b.AddTrade(openTrade2)
	b.AddTrade(stopTrade)
	fee = calFee(b.fee, openTrade, openTrade2, stopTrade)
	if b.Get() != 975-fee {
		t.Fatal("balance stop error:", b.Get())
	}
}

func TestShort(t *testing.T) {
	tm := time.Now()
	openTrade := Trade{
		ID:     "1",
		Action: OpenShort,
		Time:   tm,
		Price:  110,
		Amount: 1,
	}
	closeTrade := Trade{
		ID:     "2",
		Action: CloseShort,
		Time:   tm.Add(time.Second),
		Price:  100,
		Amount: 1,
	}
	stopTrade := Trade{
		ID:     "3",
		Action: StopShort,
		Time:   tm.Add(time.Second * 2),
		Price:  120,
		Amount: 1,
	}

	b := NewVBalance()
	b.Set(1000)
	b.AddTrade(openTrade)
	b.AddTrade(closeTrade)
	fee := calFee(b.fee, openTrade, closeTrade)
	if b.Get() != 1010-fee {
		t.Fatal("balance close error:", b.Get(), 1010-fee)
	}
	b.Set(1000)
	b.AddTrade(openTrade)
	b.AddTrade(stopTrade)
	fee = calFee(b.fee, openTrade, stopTrade)
	if b.Get() != 990-fee {
		t.Fatal("balance stop error:", b.Get())
	}
}

func TestMultiShort(t *testing.T) {
	tm := time.Now()
	openTrade := Trade{
		ID:     "1",
		Action: OpenShort,
		Time:   tm,
		Price:  110,
		Amount: 1,
	}
	openTrade2 := Trade{
		ID:     "1",
		Action: OpenShort,
		Time:   tm,
		Price:  120,
		Amount: 1,
	}
	closeTrade := Trade{
		ID:     "2",
		Action: CloseShort,
		Time:   tm.Add(time.Second),
		Price:  100,
		Amount: 2,
	}
	stopTrade := Trade{
		ID:     "3",
		Action: StopShort,
		Time:   tm.Add(time.Second * 2),
		Price:  130,
		Amount: 2,
	}

	b := NewVBalance()
	b.Set(1000)
	b.AddTrade(openTrade)
	b.AddTrade(openTrade2)
	b.AddTrade(closeTrade)
	fee := calFee(b.fee, openTrade, openTrade2, closeTrade)
	if b.Get() != 1030-fee {
		t.Fatal("balance close error:", b.Get())
	}
	b.Set(1000)
	b.AddTrade(openTrade)
	b.AddTrade(openTrade2)
	b.AddTrade(stopTrade)
	fee = calFee(b.fee, openTrade, openTrade2, stopTrade)
	if b.Get() != 970-fee {
		t.Fatal("balance stop error:", b.Get())
	}
}
