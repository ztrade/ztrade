package goscript

import (
	"fmt"
	"path/filepath"
	"reflect"
	. "reflect"
	"strconv"
	"ztrade/pkg/common"

	. "github.com/SuperGod/trademodel"
	"github.com/cosmos72/gomacro/base/paths"
	"github.com/cosmos72/gomacro/imports"
)

func init() {

	paths.GoSrcDir = filepath.Join(common.GetExecDir(), "plugins")
	imports.Packages["github.com/SuperGod/trademodel"] = imports.Package{
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
			"ParamData":   TypeOf(ParamData{}),
		},
		Binds: map[string]Value{
			"min":         reflect.ValueOf(min),
			"max":         reflect.ValueOf(max),
			"formatFloat": reflect.ValueOf(formatFloat),
			"FloatAdd":    reflect.ValueOf(common.FloatAdd),
			"FloatSub":    reflect.ValueOf(common.FloatSub),
			"FloatMul":    reflect.ValueOf(common.FloatMul),
			"FloatDiv":    reflect.ValueOf(common.FloatDiv),
		},
	}
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

func formatFloat(n float64, precision int) float64 {
	str := fmt.Sprintf("%df", precision)
	n2, _ := strconv.ParseFloat(fmt.Sprintf("%."+str, n), 64)
	return n2
}
