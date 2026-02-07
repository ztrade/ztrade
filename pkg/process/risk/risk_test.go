package risk

import (
	"sync"
	"testing"
	"time"

	. "github.com/ztrade/trademodel"
	. "github.com/ztrade/ztrade/pkg/core"
	. "github.com/ztrade/ztrade/pkg/event"
)

// testCollector is a minimal processer that collects events sent by RiskManager
type testCollector struct {
	BaseProcesser
	mu      sync.Mutex
	orders  []*TradeAction
	notifys []*NotifyEvent
}

func newTestCollector() *testCollector {
	c := &testCollector{}
	c.Name = "collector"
	return c
}

func (c *testCollector) Init(bus *Bus) (err error) {
	c.BaseProcesser.Init(bus)
	c.Subscribe(EventOrder, c.onOrder)
	c.Subscribe(EventNotify, c.onNotify)
	return
}

func (c *testCollector) onOrder(e *Event) error {
	act := e.GetData().(*TradeAction)
	if act != nil {
		c.mu.Lock()
		c.orders = append(c.orders, act)
		c.mu.Unlock()
	}
	return nil
}

func (c *testCollector) onNotify(e *Event) error {
	n := e.GetData().(*NotifyEvent)
	if n != nil {
		c.mu.Lock()
		c.notifys = append(c.notifys, n)
		c.mu.Unlock()
	}
	return nil
}

// setupSyncPipeline creates a sync pipeline: sender -> RiskManager -> collector
func setupSyncPipeline(config RiskConfig) (*BaseProcesser, *RiskManager, *testCollector, *Processers) {
	sender := NewBaseProcesser("sender")
	rm := NewRiskManager("BTCUSDT", config)
	collector := newTestCollector()

	p := NewSyncProcessers()
	p.Add(sender)
	p.Add(rm)
	p.Add(collector)

	return sender, rm, collector, p
}

func mustStart(t *testing.T, p *Processers) {
	t.Helper()
	if err := p.Start(); err != nil {
		t.Fatalf("failed to start processers: %v", err)
	}
}

func sendBalanceInit(sender *BaseProcesser, balance float64) {
	sender.Send("balance_init", EventBalanceInit, &BalanceInfo{Balance: balance})
}

func sendBalance(sender *BaseProcesser, balance float64) {
	sender.Send("balance", EventBalance, &Balance{Balance: balance})
}

func sendPosition(sender *BaseProcesser, hold, price float64) {
	sender.Send("position", EventPosition, &Position{Hold: hold, Price: price})
}

func sendOrder(sender *BaseProcesser, action TradeType, amount, price float64) {
	sender.Send("order", EventOrder, &TradeAction{
		Action: action,
		Amount: amount,
		Price:  price,
		Time:   time.Now(),
	})
}

// --- Tests ---

func TestNewRiskManager(t *testing.T) {
	cfg := RiskConfig{
		MaxPosition:    10.0,
		MaxDailyLoss:   0.1,
		MaxOrderRate:   60,
		PriceDeviation: 0.05,
	}
	rm := NewRiskManager("BTCUSDT", cfg)
	if rm.Name != "RiskManager" {
		t.Errorf("expected name RiskManager, got %s", rm.Name)
	}
	if rm.symbol != "BTCUSDT" {
		t.Errorf("expected symbol BTCUSDT, got %s", rm.symbol)
	}
	if rm.config.MaxPosition != 10.0 {
		t.Errorf("expected MaxPosition 10.0, got %f", rm.config.MaxPosition)
	}
	if rm.breached {
		t.Error("expected breached=false on new RiskManager")
	}
}

func TestSyncMode_NoDeadlock_OrderRateLimit(t *testing.T) {
	// This test verifies that triggerCancel does NOT deadlock in sync mode.
	// In sync mode, rm.Send(EventOrder, ...) will synchronously re-enter onEventOrder.
	cfg := RiskConfig{MaxOrderRate: 2}
	sender, _, collector, p := setupSyncPipeline(cfg)
	mustStart(t, p)

	sendOrder(sender, OpenLong, 1.0, 100.0)
	sendOrder(sender, OpenLong, 1.0, 100.0)

	done := make(chan struct{})
	go func() {
		sendOrder(sender, OpenLong, 1.0, 100.0) // triggers rate limit
		close(done)
	}()

	select {
	case <-done:
		// No deadlock
	case <-time.After(3 * time.Second):
		t.Fatal("DEADLOCK: onEventOrder -> triggerCancel -> Send(EventOrder) -> onEventOrder re-entry")
	}

	hasCancelAll := false
	for _, o := range collector.orders {
		if o.Action == CancelAll {
			hasCancelAll = true
			break
		}
	}
	if !hasCancelAll {
		t.Error("expected CancelAll order after rate limit")
	}
}

