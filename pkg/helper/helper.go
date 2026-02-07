package helper

// This package provides type aliases and placeholder functions for writing
// strategy scripts. When strategies are compiled as Go plugins via the
// "build" command, these stubs are replaced by real implementations from
// the tmpl/define.go template. The return values here are never actually
// called at runtime.

import (
	"github.com/ztrade/base/common"
	"github.com/ztrade/base/engine"
	. "github.com/ztrade/trademodel"
)

type CandleFn func(candle Candle)
type Engine = engine.Engine
type Param = common.Param
type ParamData = common.ParamData

var StringParam = common.StringParam
var IntParam = common.IntParam
var FloatParam = common.FloatParam
var BoolParam = common.BoolParam

// The following are compile-time stubs. Real implementations are injected
// via tmpl/define.go when building strategy plugins.

func min(a, b float64) float64 {
	return 0
}

func max(a, b float64) float64 {
	return 0
}

func formatFloat(n float64, precision int) float64 {
	return 0
}

// FloatMul return a*b
func FloatMul(a, b float64) float64 {
	return 0
}

// FloatAdd return a+b
func FloatAdd(a, b float64) float64 {
	return 0
}

// FloatSub return a-b
func FloatSub(a, b float64) float64 {
	return 0
}

// FloatDiv return a/b
func FloatDiv(a, b float64) float64 {
	return 0
}
