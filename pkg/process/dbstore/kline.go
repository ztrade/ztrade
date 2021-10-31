package dbstore

import (
	"fmt"

	. "github.com/ztrade/ztrade/pkg/core"
	. "github.com/ztrade/ztrade/pkg/event"

	log "github.com/sirupsen/logrus"
	. "github.com/ztrade/trademodel"
)

// KlineTbl kline data table
type KlineTbl struct {
	BaseProcesser
	TimeTbl
	loadData bool
}

func NewKlineTbl(db *DBStore, exchange, symbol, binSize string) (t *KlineTbl) {
	t = new(KlineTbl)
	tbl := NewTimeTbl(db, t, exchange, symbol, binSize, "")
	t.TimeTbl = *tbl
	t.BaseProcesser.Name = "klinetbl:" + t.table
	return
}

func (tbl *KlineTbl) Sing() TimeData {
	return new(Candle)
}

func (tbl *KlineTbl) Slice() interface{} {
	return &[]*Candle{}
}
func (tbl *KlineTbl) SetLoadDataMode(bLoad bool) {
	tbl.loadData = bLoad
}

func (tbl *KlineTbl) Init(bus *Bus) (err error) {
	tbl.BaseProcesser.Init(bus)
	if !tbl.loadData {
		tbl.Subscribe(EventCandle, tbl.onEventCandle)
	}
	tbl.Subscribe(EventWatch, tbl.onEventCandleParam)
	return
}

func (tbl *KlineTbl) GetSlice(data interface{}) (rets []interface{}) {
	datas, ok := data.(*[]*Candle)
	if !ok {
		log.Error("KlineTbl getslice error")
		return
	}
	rets = make([]interface{}, len(*datas))
	for k, v := range *datas {
		rets[k] = v
	}
	return
}

func (tbl *KlineTbl) emitCandles(param CandleParam) {
	candles, err := tbl.DataChan(param.Start, param.End, param.BinSize)
	if err != nil {
		log.Error("KlineTbl tbl get candles failed:", err.Error())
		return
	}
	var candle *Candle
	for v := range candles {
		for _, c := range v {
			candle = c.(*Candle)
			tbl.Bus.WaitEmpty()
			tbl.Send(NewCandleName("candle", param.BinSize).String(), EventCandle, candle)
		}
	}
	if tbl.closeCh != nil {
		log.Info("kline table emitCandles finished")
		tbl.closeCh <- true
	}
}

func (tbl *KlineTbl) onEventCandle(e Event) (err error) {
	var candle *Candle
	candle = Map2Candle(e.GetData())
	err = tbl.WriteData(candle)
	if err != nil {
		return
	}
	return
}

func (tbl *KlineTbl) onEventCandleParam(e Event) (err error) {
	wParam, ok := e.GetData().(*WatchParam)
	if !ok {
		err = fmt.Errorf("event not watch %s %#v", e.Name, e.Data)
		return
	}
	candleParam, _ := wParam.Data.(*CandleParam)
	if candleParam == nil {
		err = fmt.Errorf("event not CandleParam %s %#v", e.Name, e.Data)
		return
	}
	go tbl.emitCandles(*candleParam)
	return
}
