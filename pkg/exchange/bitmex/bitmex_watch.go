package bitmex

import (
	"time"

	. "github.com/ztrade/ztrade/pkg/define"

	"github.com/SuperGod/coinex/bitmex"
	. "github.com/SuperGod/trademodel"
	log "github.com/sirupsen/logrus"
)

// KlineChan impl KlineMaker
func (b *BitmexTrade) KlineChan(start, end time.Time, symbol, bSize string) (data chan *Candle, err chan error) {
	bm := b.bm.Clone()
	bm.SetSymbol(symbol)
	temp, err := bm.KlineChan(start, end, bSize)
	if err != nil {
		return
	}
	data = make(chan *Candle, 1024)
	go func() {
		for v := range temp {
			for _, d := range v {
				cd := d.(*Candle)
				data <- cd
			}
		}
	}()
	return
}

// WatchKline impl KlineMaker
func (b *BitmexTrade) WatchKline(symbols SymbolInfo) (datas chan *CandleInfo, stopC chan struct{}, err error) {
	datas = make(chan *CandleInfo)
	stopC = make(chan struct{})
	bm := b.bm.Clone()
	tbls := make(map[string]string)
	var bSize string
	var subs []bitmex.SubscribeInfo

	for _, r := range symbols.GetResolutions() {
		bSize = "tradeBin" + r
		subs = append(subs, bitmex.SubscribeInfo{Op: bSize, Param: symbols.Symbol})
		tbls[bSize] = r
	}

	log.Info("subscribe:", subs)
	bm.WS().SetSubscribe(subs)

	process := func(tbl string, msg *bitmex.Resp) {
		rets := msg.GetTradeBin()
		if len(rets) == 0 {
			return
		}
		// var t time.Time
		for _, v := range rets {
			// t = time.Time(*v.Timestamp)
			info := &CandleInfo{Exchange: b.Name,
				Symbol:  *v.Symbol,
				BinSize: tbls[tbl],
			}
			info.Data = bitmex.TransCandle(tbls[tbl], v)
			datas <- info
		}
	}
	for k := range tbls {
		bm.WS().SetHandle(k, process)
	}
	err = bm.StartWS()
	if err != nil {
		log.Error("bitmex startws error:", err.Error())
	}
	return
}
