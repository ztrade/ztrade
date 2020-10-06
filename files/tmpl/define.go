package main

import (
	"github.com/ztrade/base/common"
	"github.com/ztrade/ztrade/pkg/process/goscript/engine"
)

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
