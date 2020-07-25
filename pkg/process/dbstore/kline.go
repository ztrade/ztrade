package dbstore

import (
	. "github.com/ztrade/ztrade/pkg/define"
	. "github.com/ztrade/ztrade/pkg/event"

	. "github.com/SuperGod/trademodel"
	"github.com/mitchellh/mapstructure"
	log "github.com/sirupsen/logrus"
)

// KlineTbl kline data table
type KlineTbl struct {
	BaseProcesser
	TimeTbl
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

func (tbl *KlineTbl) Init(bus *Bus) (err error) {
	tbl.BaseProcesser.Init(bus)
	bus.Subscribe(EventCandle, tbl.onEventCandle)
	bus.Subscribe(EventCandleParam, tbl.onEventCandleParam)
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
	var cParam CandleParam
	// d := e.GetData()
	err = mapstructure.Decode(e.GetData(), &cParam)
	if err != nil {
		log.Error("KlineTbl OnEventCandleParam error:", err.Error())
		return
	}
	if e.GetName() == "load_candle" {
		go tbl.emitCandles(cParam)
	}
	return
}
