package engine

import (
	"fmt"
	"time"

	"github.com/ztrade/base/common"
	. "github.com/ztrade/ztrade/pkg/define"
	. "github.com/ztrade/ztrade/pkg/event"

	"github.com/SuperGod/indicator"
	. "github.com/SuperGod/trademodel"
	log "github.com/sirupsen/logrus"
)

type Engine struct {
	proc     *BaseProcesser
	pos      float64
	posPrice float64
	balance  float64
	merges   []*KlinePlugin
}

func NewEngine(proc *BaseProcesser) (e *Engine) {
	e = new(Engine)
	e.proc = proc
	return
}

func (e *Engine) OpenLong(price, amount float64) {
	e.addOrder(price, amount, OpenLong)
}
func (e *Engine) CloseLong(price, amount float64) {
	e.addOrder(price, amount, CloseLong)
}
func (e *Engine) OpenShort(price, amount float64) {
	e.addOrder(price, amount, OpenShort)
}
func (e *Engine) CloseShort(price, amount float64) {
	e.addOrder(price, amount, CloseShort)
}
func (e *Engine) StopLong(price, amount float64) {
	e.addOrder(price, amount, StopLong)
}
func (e *Engine) StopShort(price, amount float64) {
	e.addOrder(price, amount, StopShort)
}
func (e *Engine) CancelAllOrder() {
	e.proc.Send(EventOrder, EventOrderCancelAll, nil)
}

func (e *Engine) AddIndicator(name string, params ...int) (ind indicator.CommonIndicator) {
	var err error
	ind, err = indicator.NewCommonIndicator(name, params...)
	if err != nil {
		log.Errorf("ScriptEngine addIndicator failed %s %v", name, params)
		return nil
	}
	return
}
func (e *Engine) UpdatePosition(pos, price float64) {
	e.pos = pos
	e.posPrice = price
}

func (e *Engine) Position() (float64, float64) {
	return e.pos, e.posPrice
}

func (e *Engine) Log(v ...interface{}) {
	fmt.Println(v...)
}

func (e *Engine) addOrder(price, amount float64, orderType TradeType) {
	// FixMe: in backtest, time may be the time of candle
	act := TradeAction{Action: orderType, Amount: amount, Price: price, Time: time.Now()}
	e.proc.Send(EventOrder, EventOrder, act)
}

func (e *Engine) Watch(watchType string) {
	param := WatchParam{Type: watchType}
	e.proc.Send(EventWatch, EventWatch, param)
	return
}

func (e *Engine) SendNotify(content, contentType string) {
	if contentType == "" {
		contentType = "text"
	}
	data := NotifyEvent{Type: contentType, Content: content}
	e.proc.Send("notify", EventNotify, data)
}

func (e *Engine) SetBalance(balance float64) {
	e.proc.Send("balance", "init_balance", map[string]interface{}{"balance": balance})
}

func (e *Engine) Balance() (balance float64) {
	return e.balance
}

func (e *Engine) Merge(src, dst string, fn common.CandleFn) {
	e.merges = append(e.merges, NewKlinePlugin(src, dst, fn))
	return
}

func (e *Engine) OnCandle(candle Candle) {
	for _, v := range e.merges {
		v.Update(candle)
	}
}

func (e *Engine) UpdateBalance(balance float64) {
	e.balance = balance
}
