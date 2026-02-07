package engine

import (
	"fmt"
	"math/rand/v2"
	"sync"
	"time"

	"github.com/ztrade/base/common"
	. "github.com/ztrade/ztrade/pkg/core"
	. "github.com/ztrade/ztrade/pkg/event"

	log "github.com/sirupsen/logrus"
	"github.com/ztrade/indicator"
	. "github.com/ztrade/trademodel"
)

const letterBytes = "abcdefghijklmnopqrstuvwxyz123456789"

func randStringBytes(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.IntN(len(letterBytes))]
	}
	return string(b)
}

func getActionID() string {
	return randStringBytes(8)
}

type EngineImpl struct {
	proc        *BaseProcesser
	pos         float64
	posPrice    float64
	balance     float64
	merges      map[string][]*KlinePlugin
	mergesMutex sync.Mutex
	symbol      string
}

type UpdateStatusFn func(vm string, status int, msg string)
type EngineWrapper struct {
	*EngineImpl
	VmID string
	Cb   UpdateStatusFn
}

func (e *EngineWrapper) UpdateStatus(status int, msg string) {
	e.Cb(e.VmID, status, msg)
}

func (e *EngineWrapper) Merge(src, dst string, fn common.CandleFn) {
	e.EngineImpl.Merge(e.VmID, src, dst, fn)
}

func (e *EngineWrapper) CleanMerges() {
	e.EngineImpl.RemoveMerge(e.VmID)
}

func NewEngineWrapper(proc *BaseProcesser, cb UpdateStatusFn, symbol string, id string) *EngineWrapper {
	return &EngineWrapper{EngineImpl: NewEngineImpl(proc, symbol), Cb: cb, VmID: id}
}

func NewEngineImpl(proc *BaseProcesser, symbol string) *EngineImpl {
	e := new(EngineImpl)
	e.merges = make(map[string][]*KlinePlugin)
	e.symbol = symbol
	e.proc = proc
	return e
}

func (e *EngineWrapper) OpenLong(price, amount float64) string {
	return e.addOrder(price, amount, OpenLong)
}
func (e *EngineWrapper) CloseLong(price, amount float64) string {
	return e.addOrder(price, amount, CloseLong)
}
func (e *EngineWrapper) OpenShort(price, amount float64) string {
	return e.addOrder(price, amount, OpenShort)
}
func (e *EngineWrapper) CloseShort(price, amount float64) string {
	return e.addOrder(price, amount, CloseShort)
}
func (e *EngineWrapper) StopLong(price, amount float64) string {
	return e.addOrder(price, amount, StopLong)
}
func (e *EngineWrapper) StopShort(price, amount float64) string {
	return e.addOrder(price, amount, StopShort)
}

func (e *EngineWrapper) DoOrder(typ TradeType, price, amount float64) string {
	return e.addOrder(price, amount, typ)
}

func (e *EngineImpl) CancelAllOrder() {
	e.proc.Send(EventOrder, EventOrder, &TradeAction{Action: CancelAll})
}

func (e *EngineImpl) CancelOrder(id string) {
	e.proc.Send(EventOrder, EventOrder, &TradeAction{Action: CancelOne, ID: id})
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

func (e *EngineWrapper) addOrder(price, amount float64, orderType TradeType) (id string) {
	// FixMe: in backtest, time may be the time of candle
	id = fmt.Sprintf("%s-%s", e.VmID, getActionID())
	act := TradeAction{ID: id, Action: orderType, Symbol: e.symbol, Amount: amount, Price: price, Time: time.Now()}
	e.proc.Send(EventOrder, EventOrder, &act)
	return
}

func (e *EngineImpl) Watch(watchType string) {
	param := WatchParam{Type: watchType}
	e.proc.Send(EventWatch, EventWatch, &param)
}

func (e *EngineImpl) SendNotify(title, content, contentType string) {
	if contentType == "" {
		contentType = "text"
	}
	data := NotifyEvent{Title: title, Type: contentType, Content: content}
	e.proc.Send("notify", EventNotify, &data)
}

func (e *EngineImpl) SetBalance(balance float64) {
	e.proc.Send("balance", "init_balance", map[string]interface{}{"balance": balance})
}

func (e *EngineImpl) Balance() (balance float64) {
	return e.balance
}

func (e *EngineImpl) Merge(vmID, src, dst string, fn common.CandleFn) {
	e.mergesMutex.Lock()
	defer e.mergesMutex.Unlock()
	kp := NewKlinePlugin(src, dst, fn)
	ms, ok := e.merges[vmID]
	if ok {
		e.merges[vmID] = append(ms, kp)
	} else {
		e.merges[vmID] = []*KlinePlugin{kp}
	}
}

func (e *EngineImpl) RemoveMerge(vmID string) {
	e.mergesMutex.Lock()
	defer e.mergesMutex.Unlock()
	delete(e.merges, vmID)
}

func (e *EngineImpl) OnCandle(candle *Candle) {
	for _, kls := range e.merges {
		for _, v := range kls {
			v.Update(candle)
		}
	}
}

func (e *EngineImpl) UpdateBalance(balance float64) {
	e.balance = balance
}

func (e *EngineImpl) UpdateStatus(status int, msg string) {
	log.Error("EngineImpl UpdateStatus, never called")
}
