package goscript

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
	"ztrade/pkg/common"
	. "ztrade/pkg/define"
	. "ztrade/pkg/event"

	"github.com/SuperGod/coinex"
	"github.com/SuperGod/indicator"
	. "github.com/SuperGod/trademodel"
	log "github.com/sirupsen/logrus"
)

type scriptInfo struct {
	*Runner
	params   ParamData
	binSizes []string
}

type GoEngine struct {
	BaseProcesser
	engine      *Engine
	vms         map[string]*scriptInfo
	scriptMutex sync.Mutex
	mutex       sync.Mutex
	pos         float64
	binSizes    []string
	started     int32
}

func NewDefaultGoEngine() (s *GoEngine, err error) {
	return NewGoEngine(common.DefaultBinSizes)
}
func NewGoEngine(binSizes string) (s *GoEngine, err error) {
	s = new(GoEngine)
	s.binSizes = common.ParseBinStrs(binSizes)
	s.Name = "multi_script"
	s.vms = make(map[string]*scriptInfo)
	s.engine = NewEngine(&s.BaseProcesser)
	return
}

func (s *GoEngine) Init(bus *Bus) (err error) {
	s.BaseProcesser.Init(bus)
	bus.Subscribe(EventCandle, s.onEventCandle)
	bus.Subscribe(EventTrade, s.onEventTrade)
	bus.Subscribe(EventPosition, s.onEventPosition)
	bus.Subscribe(EventTradeHistory, s.onEventTradeHistory)
	bus.Subscribe(EventDepth, s.onEventDepth)
	bus.Subscribe(EventBalance, s.onEventBalance)
	return
}

func (s *GoEngine) Start() (err error) {
	atomic.StoreInt32(&s.started, 1)
	for _, v := range s.vms {
		v.Init(s.engine, map[string]interface{}{})
	}
	return
}

func (s *GoEngine) Stop() (err error) {
	return
}

func (s *GoEngine) addOrder(price, amount float64, orderType TradeType) {
	// FixMe: in backtest, time may be the time of candle
	e := TradeAction{Action: orderType, Amount: amount, Price: price, Time: time.Now()}
	s.Send(EventOrder, EventOrder, e)
}

func (s *GoEngine) sendNotify(content, contentType string) {
	if contentType == "" {
		contentType = "text"
	}
	data := NotifyEvent{Type: contentType, Content: content}
	s.Send("notify", EventNotify, data)
}

func (s *GoEngine) addIndicator(name string, params ...int) (ind indicator.CommonIndicator) {
	var err error
	ind, err = indicator.NewCommonIndicator(name, params...)
	if err != nil {
		log.Errorf("GoEngine addIndicator failed %s %v", name, params)
		return nil
	}
	return
}

func (s *GoEngine) openLong(price, amount float64) {
	s.addOrder(price, amount, OpenLong)
}

func (s *GoEngine) openShort(price, amount float64) {
	s.addOrder(price, amount, OpenShort)
}
func (s *GoEngine) closeLong(price, amount float64) {
	s.addOrder(price, amount, CloseLong)
}

func (s *GoEngine) cancelAllOrder() {
	// e := TradeAction{Action: orderType, Amount: amount, Price: price, Time: time.Now()}
	s.Send(EventOrder, EventOrderCancelAll, nil)
}

func (s *GoEngine) closeShort(price, amount float64) {
	s.addOrder(price, amount, CloseShort)
}

func (s *GoEngine) stopShort(price, amount float64) {
	s.addOrder(price, amount, StopShort)
}

func (s *GoEngine) stopLong(price, amount float64) {
	s.addOrder(price, amount, StopLong)
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
	paramData := ParamData(param)
	r, err := NewRunner(src)
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

func (s *GoEngine) onTrades(trades []Trade) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	for _, v := range trades {
		for _, vm := range s.vms {
			vm.OnTrade(v)
		}
	}
}

func (s *GoEngine) onPosition(pos coinex.Position) {
	log.Debug("on position:", pos.Hold)
	if s.engine.pos == pos.Hold {
		return
	}
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.engine.updatePosition(pos.Hold, pos.Price)
	for _, vm := range s.vms {
		vm.OnPosition(pos.Hold, pos.Price)
	}
}

func (s *GoEngine) onBalance(balance float64) {
	if s.engine.balance == balance {
		return
	}
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.engine.updateBalance(balance)
}

func (s *GoEngine) onCandle(name, binSize string, candle Candle) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	for _, vm := range s.vms {
		vm.OnCandle(candle)
	}
	s.engine.onCandle(candle)
}

func (s *GoEngine) onTradeHistory(th Trade) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	for _, vm := range s.vms {
		vm.OnTradeHistory(th)
	}
}

func (s *GoEngine) onDepth(depth Depth) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	for _, vm := range s.vms {
		vm.OnDepth(depth)
	}
}

func (s *GoEngine) onEventCandle(e Event) (err error) {
	ret, ok := e.GetData().(*Candle)
	if !ok {
		log.Errorf("onEventCandle type error: %##v", e.GetData())
		return
	}
	cn := ParseCandleName(e.GetName())
	if cn.Name == "recent" {
		ret.ID = -1
	}
	s.onCandle(cn.Name, cn.BinSize, *ret)
	return
}

func (s *GoEngine) onEventTrade(e Event) (err error) {
	tr, ok := e.GetData().(*Trade)
	if !ok {
		log.Errorf("onEventTrade type error: %##v", e.GetData())
		return
	}
	s.onTrades([]Trade{*tr})
	return
}

func (s *GoEngine) onEventPosition(e Event) (err error) {
	pos, ok := e.GetData().(*coinex.Position)
	if !ok {
		log.Errorf("onEventPosition type error: %##v", e.GetData())
		return
	}
	s.onPosition(*pos)
	return
}
func (s *GoEngine) onEventTradeHistory(e Event) (err error) {
	th, ok := e.GetData().(*Trade)
	if !ok {
		log.Errorf("onEventTradeHistory type error: %##v", e.GetData())
		return
	}
	s.onTradeHistory(*th)
	return
}

func (s *GoEngine) onEventDepth(e Event) (err error) {
	depth, ok := e.GetData().(*Depth)
	if !ok {
		log.Errorf("onEventDepth type error: %##v", e.GetData())
		return
	}
	s.onDepth(*depth)
	return
}

func (s *GoEngine) onEventBalance(e Event) (err error) {
	balance, ok := e.GetData().(*BalanceInfo)
	if !ok {
		log.Errorf("onEventBalance type error: %##v", e.GetData())
		return
	}
	s.onBalance(balance.Balance)
	return
}
