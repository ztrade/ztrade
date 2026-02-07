package risk

import (
	"fmt"
	"math"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	. "github.com/ztrade/trademodel"
	. "github.com/ztrade/ztrade/pkg/core"
	. "github.com/ztrade/ztrade/pkg/event"
)

// RiskConfig configurable risk control parameters
type RiskConfig struct {
	// MaxPosition maximum allowed position size (0 = unlimited)
	MaxPosition float64
	// MaxDailyLoss maximum daily loss ratio (e.g. 0.1 = 10%, 0 = unlimited)
	MaxDailyLoss float64
	// MaxOrderRate maximum orders per minute (0 = unlimited)
	MaxOrderRate int
	// PriceDeviation max price deviation from last known price (e.g. 0.05 = 5%, 0 = disabled)
	PriceDeviation float64
}

// RiskManager monitors positions and P&L, triggers protective actions when limits are breached.
// It acts as a guardian/observer in the processer chain - it does NOT block orders, but when
// risk limits are breached it sends CancelAll + forced close orders to the exchange.
type RiskManager struct {
	BaseProcesser

	config RiskConfig
	symbol string

	// state tracking
	mu          sync.Mutex
	position    float64
	lastPrice   float64
	dailyPnL    float64
	dailyStart  time.Time
	balanceInit float64
	balance     float64
	orderTimes  []time.Time
	breached    bool // once breached, stop monitoring (already closed)
	lever       float64
	totalPnL    float64
	tradeCount  int
	rejectCount int
}

// NewRiskManager creates a new risk manager with the given config
func NewRiskManager(symbol string, config RiskConfig) *RiskManager {
	rm := new(RiskManager)
	rm.Name = "RiskManager"
	rm.symbol = symbol
	rm.config = config
	rm.dailyStart = time.Now().Truncate(24 * time.Hour)
	return rm
}

func (rm *RiskManager) Init(bus *Bus) (err error) {
	rm.BaseProcesser.Init(bus)
	rm.Subscribe(EventTrade, rm.onEventTrade)
	rm.Subscribe(EventPosition, rm.onEventPosition)
	rm.Subscribe(EventBalance, rm.onEventBalance)
	rm.Subscribe(EventBalanceInit, rm.onEventBalanceInit)
	rm.Subscribe(EventRiskLimit, rm.onEventRiskLimit)
	rm.Subscribe(EventOrder, rm.onEventOrder)
	return
}

func (rm *RiskManager) Start() (err error) {
	log.Infof("RiskManager started with config: maxPos=%.4f maxDailyLoss=%.2f%% maxOrderRate=%d/min priceDeviation=%.2f%%",
		rm.config.MaxPosition,
		rm.config.MaxDailyLoss*100,
		rm.config.MaxOrderRate,
		rm.config.PriceDeviation*100,
	)
	return
}

func (rm *RiskManager) Stop() (err error) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	log.Infof("RiskManager stats: trades=%d rejects=%d totalPnL=%.4f", rm.tradeCount, rm.rejectCount, rm.totalPnL)
	return
}

// onEventOrder monitors orders and checks rate limits + position size pre-check
func (rm *RiskManager) onEventOrder(e *Event) (err error) {
	act := e.GetData().(*TradeAction)
	if act == nil {
		return
	}
	// Skip cancel operations
	if act.Action == CancelAll || act.Action == CancelOne {
		return
	}

	var cancelReason string

	rm.mu.Lock()

	if rm.breached {
		rm.mu.Unlock()
		return
	}

	// Check order rate limit
	if rm.config.MaxOrderRate > 0 {
		now := time.Now()
		cutoff := now.Add(-time.Minute)
		// Clean old entries
		valid := rm.orderTimes[:0]
		for _, t := range rm.orderTimes {
			if t.After(cutoff) {
				valid = append(valid, t)
			}
		}
		rm.orderTimes = valid
		if len(rm.orderTimes) >= rm.config.MaxOrderRate {
			log.Warnf("RiskManager: order rate limit reached (%d/%d per minute), sending cancel all",
				len(rm.orderTimes), rm.config.MaxOrderRate)
			rm.rejectCount++
			cancelReason = "order rate limit exceeded"
			rm.mu.Unlock()
			rm.triggerCancel(cancelReason)
			return
		}
		rm.orderTimes = append(rm.orderTimes, now)
	}

	// Check position size pre-check (estimate new position after this order)
	if rm.config.MaxPosition > 0 && act.Action.IsOpen() {
		var estimatedPos float64
		if act.Action.IsLong() {
			estimatedPos = rm.position + act.Amount
		} else {
			estimatedPos = rm.position - act.Amount
		}
		if math.Abs(estimatedPos) > rm.config.MaxPosition {
			log.Warnf("RiskManager: position limit would be exceeded (current=%.4f, order=%.4f, limit=%.4f), sending cancel all",
				rm.position, act.Amount, rm.config.MaxPosition)
			rm.rejectCount++
			cancelReason = "position limit exceeded"
			rm.mu.Unlock()
			rm.triggerCancel(cancelReason)
			return
		}
	}

	// Check price deviation
	if rm.config.PriceDeviation > 0 && rm.lastPrice > 0 && act.Price > 0 {
		deviation := math.Abs(act.Price-rm.lastPrice) / rm.lastPrice
		if deviation > rm.config.PriceDeviation {
			log.Warnf("RiskManager: price deviation too large (price=%.4f, lastPrice=%.4f, deviation=%.2f%%, limit=%.2f%%)",
				act.Price, rm.lastPrice, deviation*100, rm.config.PriceDeviation*100)
			rm.rejectCount++
			cancelReason = "price deviation exceeded"
			rm.mu.Unlock()
			rm.triggerCancel(cancelReason)
			return
		}
	}

	rm.mu.Unlock()
	return
}