func TestSyncMode_NoDeadlock_DailyLossBreach(t *testing.T) {
	// triggerBreach sends EventOrder which re-enters onEventOrder in sync mode.
	cfg := RiskConfig{MaxDailyLoss: 0.05}
	sender, _, collector, p := setupSyncPipeline(cfg)
	mustStart(t, p)

	sendBalanceInit(sender, 100000.0)
	sendPosition(sender, 5.0, 50000.0)

	done := make(chan struct{})
	go func() {
		sendBalance(sender, 94000.0) // lose 6% > 5% limit
		close(done)
	}()

	select {
	case <-done:
		// No deadlock
	case <-time.After(3 * time.Second):
		t.Fatal("DEADLOCK: onEventBalance -> triggerBreach -> Send(EventOrder) -> onEventOrder re-entry")
	}

	hasCancelAll := false
	hasClose := false
	for _, o := range collector.orders {
		if o.Action == CancelAll {
			hasCancelAll = true
		}
		if o.Action == CloseLong {
			hasClose = true
		}
	}
	if !hasCancelAll {
		t.Error("expected CancelAll after breach")
	}
	if !hasClose {
		t.Error("expected CloseLong after breach with positive position")
	}
}

func TestOrderRateLimit(t *testing.T) {
	cfg := RiskConfig{MaxOrderRate: 3}
	sender, rm, collector, p := setupSyncPipeline(cfg)
	mustStart(t, p)

	sendOrder(sender, OpenLong, 1.0, 100.0)
	sendOrder(sender, OpenLong, 1.0, 100.0)
	sendOrder(sender, OpenLong, 1.0, 100.0)

	if rm.rejectCount != 0 {
		t.Errorf("expected 0 rejects after 3 orders (limit=3), got %d", rm.rejectCount)
	}

	sendOrder(sender, OpenLong, 1.0, 100.0)
	if rm.rejectCount != 1 {
		t.Errorf("expected 1 reject after 4th order, got %d", rm.rejectCount)
	}

	hasCancelAll := false
	for _, o := range collector.orders {
		if o.Action == CancelAll {
			hasCancelAll = true
			break
		}
	}
	if !hasCancelAll {
		t.Error("expected CancelAll after rate limit exceeded")
	}
}

func TestPositionLimitPreCheck(t *testing.T) {
	cfg := RiskConfig{MaxPosition: 5.0}
	sender, rm, collector, p := setupSyncPipeline(cfg)
	mustStart(t, p)

	sendPosition(sender, 4.0, 100.0)
	sendOrder(sender, OpenLong, 2.0, 100.0) // 4+2=6 > 5

	if rm.rejectCount != 1 {
		t.Errorf("expected 1 reject for exceeding position limit, got %d", rm.rejectCount)
	}

	hasCancelAll := false
	for _, o := range collector.orders {
		if o.Action == CancelAll {
			hasCancelAll = true
			break
		}
	}
	if !hasCancelAll {
		t.Error("expected CancelAll after position limit pre-check")
	}
}

func TestPositionLimitAllowsWithinLimit(t *testing.T) {
	cfg := RiskConfig{MaxPosition: 5.0}
	sender, rm, _, p := setupSyncPipeline(cfg)
	mustStart(t, p)

	sendPosition(sender, 3.0, 100.0)
	sendOrder(sender, OpenLong, 1.5, 100.0) // 3+1.5=4.5 <= 5

	if rm.rejectCount != 0 {
		t.Errorf("expected 0 rejects for within-limit order, got %d", rm.rejectCount)
	}
}

func TestPriceDeviation(t *testing.T) {
	cfg := RiskConfig{PriceDeviation: 0.05}
	sender, rm, collector, p := setupSyncPipeline(cfg)
	mustStart(t, p)

	sender.Send("trade", EventTrade, &Trade{Price: 100.0})
	sendOrder(sender, OpenLong, 1.0, 106.1) // 6.1% > 5%

	if rm.rejectCount != 1 {
		t.Errorf("expected 1 reject for price deviation, got %d", rm.rejectCount)
	}

	hasCancelAll := false
	for _, o := range collector.orders {
		if o.Action == CancelAll {
			hasCancelAll = true
			break
		}
	}
	if !hasCancelAll {
		t.Error("expected CancelAll after price deviation exceeded")
	}
}

func TestPriceDeviationWithinLimit(t *testing.T) {
	cfg := RiskConfig{PriceDeviation: 0.05}
	sender, rm, _, p := setupSyncPipeline(cfg)
	mustStart(t, p)

	sender.Send("trade", EventTrade, &Trade{Price: 100.0})
	sendOrder(sender, OpenLong, 1.0, 103.0) // 3% < 5%

	if rm.rejectCount != 0 {
		t.Errorf("expected 0 rejects for within-deviation order, got %d", rm.rejectCount)
	}
}

