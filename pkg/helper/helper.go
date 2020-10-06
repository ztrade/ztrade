package helper

import (
	"github.com/SuperGod/indicator"
	. "github.com/SuperGod/trademodel"
)

type CandleFn func(candle Candle)

type Engine struct {
}

// OpenLong open long order
func (e *Engine) OpenLong(price, amount float64) {
}

// CloseLong close long order
func (e *Engine) CloseLong(price, amount float64) {
}

// OpenShort open short order
func (e *Engine) OpenShort(price, amount float64) {
}

// CLoseShort close short order
func (e *Engine) CloseShort(price, amount float64) {
}

// StopLong stop long order
func (e *Engine) StopLong(price, amount float64) {
}

// StopShort stop short order
func (e *Engine) StopShort(price, amount float64) {
}

// CancelAllOrder cancel all order
func (e *Engine) CancelAllOrder() {
}

// AddIndicator add indicator
func (e *Engine) AddIndicator(name string, params ...int) (ind indicator.CommonIndicator) {
	return
}

// position get current position
func (e *Engine) Position() (pos, price float64) {
	return
}

// Balance get current balance
func (e *Engine) Balance() float64 {
	return 0
}

// Log log
func (e *Engine) Log(v ...interface{}) {
}

// Watch add watch events: trade_history, depth
func (e *Engine) Watch(watchType string) {
	return
}

// SendNotify send notify
func (e *Engine) SendNotify(content, contentType string) {
}

// Merge merge klines
func (e *Engine) Merge(src, dst string, fn CandleFn) {
	return
}

func (e *Engine) SetBalance(balance float64) {
}

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