// onEventTrade monitors filled trades and tracks P&L
func (rm *RiskManager) onEventTrade(e *Event) (err error) {
	tr := e.GetData().(*Trade)
	if tr == nil {
		return
	}

	rm.mu.Lock()
	defer rm.mu.Unlock()

	rm.tradeCount++
	rm.lastPrice = tr.Price

	// Reset daily P&L at day boundary
	now := time.Now()
	today := now.Truncate(24 * time.Hour)
	if today.After(rm.dailyStart) {
		rm.dailyPnL = 0
		rm.dailyStart = today
		rm.breached = false // reset breach flag for new day
	}

	return
}

// onEventPosition monitors position changes and checks P&L limits
func (rm *RiskManager) onEventPosition(e *Event) (err error) {
	pos := e.GetData().(*Position)
	if pos == nil {
		return
	}

	rm.mu.Lock()
	defer rm.mu.Unlock()

	rm.position = pos.Hold
	rm.lastPrice = pos.Price

	return
}

// onEventBalance monitors balance and checks daily loss
func (rm *RiskManager) onEventBalance(e *Event) (err error) {
	balance := e.GetData().(*Balance)
	if balance == nil {
		return
	}

	rm.mu.Lock()

	oldBalance := rm.balance
	rm.balance = balance.Balance

	if rm.balanceInit > 0 && oldBalance > 0 {
		pnl := rm.balance - oldBalance
		rm.dailyPnL += pnl
		rm.totalPnL += pnl
	}

	var breachReason string
	var breachPos, breachPrice float64

	// Check daily loss limit
	if rm.config.MaxDailyLoss > 0 && rm.balanceInit > 0 && !rm.breached {
		dailyLossRatio := -rm.dailyPnL / rm.balanceInit
		if dailyLossRatio > rm.config.MaxDailyLoss {
			log.Warnf("RiskManager: daily loss limit breached (loss=%.4f, ratio=%.2f%%, limit=%.2f%%)",
				-rm.dailyPnL, dailyLossRatio*100, rm.config.MaxDailyLoss*100)
			rm.breached = true
			breachReason = "daily loss limit exceeded"
			breachPos = rm.position
			breachPrice = rm.lastPrice
		}
	}

	// Check max position after balance update
	if breachReason == "" && rm.config.MaxPosition > 0 && math.Abs(rm.position) > rm.config.MaxPosition && !rm.breached {
		log.Warnf("RiskManager: position limit breached (position=%.4f, limit=%.4f)",
			rm.position, rm.config.MaxPosition)
		rm.breached = true
		breachReason = "position limit exceeded"
		breachPos = rm.position
		breachPrice = rm.lastPrice
	}

	rm.mu.Unlock()

	if breachReason != "" {
		rm.triggerBreach(breachReason, breachPos, breachPrice)
	}
	return
}

func (rm *RiskManager) onEventBalanceInit(e *Event) (err error) {
	info := e.GetData().(*BalanceInfo)
	if info == nil {
		return
	}
	rm.mu.Lock()
	rm.balanceInit = info.Balance
	rm.balance = info.Balance
	rm.mu.Unlock()
	return
}

func (rm *RiskManager) onEventRiskLimit(e *Event) (err error) {
	info := e.GetData().(*RiskLimit)
	if info == nil {
		return
	}
	rm.mu.Lock()
	rm.lever = info.Lever
	rm.mu.Unlock()
	return
}

// triggerCancel sends a CancelAll order (does not close position).
// Must be called WITHOUT holding rm.mu to avoid deadlock in sync mode.
func (rm *RiskManager) triggerCancel(reason string) {
	rm.Send(EventOrder, EventOrder, &TradeAction{Action: CancelAll})
	rm.Send("risk", EventNotify, &NotifyEvent{
		Title:   "Risk Control",
		Type:    "text",
		Content: fmt.Sprintf("Orders cancelled: %s", reason),
	})
}

// triggerBreach sends CancelAll + force close position.
// Must be called WITHOUT holding rm.mu to avoid deadlock in sync mode.
func (rm *RiskManager) triggerBreach(reason string, position, lastPrice float64) {
	// Cancel all pending orders
	rm.Send(EventOrder, EventOrder, &TradeAction{Action: CancelAll})
	// Force close position
	if position > 0 {
		rm.Send(EventOrder, EventOrder, &TradeAction{
			ID:     "risk-close",
			Symbol: rm.symbol,
			Action: CloseLong,
			Amount: math.Abs(position),
			Price:  lastPrice,
			Time:   time.Now(),
		})
	} else if position < 0 {
		rm.Send(EventOrder, EventOrder, &TradeAction{
			ID:     "risk-close",
			Symbol: rm.symbol,
			Action: CloseShort,
			Amount: math.Abs(position),
			Price:  lastPrice,
			Time:   time.Now(),
		})
	}
	rm.Send("risk", EventNotify, &NotifyEvent{
		Title:   "Risk Control - BREACH",
		Type:    "text",
		Content: fmt.Sprintf("Risk limit breached: %s. All orders cancelled, position force closed.", reason),
	})
}
