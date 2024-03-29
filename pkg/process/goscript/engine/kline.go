package engine

import (
	"github.com/ztrade/base/common"

	log "github.com/sirupsen/logrus"
	. "github.com/ztrade/trademodel"
)

type KlinePlugin struct {
	kl      *common.KlineMerge
	cb      common.CandleFn
	bRecent bool
}

func NewKlinePlugin(src, dst string, fn common.CandleFn) (kp *KlinePlugin) {
	kp = new(KlinePlugin)
	kp.cb = fn
	kp.kl = common.NewKlineMergeStr(src, dst)
	return
}

func (kp *KlinePlugin) Update(candle *Candle) {
	if candle.ID == -1 {
		kp.bRecent = true
	} else {
		kp.bRecent = false
	}
	ret := kp.kl.Update(candle)
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
	kp.cb(newCandle)
}
