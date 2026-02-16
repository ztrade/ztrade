package vex

import (
	"testing"
	"time"

	. "github.com/ztrade/trademodel"
	. "github.com/ztrade/ztrade/pkg/core"
	. "github.com/ztrade/ztrade/pkg/event"
)

type cancelOnTradeProcesser struct {
	BaseProcesser
	orderID string
	traded  chan struct{}
}

func newCancelOnTradeProcesser(orderID string) *cancelOnTradeProcesser {
	p := &cancelOnTradeProcesser{orderID: orderID, traded: make(chan struct{}, 1)}
	p.Name = "cancelOnTrade"
	return p
}

func (p *cancelOnTradeProcesser) Init(bus *Bus) (err error) {
	p.BaseProcesser.Init(bus)
	p.Subscribe(EventTrade, p.onEventTrade)
	return
}

func (p *cancelOnTradeProcesser) onEventTrade(e *Event) error {
	select {
	case p.traded <- struct{}{}:
	default:
	}
	p.Send("cancel_order", EventOrder, &TradeAction{Action: CancelOne, ID: p.orderID})
	return nil
}

func TestSyncMode_NoDeadlock_WhenTradeHandlerCancelsOrder(t *testing.T) {
	sender := NewBaseProcesser("sender")
	ex := NewVExchange("BTCUSDT")
	canceler := newCancelOnTradeProcesser("order-1")

	ps := NewSyncProcessers()
	ps.Add(sender)
	ps.Add(ex)
	ps.Add(canceler)

	if err := ps.Start(); err != nil {
		t.Fatalf("failed to start processers: %v", err)
	}
	defer func() {
		_ = ps.Stop()
		ps.WaitClose(time.Second)
	}()

	sender.Send("balance_init", EventBalanceInit, &BalanceInfo{Balance: 100000, Fee: 0})
	sender.Send("order", EventOrder, &TradeAction{
		ID:     "order-1",
		Action: OpenLong,
		Amount: 1,
		Price:  100,
		Time:   time.Now(),
		Symbol: "BTCUSDT",
	})

	candle := &Candle{
		Start: time.Now().Unix(),
		Open:  100,
		High:  101,
		Low:   99,
		Close: 100,
	}

	done := make(chan struct{})
	go func() {
		sender.SendWithExtra("candle", EventCandle, candle, "1m")
		close(done)
	}()

	select {
	case <-done:
		// no deadlock
	case <-time.After(3 * time.Second):
		t.Fatal("DEADLOCK: processCandle holds orderMutex while sync trade callback sends EventOrder")
	}

	select {
	case <-canceler.traded:
	case <-time.After(time.Second):
		t.Fatal("expected trade callback to run")
	}
}
