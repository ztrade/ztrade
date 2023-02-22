package core

import (
	"fmt"
	"reflect"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/tidwall/gjson"
	. "github.com/ztrade/trademodel"
)

// Events
const (
	EventCandle = "candle"
	EventOrder  = "order"
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

	EventWatch       = "watch"
	EventWatchCandle = "watch_candle"

	EventNotify = "notify"

	EventError = "error"
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
		EventWatchCandle: reflect.TypeOf(CandleParam{}),
	}

	json = jsoniter.ConfigCompatibleWithStandardLibrary
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

type EventData struct {
	Type  string      `json:"type"`
	Data  interface{} `json:"data"`
	Extra interface{} `json:"extra"`
}

// UnmarshalJSON EventData can't be used as Embed
func (d *EventData) UnmarshalJSON(buf []byte) (err error) {
	ret := gjson.ParseBytes(buf)
	d.Type = ret.Get("type").String()
	typ, ok := EventTypes[d.Type]
	if ok {
		d.Data = reflect.New(typ).Interface()
	} else {
		d.Data = map[string]interface{}{}
	}
	err = json.Unmarshal([]byte(ret.Get("data").Raw), d.Data)
	return
}

// WatchParam add watch event param
type WatchParam = EventData

func NewWatchCandle(cp *CandleParam) *WatchParam {
	wp := &WatchParam{
		Type:  EventWatchCandle,
		Data:  cp,
		Extra: cp.Symbol,
	}
	return wp
}

// BalanceInfo balance
type BalanceInfo struct {
	Balance float64
	Fee     float64
}
