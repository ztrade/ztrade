package engine

import (
	"fmt"
	"path/filepath"

	"github.com/ztrade/base/common"
	"github.com/ztrade/base/engine"
	. "github.com/ztrade/trademodel"
	. "github.com/ztrade/ztrade/pkg/event"
)

type NewRunnerFn func(file string) (r Runner, err error)

var (
	factory = map[string]NewRunnerFn{}
)

func Register(ext string, fn NewRunnerFn) {
	factory[ext] = fn
}

type Runner interface {
	Param() (paramInfo []common.Param, err error)
	Init(engine engine.Engine, params common.ParamData) (err error)
	OnCandle(candle *Candle) (err error)
	OnPosition(pos, price float64) (err error)
	OnTrade(trade *Trade) (err error)
	OnTradeMarket(trade *Trade) (err error)
	OnDepth(depth *Depth) (err error)
	OnEvent(e *Event) (err error)
	GetName() string
}

func NewRunner(file string) (r Runner, err error) {
	ext := filepath.Ext(file)
	f, ok := factory[ext]
	if !ok {
		err = fmt.Errorf("unsupport file format: %s", ext)
		return
	}
	r, err = f(file)
	return
}
