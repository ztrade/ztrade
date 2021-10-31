package plugin

import (
	"fmt"
	"path/filepath"
	"plugin"
	"reflect"

	"github.com/ztrade/base/common"
	bengine "github.com/ztrade/base/engine"
	. "github.com/ztrade/trademodel"
	. "github.com/ztrade/ztrade/pkg/event"
	"github.com/ztrade/ztrade/pkg/process/goscript/engine"
)

func init() {
	engine.Register(".so", NewPlugin)
	engine.Register(".dll", NewPlugin)
	engine.Register(".dylib", NewPlugin)
}

type newFn func() interface{}

type StrategyPlugin struct {
	name string
	Runner
}

func NewPlugin(file string) (r engine.Runner, err error) {
	pl, err := plugin.Open(file)
	if err != nil {
		return
	}
	v, err := pl.Lookup("NewStrategy")
	if err != nil {
		return
	}

	rValue := reflect.ValueOf(v).Elem()
	ret := rValue.Call([]reflect.Value{})
	if len(ret) == 0 {
		err = fmt.Errorf("%s constructor error %#v", file, v)
		return
	}
	value := ret[0].Interface()
	temp, ok := value.(Runner)
	if !ok {
		err = fmt.Errorf("%s not impl func() Runner %#v", file, value)
		return
	}
	sp := new(StrategyPlugin)
	sp.name = filepath.Base(file)
	sp.Runner = temp
	r = sp
	return
}
func (sp *StrategyPlugin) GetName() string {
	return sp.name
}

func (sp *StrategyPlugin) Param() (paramInfo []common.Param, err error) {
	paramInfo = sp.Runner.Param()
	return
}
func (sp *StrategyPlugin) Init(engine bengine.Engine, params common.ParamData) (err error) {
	sp.Runner.Init(engine, params)
	return
}
func (sp *StrategyPlugin) OnCandle(candle Candle) (err error) {
	sp.Runner.OnCandle(candle)
	return
}
func (sp *StrategyPlugin) OnPosition(pos, price float64) (err error) {
	sp.Runner.OnPosition(pos, price)
	return
}
func (sp *StrategyPlugin) OnTrade(trade Trade) (err error) {
	sp.Runner.OnTrade(trade)
	return
}
func (sp *StrategyPlugin) OnTradeMarket(trade Trade) (err error) {
	sp.Runner.OnTradeMarket(trade)
	return
}
func (sp *StrategyPlugin) OnDepth(depth Depth) (err error) {
	sp.Runner.OnDepth(depth)
	return
}
func (sp *StrategyPlugin) OnEvent(e Event) (err error) {
	return
}
