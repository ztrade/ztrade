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

type NewExchangeFn func(cfg *viper.Viper, cltName, symbol string) (t Exchange, err error)

func RegisterExchange(name string, fn NewExchangeFn) {
	exchangeFactory[name] = fn
}

func NewExchange(name string, cfg *viper.Viper, cltName, symbol string) (ex Exchange, err error) {
	fn, ok := exchangeFactory[name]
	if !ok {
		err = fmt.Errorf("no such exchange %s", name)
		return
	}
	ex, err = fn(cfg, cltName, symbol)
	return
}

type ExchangeData struct {
	Name string
	Type string // EventBalance
	Data interface{}
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

	// GetBalanceChan
	GetDataChan() chan *ExchangeData
}
