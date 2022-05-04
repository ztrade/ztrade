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
}

type GoEngine struct {
	BaseProcesser
	engine   bengine.Engine
	vms      map[string]*scriptInfo
	mutex    sync.Mutex
	binSizes []string
	started  int32
}

func NewDefaultGoEngine() (s *GoEngine, err error) {
	return NewGoEngine(common.DefaultBinSizes)
}
func NewGoEngine(binSizes string) (s *GoEngine, err error) {
	s = new(GoEngine)
	s.binSizes = common.ParseBinStrs(binSizes)
	s.Name = "multi_script"
	s.vms = make(map[string]*scriptInfo)
	s.engine = engine.NewEngine(&s.BaseProcesser)
	return
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
	for _, v := range s.vms {
		v.Init(s.engine, v.params)
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
	_, ok := s.vms[name]
	if !ok {
		err = fmt.Errorf("%s script not  exist", name)
		return
	}
	delete(s.vms, name)
	return
}

func (s *GoEngine) AddScript(name, src string, param map[string]interface{}) (err error) {
	err = s.doAddScript(name, src, param)
	return
}
func (s *GoEngine) doAddScript(name, src string, param map[string]interface{}) (err error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	_, ok := s.vms[name]
	if ok {
		err = fmt.Errorf("%s script aleady exist", name)
		return
	}
	paramData := common.ParamData(param)
	r, err := engine.NewRunner(src)
	if err != nil {
		err = fmt.Errorf("AddScript %s %s error: %w", name, src, err)
		return
	}
	// var fnName string
	si := scriptInfo{Runner: r, params: paramData}
	s.vms[name] = &si
	isStart := atomic.LoadInt32(&s.started)
	if isStart == 1 {
		si.Runner.Init(s.engine, paramData)
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
