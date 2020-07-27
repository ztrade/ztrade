package rpt

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	. "github.com/ztrade/ztrade/pkg/define"
	. "github.com/ztrade/ztrade/pkg/event"

	. "github.com/SuperGod/trademodel"
)

// Reporter report generater
type Reporter interface {
	OnTrade(Trade)
	OnBalanceInit(balance float64) (err error)
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
	bus.Subscribe(EventTrade, rpt.OnEventTrade)
	bus.Subscribe(EventBalanceInit, rpt.OnEventBalanceInit)
	return
}

func (rpt *Rpt) Start() (err error) {
	return
}

func (rpt *Rpt) Stop() (err error) {
	return
}

func (rpt *Rpt) OnEventTrade(e Event) (err error) {
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

func (rpt *Rpt) OnEventBalanceInit(e Event) (err error) {
	balance := e.GetData().(*BalanceInfo)
	if balance == nil {
		err = fmt.Errorf("Rpt onEventBalanceInit error %w", err)
		log.Error(err.Error())
		return
	}
	if rpt.rpt != nil {
		rpt.rpt.OnBalanceInit(balance.Balance)
	}
	return
}
