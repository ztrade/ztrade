package script

import (
	"path/filepath"
	"reflect"
	. "reflect"

	"github.com/ztrade/base/common"
	"github.com/ztrade/ztrade/pkg/process/goscript/engine"

	"github.com/cosmos72/gomacro/base/paths"
	"github.com/cosmos72/gomacro/imports"
	. "github.com/ztrade/trademodel"
)

func init() {
	eng := engine.NewEngine(nil)

	paths.GoSrcDir = filepath.Join(common.GetExecDir(), "plugins")
	imports.Packages["github.com/ztrade/trademodel"] = imports.Package{
		Types: map[string]Type{
			"TradeType":   TypeOf(DirectLong),
			"Trade":       TypeOf(Trade{}),
			"TradeAction": TypeOf(TradeAction{}),
			"Ticker":      TypeOf(Ticker{}),
			"Symbol":      TypeOf(Symbol{}),
			"DepthInfo":   TypeOf(DepthInfo{}),
			"Depth":       TypeOf(Depth{}),
			"Orderbook":   TypeOf(Orderbook{}),
			"Order":       TypeOf(Order{}),
			"CandleList":  TypeOf(CandleList{}),
			"Candle":      TypeOf(Candle{}),
			"Currency":    TypeOf(Currency{}),
			"Balance":     TypeOf(Balance{}),
			"ParamData":   TypeOf(common.ParamData{}),
			"Param":       TypeOf(common.Param{}),
			"Engine":      TypeOf(eng),
		},
		Binds: map[string]Value{
			"min":         reflect.ValueOf(min),
			"max":         reflect.ValueOf(max),
			"formatFloat": reflect.ValueOf(common.FormatFloat),
			"FloatAdd":    reflect.ValueOf(common.FloatAdd),
			"FloatSub":    reflect.ValueOf(common.FloatSub),
			"FloatMul":    reflect.ValueOf(common.FloatMul),
			"FloatDiv":    reflect.ValueOf(common.FloatDiv),
		},
	}

	engine.Register(".go", NewRunnerExport)
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
