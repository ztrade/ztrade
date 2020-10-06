package script

import (
	"errors"
	"fmt"
	"reflect"

	. "github.com/ztrade/ztrade/pkg/event"

	. "github.com/SuperGod/trademodel"
	"github.com/cosmos72/gomacro/fast"
	"github.com/cosmos72/gomacro/xreflect"
	"github.com/ztrade/base/common"
	"github.com/ztrade/ztrade/pkg/process/goscript/engine"
)

var (
	ErrNoMethod = errors.New("no such method")
)

type CallInfo struct {
	name           string
	instance       reflect.Value
	constructor    reflect.Value
	param          reflect.Value
	init           reflect.Value
	onCandle       reflect.Value
	onPosition     reflect.Value
	onTrade        reflect.Value
	onTradeHistory reflect.Value
	onDepth        reflect.Value
	onEvent        reflect.Value
}

func NewCallInfo(p *fast.Interp, name string, t xreflect.Type) (ci *CallInfo, err error) {
	ci = new(CallInfo)
	ci.name = name
	ci.constructor = p.ValueOf("New" + name).ReflectValue()
	if !ci.constructor.IsValid() {
		err = fmt.Errorf("%w New%s", ErrNoMethod, name)
		return
	}
	rets := ci.constructor.Call([]reflect.Value{})
	if len(rets) == 0 {
		err = fmt.Errorf("constructor error")
		return
	}
	ci.instance = rets[0]
	ci.param, err = ci.extraFunc(t, "Param")
	if err != nil {
		return
	}
	ci.init, err = ci.extraFunc(t, "Init")
	if err != nil {
		return
	}
	ci.onCandle, err = ci.extraFunc(t, "OnCandle")
	if err != nil {
		return
	}
	ci.onPosition, _ = ci.extraFunc(t, "OnPosition")
	ci.onTrade, _ = ci.extraFunc(t, "OnTrade")
	ci.onTradeHistory, _ = ci.extraFunc(t, "OnTradeHistory")
	ci.onDepth, _ = ci.extraFunc(t, "OnDepth")
	ci.onEvent, _ = ci.extraFunc(t, "OnEvent")
	return
}
func (ci *CallInfo) extraFunc(t xreflect.Type, name string) (method reflect.Value, err error) {
	ret, n := t.MethodByName(name, "")
	if n < 1 || ret.Funs == nil || len(*ret.Funs) == 0 {
		err = fmt.Errorf("%w %s", ErrNoMethod, name)
		return
	}
	method = (*ret.Funs)[ret.Index]
	return
}

func (ci *CallInfo) Param() (paramInfo []common.Param, err error) {
	if !ci.param.IsValid() {
		err = fmt.Errorf("%w param", ErrNoMethod)
		return
	}
	rets := ci.param.Call([]reflect.Value{ci.instance})
	if len(rets) == 0 {
		err = fmt.Errorf("call Param() but no returns")
		return
	}
	ret := rets[0].Interface()
	paramInfo, ok := ret.([]common.Param)
	if !ok {
		err = fmt.Errorf("call Param success but return value is not map[string]string")
		return
	}
	return
}
func (ci *CallInfo) Init(engine *engine.Engine, data common.ParamData) (err error) {
	if !ci.init.IsValid() {
		err = fmt.Errorf("%w init", ErrNoMethod)
		return
	}
	ci.init.Call([]reflect.Value{ci.instance, reflect.ValueOf(engine), reflect.ValueOf(data)})
	return
}
func (ci *CallInfo) OnCandle(candle Candle) (err error) {
	if !ci.onCandle.IsValid() {
		err = fmt.Errorf("%w onCandle", ErrNoMethod)
		return
	}
	ci.onCandle.Call([]reflect.Value{ci.instance, reflect.ValueOf(candle)})
	return
}

func (ci *CallInfo) OnPosition(pos, price float64) (err error) {
	if !ci.onPosition.IsValid() {
		err = fmt.Errorf("%w onPosition", ErrNoMethod)
		return
	}
	ci.onPosition.Call([]reflect.Value{ci.instance, reflect.ValueOf(pos), reflect.ValueOf(price)})
	return
}

func (ci *CallInfo) OnTrade(trade Trade) (err error) {
	if !ci.onTrade.IsValid() {
		err = fmt.Errorf("%w onTrade", ErrNoMethod)
		return
	}
	ci.onTrade.Call([]reflect.Value{ci.instance, reflect.ValueOf(trade)})
	return
}
func (ci *CallInfo) OnTradeHistory(trade Trade) (err error) {
	if !ci.onTradeHistory.IsValid() {
		err = fmt.Errorf("%w onTradeHistory", ErrNoMethod)
		return
	}
	ci.onTradeHistory.Call([]reflect.Value{ci.instance, reflect.ValueOf(trade)})
	return
}
func (ci *CallInfo) OnDepth(depth Depth) (err error) {
	if !ci.onDepth.IsValid() {
		err = fmt.Errorf("%w onDepth", ErrNoMethod)
		return
	}
	ci.onDepth.Call([]reflect.Value{ci.instance, reflect.ValueOf(depth)})
	return
}

func (ci *CallInfo) OnEvent(e Event) (err error) {
	if !ci.onEvent.IsValid() {
		err = fmt.Errorf("%w onEvent", ErrNoMethod)
		return
	}
	ci.onEvent.Call([]reflect.Value{ci.instance, reflect.ValueOf(e)})
	return
}
