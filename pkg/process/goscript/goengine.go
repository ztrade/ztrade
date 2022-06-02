package goscript

import (
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/ztrade/base/common"
	bengine "github.com/ztrade/base/engine"
	. "github.com/ztrade/ztrade/pkg/core"
	. "github.com/ztrade/ztrade/pkg/event"
	"github.com/ztrade/ztrade/pkg/process/goscript/engine"

	log "github.com/sirupsen/logrus"
	. "github.com/ztrade/trademodel"
)

type scriptInfo struct {
	engine.Runner
	params common.ParamData
	wrap   *engine.EngineWrapper
}

type Status struct {
	Name   string
	Status int
	Msg    string
}

type GoEngine struct {
	BaseProcesser
	engine   *engine.EngineWrapper
	vms      map[string]*scriptInfo
	mutex    sync.Mutex
	started  int32
	statusCh chan *Status
}

func NewDefaultGoEngine() (s *GoEngine, err error) {
	return NewGoEngine("")
}
func NewGoEngine(symbol string) (s *GoEngine, err error) {
	s = new(GoEngine)
	s.Name = "multi_script"
	s.vms = make(map[string]*scriptInfo)
	s.engine = engine.NewEngineWrapper(&s.BaseProcesser, nil, symbol, "")
	return
}

func (s *GoEngine) SetStatusCh(ch chan *Status) {
	s.statusCh = ch
}

func (s *GoEngine) Init(bus *Bus) (err error) {
	s.BaseProcesser.Init(bus)
	s.Subscribe(EventCandle, s.onEventCandle)
	s.Subscribe(EventTrade, s.onEventTrade)
	s.Subscribe(EventPosition, s.onEventPosition)
	s.Subscribe(EventTradeMarket, s.onEventTradeMarket)
	s.Subscribe(EventDepth, s.onEventDepth)
	s.Subscribe(EventBalance, s.onEventBalance)
	return
}

func (s *GoEngine) Start() (err error) {
	atomic.StoreInt32(&s.started, 1)
	for k, v := range s.vms {
		tempEng := engine.EngineWrapper{EngineImpl: s.engine.EngineImpl, VmID: k}
		tempEng.Cb = s.updateScriptStatus
		v.wrap = &tempEng
		v.Init(&tempEng, v.params)
	}
	return
}

func (s *GoEngine) ScriptCount() int {
	return len(s.vms)
}

func (s *GoEngine) Stop() (err error) {
	return
}

func (s *GoEngine) RemoveScript(name string) (err error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.doRemoveScript(name)
}

func (s *GoEngine) doRemoveScript(name string) (err error) {
	vm, ok := s.vms[name]
	if !ok {
		log.Warnf("%s script not exist", name)
		return
	}
	vm.wrap.CleanMerges()
	delete(s.vms, name)
	return
}

func (s *GoEngine) AddScript(name, src, param string) (err error) {
	err = s.doAddScript(name, src, param)
	return
}
func (s *GoEngine) doAddScript(name, src, param string) (err error) {
	log.Info("GoEngine doAddScript:", name, src, param)
	s.mutex.Lock()
	defer s.mutex.Unlock()
	_, ok := s.vms[name]
	if ok {
		err = fmt.Errorf("%s script aleady exist", name)
		return
	}
	r, err := engine.NewRunner(src)
	if err != nil {
		err = fmt.Errorf("AddScript %s %s error: %w", name, src, err)
		return
	}
	paramInfo, err := r.Param()
	if err != nil {
		err = fmt.Errorf("AddScript %s %s get Params error: %w", name, src, err)
		return
	}
	paramData := make(common.ParamData)
	if param != "" {
		paramData, err = common.ParseParams(param, paramInfo)
		if err != nil {
			err = fmt.Errorf("AddScript %s %s ParseParams error: %w", name, src, err)
			return
		}
	}
	// var fnName string
	si := scriptInfo{Runner: r, params: paramData}
	s.vms[name] = &si
	isStart := atomic.LoadInt32(&s.started)
	if isStart == 1 {
		tempEng := engine.EngineWrapper{EngineImpl: s.engine.EngineImpl, VmID: name}
		tempEng.Cb = s.updateScriptStatus
		si.wrap = &tempEng
		si.Runner.Init(&tempEng, paramData)
	}
	return
}

func (s *GoEngine) onTrade(trade *Trade) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	for _, vm := range s.vms {
		vm.OnTrade(trade)
	}

}

func (s *GoEngine) onPosition(pos *Position) {
	log.Debug("on position:", pos.Hold)
	posHold, _ := s.engine.Position()
	if posHold == pos.Hold {
		return
	}
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.engine.UpdatePosition(pos.Hold, pos.Price)
	for _, vm := range s.vms {
		vm.OnPosition(pos.Hold, pos.Price)
	}
}

func (s *GoEngine) onBalance(balance float64) {
	if s.engine.Balance() == balance {
		return
	}
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.engine.UpdateBalance(balance)
}

func (s *GoEngine) onCandle(name, binSize string, candle *Candle) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	for _, vm := range s.vms {
		vm.OnCandle(candle)
	}
	s.engine.OnCandle(candle)
}

func (s *GoEngine) onTradeMarket(th *Trade) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	for _, vm := range s.vms {
		vm.OnTradeMarket(th)
	}
}

func (s *GoEngine) onDepth(depth *Depth) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	for _, vm := range s.vms {
		vm.OnDepth(depth)
	}
}

func (s *GoEngine) onEventCandle(e *Event) (err error) {
	ret, ok := e.GetData().(*Candle)
	if !ok {
		log.Errorf("onEventCandle type error: %##v", e.GetData())
		return
	}

	name := e.GetName()
	binSize := e.GetExtra().(string)
	if name == "recent" {
		ret.ID = -1
	}
	s.onCandle(name, binSize, ret)
	return
}

func (s *GoEngine) onEventTrade(e *Event) (err error) {
	tr, ok := e.GetData().(*Trade)
	if !ok {
		log.Errorf("onEventTrade type error: %##v", e.GetData())
		return
	}
	s.onTrade(tr)
	return
}

func (s *GoEngine) onEventPosition(e *Event) (err error) {
	pos, ok := e.GetData().(*Position)
	if !ok {
		log.Errorf("onEventPosition type error: %##v", e.GetData())
		return
	}
	s.onPosition(pos)
	return
}
func (s *GoEngine) onEventTradeMarket(e *Event) (err error) {
	th, ok := e.GetData().(*Trade)
	if !ok {
		log.Errorf("onEventTradeMarket type error: %##v", e.GetData())
		return
	}
	s.onTradeMarket(th)
	return
}

func (s *GoEngine) onEventDepth(e *Event) (err error) {
	depth, ok := e.GetData().(*Depth)
	if !ok {
		log.Errorf("onEventDepth type error: %##v", e.GetData())
		return
	}
	s.onDepth(depth)
	return
}

func (s *GoEngine) onEventBalance(e *Event) (err error) {
	balance, ok := e.GetData().(*Balance)
	if !ok {
		log.Errorf("onEventBalance type error: %##v", e.GetData())
		return
	}
	s.onBalance(balance.Balance)
	return
}

func (s *GoEngine) updateScriptStatus(name string, status int, msg string) {
	// call in script, no need lock
	switch status {
	case bengine.StatusRunning:
	case bengine.StatusSuccess, bengine.StatusFail:
		s.doRemoveScript(name)
	default:
		log.Errorf("GoEngine updateScriptStatus script %s unknown status: %d,", name, status)
	}
	if s.statusCh != nil {
		s.statusCh <- &Status{Name: name, Status: status, Msg: msg}
	}
}
