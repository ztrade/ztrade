package helper

import (
	"github.com/SuperGod/indicator"
	. "github.com/SuperGod/trademodel"
)

type Param struct {
	Name string
	Type string
	Info string
}

type CandleFn func(candle Candle)

type Engine struct {
}

func (e *Engine) OpenLong(price, amount float64) {
}
func (e *Engine) CloseLong(price, amount float64) {
}
func (e *Engine) OpenShort(price, amount float64) {
}
func (e *Engine) CloseShort(price, amount float64) {
}
func (e *Engine) StopLong(price, amount float64) {
}
func (e *Engine) StopShort(price, amount float64) {
}
func (e *Engine) CancelAllOrder() {
}

func (e *Engine) AddIndicator(name string, params ...int) (ind indicator.CommonIndicator) {
	return
}

func (e *Engine) Position() (pos, price float64) {
	return
}

func (e *Engine) Balance() float64 {
	return 0
}

func (e *Engine) Log(v ...interface{}) {
}

func (e *Engine) Watch(watchType string) {
	return
}

func (e *Engine) SendNotify(content, contentType string) {
}
func (e *Engine) Merge(src, dst string, fn CandleFn) {
	return
}

func (e *Engine) SetBalance(balance float64) {
}

type ParamData map[string]interface{}

func (d ParamData) GetString(key, defaultValue string) string {
	return ""
}
func (d ParamData) GetFloat(key string, defaultValue float64) float64 {
	return 0
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
