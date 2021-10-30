package helper

import (
	"github.com/ztrade/base/common"
	"github.com/ztrade/base/engine"
	. "github.com/ztrade/trademodel"
)

type CandleFn func(candle Candle)
type Engine = engine.Engine
type Param = common.Param
type ParamData = common.ParamData

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

// FloatAdd return a*b
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
