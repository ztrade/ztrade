package define

import (
	"time"
)

// CandleInfo candle data with  symbol info
type CandleInfo struct {
	Exchange string
	Symbol   string
	BinSize  string
	Data     interface{}
}

// KlineMaker kline maker
type KlineMaker interface {
	GetName() string
	KlineChan(start, end time.Time, symbol, bSize string) (klines chan []interface{}, err chan error)
	WatchKline(symbols []SymbolInfo, datas chan *CandleInfo) error
}
