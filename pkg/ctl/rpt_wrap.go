package ctl

import (
	. "github.com/ztrade/ztrade/pkg/define"
	. "github.com/ztrade/ztrade/pkg/event"

	. "github.com/SuperGod/trademodel"
	"github.com/mitchellh/mapstructure"
)

// Reporter report generater
type Reporter interface {
	OnTrade(Trade)
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
	return
}

func (rpt *Rpt) Start() (err error) {
	return
}

func (rpt *Rpt) Stop() (err error) {
	return
}

func (rpt *Rpt) OnEventTrade(e Event) (err error) {
	var cParam Trade
	err = mapstructure.Decode(e.GetData(), &cParam)
	if err != nil {
		return
	}
	if rpt.rpt != nil {
		rpt.rpt.OnTrade(cParam)
	}
	return
}
