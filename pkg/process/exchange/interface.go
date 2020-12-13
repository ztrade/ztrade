package exchange

import (
	"time"

	. "github.com/SuperGod/trademodel"
	"github.com/spf13/viper"
	. "github.com/ztrade/ztrade/pkg/define"
)

var (
	exchangeFactory = map[string]NewExchangeFn{}
)

func RegisterExchange(name string, fn NewExchangeFn) {
	exchangeFactory[name] = fn
}

type ExchangeChan struct {
	BalanceChan chan Balance
	PosChan     chan Position
	OrderChan   chan Order
	DepthChan   chan Depth
	TradeChan   chan Trade
}

func NewExchangeChan() *ExchangeChan {
	ec := new(ExchangeChan)
	ec.BalanceChan = make(chan Balance)
	ec.PosChan = make(chan Position)
	ec.OrderChan = make(chan Order)
	ec.DepthChan = make(chan Depth)
	ec.TradeChan = make(chan Trade)
	return ec
}

func (ec *ExchangeChan) Close() {
	close(ec.BalanceChan)
	close(ec.PosChan)
	close(ec.OrderChan)
	close(ec.DepthChan)
	close(ec.TradeChan)
}

type NewExchangeFn func(cfg *viper.Viper, cltName, symbol string) (t TradeExchange, err error)

type Exchange interface {
	Start() error
	Stop() error

	// KlineChan get klines
	KlineChan(start, end time.Time, symbol, bSize string) (data chan *Candle, err chan error)

	// watch kline changes
	WatchKline(symbols SymbolInfo) (datas chan *CandleInfo, stopC chan struct{}, err error)
	Watch(WatchParam) error

	// for trade
	// ProcessOrder process order
	ProcessOrder(act TradeAction) (ret *Order, err error)
	CancelAllOrders() (orders []*Order, err error)

	// GetBalanceChan
	GetDataChan() *ExchangeChan
}
