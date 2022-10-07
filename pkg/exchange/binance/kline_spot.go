package binance

import (
	"sync"

	gobinance "github.com/adshao/go-binance/v2"
	"github.com/ztrade/ztrade/pkg/core"
)

type spotKlineFilter struct {
	name       string
	binSize    string
	datasMutex sync.Mutex
	datas      chan *core.CandleInfo
	last       *gobinance.WsKline
	doneC      chan struct{}
	isClosed   bool
}

func newSpotKlineFilter(name, binSize string) *spotKlineFilter {
	f := new(spotKlineFilter)
	f.name = name
	f.binSize = binSize
	f.datas = make(chan *core.CandleInfo)
	f.doneC = make(chan struct{})
	return f
}

func (f *spotKlineFilter) GetData() chan *core.CandleInfo {
	return f.datas
}

func (f *spotKlineFilter) ProcessEvent(event *gobinance.WsKlineEvent) {
	f.datasMutex.Lock()
	defer f.datasMutex.Unlock()
	if f.isClosed {
		return
	}
	if f.last == nil || event.Kline.StartTime == f.last.StartTime {
		f.last = &event.Kline
		return
	}
	ci := transSpotWSCandle(f.last)
	ci.BinSize = f.binSize
	ci.Exchange = f.name
	f.last = &event.Kline
	f.datas <- ci
}

func (f *spotKlineFilter) Close() {
	f.datasMutex.Lock()
	f.isClosed = true
	close(f.datas)
	f.datasMutex.Unlock()
}
