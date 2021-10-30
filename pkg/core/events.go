package core

import (
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	. "github.com/ztrade/trademodel"
)

// Events
const (
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
	EventTradeMarket = "trade_market"

	EventBalance     = "balance"
	EventBalanceInit = "balance_init"

	EventWatch = "watch"

	EventNotify = "notify"
)

var (
	EventTypes = map[string]reflect.Type{
		EventCandle: reflect.TypeOf(Candle{}),
		EventOrder:  reflect.TypeOf(TradeAction{}),
		// EventOrderCancelAll     = "order_cancel_all"
		EventTrade:    reflect.TypeOf(Trade{}),
		EventPosition: reflect.TypeOf(Position{}),
		// EventCurPosition        = "cur_position" // position of current script
		// EventRiskLimit          = "risk_limit"
		EventDepth:       reflect.TypeOf(Depth{}),
		EventTradeMarket: reflect.TypeOf(Trade{}),
		EventBalance:     reflect.TypeOf(Balance{}),
		EventBalanceInit: reflect.TypeOf(BalanceInfo{}),
		EventWatch:       reflect.TypeOf(WatchParam{}),
		EventNotify:      reflect.TypeOf(NotifyEvent{}),
	}
)

type Initer interface {
	Init() error
}

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
	Param interface{}
}

func (wp *WatchParam) Init() (err error) {
	if wp.Type != EventCandle {
		return
	}
	var buf []byte
	buf, err = json.Marshal(wp.Param)
	if err != nil {
		return
	}
	p := &CandleParam{}
	err = json.Unmarshal(buf, p)
	if err != nil {
		return
	}
	wp.Param = p
	return
}

func NewWatchCandle(cp *CandleParam) *WatchParam {
	wp := &WatchParam{Type: EventCandle,
		Param: cp}
	return wp
}

// BalanceInfo balance
type BalanceInfo struct {
	Balance float64
}
