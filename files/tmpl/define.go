package main

import (
	. "github.com/SuperGod/trademodel"
	"github.com/ztrade/base/common"
	"github.com/ztrade/base/engine"
)

type Runner interface {
	Param() (paramInfo []common.Param)
	Init(engine Engine, params common.ParamData)
	OnCandle(candle Candle)
	OnPosition(pos, price float64)
	OnTrade(trade Trade)
	OnTradeHistory(trade Trade)
	OnDepth(depth Depth)
	// OnEvent(e Event)
}

type CandleFn = common.CandleFn
type Param = common.Param
type ParamData = common.ParamData

type Engine = engine.Engine

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

var formatFloat = common.FormatFloat

// FloatMul return a*b
var FloatMul = common.FloatMul

// FloatAdd return a*b
var FloatAdd = common.FloatAdd

// FloatSub return a-b
var FloatSub = common.FloatSub

// FloatDiv return a/b
var FloatDiv = common.FloatDiv
