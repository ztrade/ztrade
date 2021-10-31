package core

// CandleInfo candle data with  symbol info
type CandleInfo struct {
	Exchange string
	Symbol   string
	BinSize  string
	Data     interface{}
}
