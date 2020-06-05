package rpt

import (
	"sort"
	. "ztrade/pkg/define"
	. "ztrade/pkg/event"
	"ztrade/pkg/report"

	. "github.com/SuperGod/trademodel"
	"github.com/mitchellh/mapstructure"
)

type RPTProcesser struct {
	BaseProcesser
	trades []Trade
}

func NewRPTProcesser() (r *RPTProcesser) {
	r = new(RPTProcesser)
	r.Name = "report"
	return
}

func (rpt *RPTProcesser) OnTrade(t Trade) {
	rpt.trades = append(rpt.trades, t)
	return
}

func (rpt *RPTProcesser) Init(bus *Bus) (err error) {
	rpt.BaseProcesser.Init(bus)
	bus.Subscribe(EventTrade, rpt.onEventTrade)
	return
}

func (rpt *RPTProcesser) GenRPT(fPath string) (err error) {
	sort.Slice(rpt.trades, func(i int, j int) bool {
		return rpt.trades[i].Time.Unix() < rpt.trades[j].Time.Unix()
	})
	r := report.NewReport(rpt.trades)
	err = r.Analyzer()
	if err != nil {
		return
	}
	err = r.GenHTMLReport(fPath)
	if err != nil {
		return
	}
	return
}

func (rpt *RPTProcesser) Start() (err error) {
	return
}

func (rpt *RPTProcesser) Stop() (err error) {
	return
}

func (rpt *RPTProcesser) onEventTrade(e Event) (err error) {
	var cParam Trade
	err = mapstructure.Decode(e.GetData(), &cParam)
	if err != nil {
		return
	}
	rpt.trades = append(rpt.trades, cParam)
	return
}
