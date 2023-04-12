package igo

import (
	"github.com/ztrade/base/common"
	"github.com/ztrade/base/engine"
	. "github.com/ztrade/trademodel"
	. "github.com/ztrade/ztrade/pkg/event"
)

type igoImpl interface {
	Param() (paramInfo []common.Param)
	Init(engine engine.Engine, params common.ParamData) error
	OnCandle(candle *Candle)
	OnPosition(pos, price float64)
	OnTrade(trade *Trade)
	OnTradeMarket(trade *Trade)
	OnDepth(depth *Depth)
	// OnEvent(e *Event)
	// GetName() string
}

type igoRunner struct {
	name string
	impl igoImpl
}

func (r *igoRunner) Param() (paramInfo []common.Param, err error) {
	paramInfo = r.impl.Param()
	return
}

func (r *igoRunner) Init(engine engine.Engine, params common.ParamData) (err error) {
	return r.impl.Init(engine, params)
}

func (r *igoRunner) OnCandle(candle *Candle) (err error) {
	r.impl.OnCandle(candle)
	return
}

func (r *igoRunner) OnPosition(pos, price float64) (err error) {
	r.impl.OnPosition(pos, price)
	return
}

func (r *igoRunner) OnTrade(trade *Trade) (err error) {
	r.impl.OnTrade(trade)
	return
}

func (r *igoRunner) OnTradeMarket(trade *Trade) (err error) {
	r.impl.OnTradeMarket(trade)
	return
}

func (r *igoRunner) OnDepth(depth *Depth) (err error) {
	r.impl.OnDepth(depth)
	return
}

func (r *igoRunner) OnEvent(e *Event) (err error) {
	return
}

func (r *igoRunner) GetName() string {
	return r.name
}
