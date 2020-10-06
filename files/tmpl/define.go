package main

import (
	. "github.com/SuperGod/trademodel"
	"github.com/ztrade/ztrade/pkg/common"
)

type CandleFn = common.CandleFn
type Param = common.Param
type ParamData = common.ParamData

type Engine interface {
	OpenLong(price, amount float64)
	CloseLong(price, amount float64)
	OpenShort(price, amount float64)
	CloseShort(price, amount float64)
	StopLong(price, amount float64)
	StopShort(price, amount float64)
	CancelAllOrder()
	AddIndicator(name string, params ...int)
	Position() (pos, price float64)
	Balance() float64
	Log(v ...interface{})
	Watch(watchType string)
	SendNotify(content, contentType string)
	Merge(src, dst string, fn CandleFn)
	SetBalance(balance float64)
	GetString(key, defaultValue string)
	GetFloat(key string, defaultValue float64) float64
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

var formatFloat = common.FormatFloat

// FloatMul return a*b
var FloatMul = common.FloatMul

// FloatAdd return a*b
var FloatAdd = common.FloatAdd

// FloatSub return a-b
var FloatSub = common.FloatSub

// FloatDiv return a/b
var FloatDiv = common.FloatDiv