func TestDailyLossBreach(t *testing.T) {
	cfg := RiskConfig{MaxDailyLoss: 0.1}
	sender, rm, collector, p := setupSyncPipeline(cfg)
	mustStart(t, p)

	sendBalanceInit(sender, 100000.0)
	sendPosition(sender, 2.0, 50000.0)
	sendBalance(sender, 89000.0) // lose 11% > 10%

	if !rm.breached {
		t.Error("expected breached=true after 11% loss (limit 10%)")
	}

	hasCancelAll := false
	hasClose := false
	for _, o := range collector.orders {
		if o.Action == CancelAll {
			hasCancelAll = true
		}
		if o.Action == CloseLong {
			hasClose = true
		}
	}
	if !hasCancelAll {
		t.Error("expected CancelAll after daily loss breach")
	}
	if !hasClose {
		t.Error("expected CloseLong after daily loss breach")
	}
	// Verify close order carries the correct Symbol
	for _, o := range collector.orders {
		if o.Action == CloseLong {
			if o.Symbol != "BTCUSDT" {
				t.Errorf("expected close order Symbol=BTCUSDT, got %q", o.Symbol)
			}
		}
	}
	if len(collector.notifys) == 0 {
		t.Error("expected notification after breach")
	}
}

func TestDailyLossBreachShortPosition(t *testing.T) {
	cfg := RiskConfig{MaxDailyLoss: 0.05}
	sender, rm, collector, p := setupSyncPipeline(cfg)
	mustStart(t, p)

	sendBalanceInit(sender, 100000.0)
	sendPosition(sender, -3.0, 50000.0)
	sendBalance(sender, 94000.0) // lose 6% > 5%

	if !rm.breached {
		t.Error("expected breached=true")
	}

	hasCloseShort := false
	for _, o := range collector.orders {
		if o.Action == CloseShort {
			hasCloseShort = true
		}
	}
	if !hasCloseShort {
		t.Error("expected CloseShort after breach with negative position")
	}
}

func TestNoBreachWhenWithinLimit(t *testing.T) {
	cfg := RiskConfig{MaxDailyLoss: 0.1}
	sender, rm, _, p := setupSyncPipeline(cfg)
	mustStart(t, p)

	sendBalanceInit(sender, 100000.0)
	sendBalance(sender, 95000.0) // lose 5% < 10%

	if rm.breached {
		t.Error("expected breached=false after 5% loss (limit 10%)")
	}
}

func TestCancelAllSkipped(t *testing.T) {
	cfg := RiskConfig{MaxOrderRate: 1}
	sender, rm, _, p := setupSyncPipeline(cfg)
	mustStart(t, p)

	sendOrder(sender, OpenLong, 1.0, 100.0)
	beforeRejects := rm.rejectCount

	sender.Send("cancel", EventOrder, &TradeAction{Action: CancelAll})

	if rm.rejectCount != beforeRejects {
		t.Error("CancelAll should not trigger rate limit check")
	}
}

func TestBreachedStopsMonitoring(t *testing.T) {
	cfg := RiskConfig{MaxDailyLoss: 0.05, MaxPosition: 100}
	sender, rm, _, p := setupSyncPipeline(cfg)
	mustStart(t, p)

	sendBalanceInit(sender, 100000.0)
	sendPosition(sender, 2.0, 50000.0)
	sendBalance(sender, 94000.0) // trigger breach

	if !rm.breached {
		t.Fatal("expected breach")
	}

	rejectsBefore := rm.rejectCount
	sendOrder(sender, OpenLong, 1.0, 100.0)

	if rm.rejectCount != rejectsBefore {
		t.Error("orders after breach should be silently skipped")
	}
}

func TestTradeUpdatesPriceAndCount(t *testing.T) {
	cfg := RiskConfig{}
	sender, rm, _, p := setupSyncPipeline(cfg)
	mustStart(t, p)

	sender.Send("trade", EventTrade, &Trade{Price: 42000.0})

	rm.mu.Lock()
	defer rm.mu.Unlock()
	if rm.lastPrice != 42000.0 {
		t.Errorf("expected lastPrice=42000, got %f", rm.lastPrice)
	}
	if rm.tradeCount != 1 {
		t.Errorf("expected tradeCount=1, got %d", rm.tradeCount)
	}
}

