package engine

import (
	"fmt"
	"time"

	"github.com/ztrade/base/common"
	"github.com/ztrade/base/engine"
	. "github.com/ztrade/ztrade/pkg/core"
	. "github.com/ztrade/ztrade/pkg/event"

	log "github.com/sirupsen/logrus"
	"github.com/ztrade/indicator"
	. "github.com/ztrade/trademodel"
)

type EngineImpl struct {
	proc     *BaseProcesser
	pos      float64
	posPrice float64
	balance  float64
	merges   []*KlinePlugin
}

func NewEngine(proc *BaseProcesser) engine.Engine {
	e := new(EngineImpl)
	e.proc = proc
	return e
}

func (e *EngineImpl) OpenLong(price, amount float64) {
	e.addOrder(price, amount, OpenLong)
}
func (e *EngineImpl) CloseLong(price, amount float64) {
	e.addOrder(price, amount, CloseLong)
}
func (e *EngineImpl) OpenShort(price, amount float64) {
	e.addOrder(price, amount, OpenShort)
}
func (e *EngineImpl) CloseShort(price, amount float64) {
	e.addOrder(price, amount, CloseShort)
}
func (e *EngineImpl) StopLong(price, amount float64) {
	e.addOrder(price, amount, StopLong)
}
func (e *EngineImpl) StopShort(price, amount float64) {
	e.addOrder(price, amount, StopShort)
}
func (e *EngineImpl) CancelAllOrder() {
	e.proc.Send(EventOrder, EventOrder, &TradeAction{Action: CancelAll})
}

func (e *EngineImpl) AddIndicator(name string, params ...int) (ind indicator.CommonIndicator) {
	var err error
	ind, err = indicator.NewCommonIndicator(name, params...)
	if err != nil {
		log.Errorf("ScriptEngineImpl addIndicator failed %s %v", name, params)
		return nil
	}
	return
}
func (e *EngineImpl) UpdatePosition(pos, price float64) {
	e.pos = pos
	e.posPrice = price
}

func (e *EngineImpl) Position() (float64, float64) {
	return e.pos, e.posPrice
}

func (e *EngineImpl) Log(v ...interface{}) {
	fmt.Println(v...)
}

func (e *EngineImpl) addOrder(price, amount float64, orderType TradeType) {
	// FixMe: in backtest, time may be the time of candle
	act := TradeAction{Action: orderType, Amount: amount, Price: price, Time: time.Now()}
	e.proc.Send(EventOrder, EventOrder, &act)
}

func (e *EngineImpl) Watch(watchType string) {
	param := WatchParam{Type: watchType}
	e.proc.Send(EventWatch, EventWatch, &param)
}

func (e *EngineImpl) SendNotify(content, contentType string) {
	if contentType == "" {
		contentType = "text"
	}
	data := NotifyEvent{Type: contentType, Content: content}
	e.proc.Send("notify", EventNotify, &data)
}

func (e *EngineImpl) SetBalance(balance float64) {
	e.proc.Send("balance", "init_balance", map[string]interface{}{"balance": balance})
}

func (e *EngineImpl) Balance() (balance float64) {
	return e.balance
}

func (e *EngineImpl) Merge(src, dst string, fn common.CandleFn) {
	e.merges = append(e.merges, NewKlinePlugin(src, dst, fn))
}

func (e *EngineImpl) OnCandle(candle *Candle) {
	for _, v := range e.merges {
		v.Update(candle)
	}
}

func (e *EngineImpl) UpdateBalance(balance float64) {
	e.balance = balance
}
