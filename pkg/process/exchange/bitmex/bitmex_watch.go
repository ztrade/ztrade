package bitmex

import (
	"reflect"
	"strings"
	"time"

	. "github.com/ztrade/ztrade/pkg/define"

	"github.com/SuperGod/coinex/bitmex"
	"github.com/SuperGod/coinex/bitmex/models"
	. "github.com/SuperGod/trademodel"
	log "github.com/sirupsen/logrus"
)

// KlineChan impl KlineMaker
func (b *BitmexTrade) KlineChan(start, end time.Time, symbol, bSize string) (data chan []interface{}, err chan error) {
	bm := b.bm.Clone()
	bm.SetSymbol(symbol)
	return bm.KlineChan(start, end, bSize)
}

func (b *BitmexTrade) checkMissingKline(binsize string, last time.Time, kline *models.TradeBin, datas chan *CandleInfo) {
	if kline == nil {
		return
	}
	symbol := *kline.Symbol
	tFirst := time.Time(*kline.Timestamp)
	nDur := tFirst.Sub(last)
	dur, err := time.ParseDuration(binsize)
	if err != nil {
		log.Errorf("BitmexTrade checkMissingKline %s %s failed:%s", symbol, binsize, err.Error())
		return
	}
	if nDur <= dur {
		return
	}
	log.Infof("BitmexTrade checkMissingKline download missing date %s %s", last, tFirst)
	klines, errChan := b.bm.KlineChan(last, tFirst, binsize)
	for ks := range klines {
		for k := 0; k != len(ks); k++ {
			v, ok := ks[k].(*Candle)
			if !ok {
				log.Errorf("candles type error:%s", reflect.TypeOf(ks[k]))
				continue
			}
			if v.Time().Sub(last) <= 0 {
				continue
			}
			info := &CandleInfo{Exchange: b.GetName(),
				Symbol:  symbol,
				BinSize: binsize,
			}
			info.Data = *v
			datas <- info
		}
	}
	err = <-errChan
	if err != nil {
		log.Infof("BitmexTrade checkMissingKline error:", err.Error())
	}
}

// WatchKline impl KlineMaker
func (b *BitmexTrade) WatchKline(symbols []SymbolInfo, datas chan *CandleInfo) (err error) {
	bm := b.bm.Clone()
	tbls := make(map[string]string)
	var bSize string
	var subs []bitmex.SubscribeInfo
	for _, v := range symbols {
		for _, r := range v.GetResolutions() {
			bSize = "tradeBin" + r
			subs = append(subs, bitmex.SubscribeInfo{Op: bSize, Param: v.Symbol})
			tbls[bSize] = r
		}
	}
	log.Info("subscribe:", subs)
	bm.WS().SetSubscribe(subs)

	process := func(tbl string, msg *bitmex.Resp) {
		rets := msg.GetTradeBin()
		if len(rets) == 0 {
			return
		}
		binsize := strings.TrimLeft(tbl, "tradeBin")
		last, ok := b.lastKlines[tbl]
		if ok {
			// check missing kline
			b.checkMissingKline(binsize, last, rets[0], datas)
		}
		var t time.Time
		for _, v := range rets {
			t = time.Time(*v.Timestamp)
			if t.Sub(last) <= 0 {
				continue
			}
			info := &CandleInfo{Exchange: b.GetName(),
				Symbol:  *v.Symbol,
				BinSize: tbls[tbl],
			}
			info.Data = bitmex.TransCandle(tbls[tbl], v)
			b.lastKlines[tbl] = t
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
