package goscript

import (
	"ztrade/pkg/common"

	. "github.com/SuperGod/trademodel"
	log "github.com/sirupsen/logrus"
)

type CandleFn func(candle Candle)

type KlinePlugin struct {
	kl      *common.KlineMerge
	cb      CandleFn
	bRecent bool
}

func NewKlinePlugin(src, dst string, fn CandleFn) (kp *KlinePlugin) {
	kp = new(KlinePlugin)
	kp.cb = fn
	kp.kl = common.NewKlineMergeStr(src, dst)
	return
}

func (kp *KlinePlugin) Update(candle Candle) {
	if candle.ID == -1 {
		kp.bRecent = true
	} else {
		kp.bRecent = false
	}
	ret := kp.kl.Update(&candle)
	if ret == nil {
		return
	}
	if kp.cb == nil {
		log.Error("KlinePlugin callback is nil")
		return
	}
	if kp.bRecent {
		temp := ret.(*Candle)
		temp.ID = -1
	}
	newCandle := ret.(*Candle)
	kp.cb(*newCandle)
}
