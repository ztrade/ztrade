package define

import (
	"fmt"
	"reflect"
	"time"

	"github.com/SuperGod/coinex"
	. "github.com/SuperGod/trademodel"
)

// Events
const (
	EventCandleParam    = "candle_param"
	EventCandle         = "candle"
	EventOrder          = "order"
	EventOrderCancelAll = "order_cancel_all"
	// own trades
	EventTrade       = "trade"
	EventPosition    = "position"
	EventCurPosition = "cur_position" // position of current script
	EventRiskLimit   = "risk_limit"
	EventDepth       = "depth"
	// all trades in the markets
	EventTradeHistory = "trade_history"

	EventBalance     = "balance"
	EventBalanceInit = "balance_init"

	EventWatch = "watch"

	EventNotify = "notify"
)

var (
	EventTypes = map[string]reflect.Type{
		EventCandleParam: reflect.TypeOf(CandleParam{}),
		EventCandle:      reflect.TypeOf(Candle{}),
		EventOrder:       reflect.TypeOf(TradeAction{}),
		// EventOrderCancelAll     = "order_cancel_all"
		EventTrade:    reflect.TypeOf(Trade{}),
		EventPosition: reflect.TypeOf(coinex.Position{}),
		// EventCurPosition        = "cur_position" // position of current script
		// EventRiskLimit          = "risk_limit"
		EventDepth:        reflect.TypeOf(Depth{}),
		EventTradeHistory: reflect.TypeOf(Trade{}),
		EventBalance:      reflect.TypeOf(Balance{}),
		EventBalanceInit:  reflect.TypeOf(BalanceInfo{}),
		EventWatch:        reflect.TypeOf(WatchParam{}),

		EventNotify: reflect.TypeOf(NotifyEvent{}),
	}
)

// CandleParam get candle param
type CandleParam struct {
	Start    time.Time
	End      time.Time
	Exchange string
	BinSize  string
	Symbol   string
}

// NotifyEvent event to send notify
type NotifyEvent struct {
	Type    string // text,markdown
	Content string
}

// RiskLimit risk limit
type RiskLimit struct {
	Code         string  // symbol info, empty = global
	Lever        float64 // lever
	MaxLostRatio float64 // max lose ratio
}

// Key key of r
func (r RiskLimit) Key() string {
	return fmt.Sprintf("%s-%.2f", r.Code, r.Lever)
}

// WatchParam add watch event param
type WatchParam struct {
	Type  string
	Param map[string]interface{}
}

// BalanceInfo balance
type BalanceInfo struct {
	Balance float64
}
