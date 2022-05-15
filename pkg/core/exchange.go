package core

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
	. "github.com/ztrade/trademodel"
)

var (
	exchangeFactory = map[string]NewExchangeFn{}
)

type NewExchangeFn func(cfg *viper.Viper, cltName string) (t Exchange, err error)

func RegisterExchange(name string, fn NewExchangeFn) {
	exchangeFactory[name] = fn
}

func NewExchange(name string, cfg *viper.Viper, cltName string) (ex Exchange, err error) {
	fn, ok := exchangeFactory[name]
	if !ok {
		err = fmt.Errorf("no such exchange %s", name)
		return
	}
	ex, err = fn(cfg, cltName)
	return
}

type ExchangeData struct {
	Data   EventData `json:"data"`
	Name   string
	Symbol string
}

func (e *ExchangeData) GetType() string {
	return e.Data.Type
}

func (e *ExchangeData) GetData() interface{} {
	return e.Data.Data
}

func NewExchangeData(name, typ string, data interface{}) *ExchangeData {
	return &ExchangeData{Name: name, Data: EventData{Type: typ, Data: data}}
}

type Exchange interface {
	Start(map[string]interface{}) error
	Stop() error

	// Kline get klines
	GetKline(symbol, bSize string, start, end time.Time) (data chan *Candle, err chan error)

	Watch(WatchParam) error

	// for trade
	// ProcessOrder process order
	ProcessOrder(act TradeAction) (ret *Order, err error)
	CancelAllOrders() (orders []*Order, err error)
	CancelOrder(old *Order) (orders *Order, err error)
	// GetBalanceChan
	GetDataChan() chan *ExchangeData
	GetSymbols() ([]SymbolInfo, error)
}