func TestBalanceInitSetsState(t *testing.T) {
	cfg := RiskConfig{}
	sender, rm, _, p := setupSyncPipeline(cfg)
	mustStart(t, p)

	sendBalanceInit(sender, 50000.0)

	rm.mu.Lock()
	defer rm.mu.Unlock()
	if rm.balanceInit != 50000.0 {
		t.Errorf("expected balanceInit=50000, got %f", rm.balanceInit)
	}
	if rm.balance != 50000.0 {
		t.Errorf("expected balance=50000, got %f", rm.balance)
	}
}

func TestRiskLimitSetsLever(t *testing.T) {
	cfg := RiskConfig{}
	sender, rm, _, p := setupSyncPipeline(cfg)
	mustStart(t, p)

	sender.Send("risk", EventRiskLimit, &RiskLimit{Lever: 10.0})

	rm.mu.Lock()
	defer rm.mu.Unlock()
	if rm.lever != 10.0 {
		t.Errorf("expected lever=10, got %f", rm.lever)
	}
}

func TestPositionBreachOnBalanceUpdate(t *testing.T) {
	cfg := RiskConfig{MaxPosition: 5.0}
	sender, rm, collector, p := setupSyncPipeline(cfg)
	mustStart(t, p)

	sendBalanceInit(sender, 100000.0)
	sendPosition(sender, 8.0, 100.0)
	sendBalance(sender, 100000.0) // triggers position check

	if !rm.breached {
		t.Error("expected breach when position exceeds limit on balance update")
	}

	hasCancelAll := false
	for _, o := range collector.orders {
		if o.Action == CancelAll {
			hasCancelAll = true
			break
		}
	}
	if !hasCancelAll {
		t.Error("expected CancelAll after position breach on balance update")
	}
}

func TestConcurrentSafety(t *testing.T) {
	// Test that RiskManager's internal state is safe under concurrent access.
	// Uses sync mode to avoid the pre-existing race in Bus.lastEventTime (async mode).
	cfg := RiskConfig{
		MaxPosition:    100.0,
		MaxDailyLoss:   0.5,
		MaxOrderRate:   1000,
		PriceDeviation: 0.5,
	}
	sender, rm, _, p := setupSyncPipeline(cfg)
	mustStart(t, p)

	sendBalanceInit(sender, 100000.0)
	sender.Send("trade", EventTrade, &Trade{Price: 100.0})

	// In sync mode, events are processed sequentially, but we can still test
	// that multiple types of events don't corrupt internal state.
	for i := 0; i < 100; i++ {
		sendOrder(sender, OpenLong, 0.1, 100.0+float64(i)*0.01)
		sendBalance(sender, 100000.0-float64(i)*10)
		sendPosition(sender, float64(i)*0.1, 100.0)
		sender.Send("trade", EventTrade, &Trade{Price: 100.0 + float64(i)*0.01})
	}

	rm.mu.Lock()
	defer rm.mu.Unlock()
	if rm.tradeCount != 101 { // 1 initial + 100 in loop
		t.Errorf("expected 101 trades, got %d", rm.tradeCount)
	}
}

func TestStopLogsStats(t *testing.T) {
	cfg := RiskConfig{MaxOrderRate: 1}
	sender, rm, _, p := setupSyncPipeline(cfg)
	mustStart(t, p)

	sendOrder(sender, OpenLong, 1.0, 100.0)
	sendOrder(sender, OpenLong, 1.0, 100.0) // triggers reject

	if rm.rejectCount != 1 {
		t.Fatalf("expected 1 reject, got %d", rm.rejectCount)
	}

	err := rm.Stop()
	if err != nil {
		t.Errorf("Stop() returned error: %v", err)
	}
}

func TestCloseOrderNotCheckedByPositionLimit(t *testing.T) {
	cfg := RiskConfig{MaxPosition: 5.0}
	sender, rm, _, p := setupSyncPipeline(cfg)
	mustStart(t, p)

	sendPosition(sender, 4.0, 100.0)
	sendOrder(sender, CloseLong, 10.0, 100.0) // close is not Open, skip pre-check

	if rm.rejectCount != 0 {
		t.Errorf("close orders should not trigger position limit, got %d rejects", rm.rejectCount)
	}
}

func TestZeroConfigDisablesAllChecks(t *testing.T) {
	cfg := RiskConfig{}
	sender, rm, _, p := setupSyncPipeline(cfg)
	mustStart(t, p)

	sendBalanceInit(sender, 100000.0)
	sender.Send("trade", EventTrade, &Trade{Price: 100.0})

	for i := 0; i < 100; i++ {
		sendOrder(sender, OpenLong, 1000.0, 999.0)
	}

	if rm.rejectCount != 0 {
		t.Errorf("zero config should disable all checks, got %d rejects", rm.rejectCount)
	}
	if rm.breached {
		t.Error("zero config should never breach")
	}
}
