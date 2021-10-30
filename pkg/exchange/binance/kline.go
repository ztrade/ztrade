package binance

import (
	"sync"

	"github.com/adshao/go-binance/v2/futures"
	"github.com/ztrade/ztrade/pkg/core"
)

type klineFilter struct {
	name       string
	binSize    string
	datasMutex sync.Mutex
	datas      chan *core.CandleInfo
	last       *futures.WsKline
	doneC      chan struct{}
	isClosed   bool
}

func newKlineFilter(name, binSize string) *klineFilter {
	f := new(klineFilter)
	f.name = name
	f.binSize = binSize
	f.datas = make(chan *core.CandleInfo)
	f.doneC = make(chan struct{})
	return f
}

func (f *klineFilter) GetData() chan *core.CandleInfo {
	return f.datas
}

func (f *klineFilter) ProcessEvent(event *futures.WsKlineEvent) {
	f.datasMutex.Lock()
	defer f.datasMutex.Unlock()
	if f.isClosed {
		return
	}
	if f.last == nil || event.Kline.StartTime == f.last.StartTime {
		f.last = &event.Kline
		return
	}
	ci := transWSCandle(f.last)
	ci.BinSize = f.binSize
	ci.Exchange = f.name
	f.last = &event.Kline
	f.datas <- ci
}

func (f *klineFilter) Close() {
	f.datasMutex.Lock()
	f.isClosed = true
	close(f.datas)
	f.datasMutex.Unlock()
}
