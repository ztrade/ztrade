package rpt

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	. "github.com/ztrade/ztrade/pkg/core"
	. "github.com/ztrade/ztrade/pkg/event"

	. "github.com/ztrade/trademodel"
)

// Reporter report generater
type Reporter interface {
	OnTrade(Trade)
	OnBalanceInit(balance, fee float64) (err error)
}

type Rpt struct {
	BaseProcesser
	rpt Reporter
}

func NewRpt(rpt Reporter) *Rpt {
	r := new(Rpt)
	r.rpt = rpt
	return r
}

func (rpt *Rpt) Init(bus *Bus) (err error) {
	rpt.BaseProcesser.Init(bus)
	rpt.Subscribe(EventTrade, rpt.OnEventTrade)
	rpt.Subscribe(EventBalanceInit, rpt.OnEventBalanceInit)
	return
}

func (rpt *Rpt) Start() (err error) {
	return
}

func (rpt *Rpt) Stop() (err error) {
	return
}

func (rpt *Rpt) OnEventTrade(e *Event) (err error) {
	t := e.GetData().(*Trade)
	if t == nil {
		err = fmt.Errorf("rpt OnEventTrade type error:%#v", e.GetData())
		log.Error(err.Error())
		return
	}
	if rpt.rpt != nil {
		rpt.rpt.OnTrade(*t)
	}
	return
}

func (rpt *Rpt) OnEventBalanceInit(e *Event) (err error) {
	balance := e.GetData().(*BalanceInfo)
	if balance == nil {
		err = fmt.Errorf("Rpt onEventBalanceInit error %w", err)
		log.Error(err.Error())
		return
	}
	if rpt.rpt != nil {
		rpt.rpt.OnBalanceInit(balance.Balance, balance.Fee)
	}
	return
}
