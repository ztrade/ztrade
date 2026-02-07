package exchange

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/ztrade/exchange"
	"github.com/ztrade/trademodel"
	. "github.com/ztrade/trademodel"
	. "github.com/ztrade/ztrade/pkg/core"
	. "github.com/ztrade/ztrade/pkg/event"
)

type OrderInfo struct {
	LocalID string
	Order
	Action TradeType
	Filled bool
}

type TradeExchange struct {
	BaseProcesser

	impl exchange.Exchange

	datas   chan interface{}
	actChan chan TradeAction

	orders          map[string]*OrderInfo
	localOrderIndex map[string]*OrderInfo
	ordersMutex     sync.RWMutex

	closeCh chan bool

	pos            Position
	positionUpdate int64
	exchangeName   string
	symbol         string

	candleParam CandleParam

	localStopOrder bool
	stopOrders     sync.Map
	stopped        int32
	stopCh         chan struct{}
}

func NewTradeExchange(exName string, impl exchange.Exchange, symbol string) *TradeExchange {
	te := new(TradeExchange)
	te.Name = fmt.Sprintf("exchange-%s", exName)
	te.exchangeName = exName
	te.impl = impl
	te.actChan = make(chan TradeAction, 10)
	te.orders = make(map[string]*OrderInfo)
	te.localOrderIndex = make(map[string]*OrderInfo)
	te.closeCh = make(chan bool)
	te.symbol = symbol
	te.datas = make(chan interface{}, 1024)
	te.stopCh = make(chan struct{})
	return te
}

func (b *TradeExchange) UseLocalStopOrder(enable bool) {
	b.localStopOrder = enable
	if enable {
		log.Warnf("%s TradeExchange use local stop order", b.impl.Info().Name)
	}
}

func (b *TradeExchange) Init(bus *Bus) (err error) {
	b.BaseProcesser.Init(bus)
	b.Subscribe(EventOrder, b.onEventOrder)
	b.Subscribe(EventWatch, b.onEventWatch)
	return
}

func (b *TradeExchange) Start() (err error) {
	atomic.StoreInt32(&b.stopped, 0)
	b.impl.Watch(exchange.WatchParam{Type: exchange.WatchTypeBalance}, func(data interface{}) {
		b.datas <- data
	})
	b.impl.Watch(exchange.WatchParam{Type: exchange.WatchTypePosition}, func(data interface{}) {
		b.datas <- data
	})
	b.impl.Watch(exchange.WatchParam{Type: exchange.WatchTypeTrade}, func(data interface{}) {
		b.datas <- data
	})
	err = b.impl.Start()
	if err != nil {
		return err
	}
	go b.recvDatas()
	go b.orderRoutine()
	return
}

func (b *TradeExchange) Stop() (err error) {
	if !atomic.CompareAndSwapInt32(&b.stopped, 0, 1) {
		return
	}
	close(b.stopCh)
	err = b.impl.Stop()
	return
}

func (b *TradeExchange) storeOrderInfo(localID string, od *Order, action TradeType) {
	oi := &OrderInfo{Order: *od, Action: action, LocalID: localID}
	b.ordersMutex.Lock()
	b.orders[od.OrderID] = oi
	b.localOrderIndex[localID] = oi
	b.ordersMutex.Unlock()
}

func (b *TradeExchange) getOrderInfo(orderID string) (*OrderInfo, bool) {
	b.ordersMutex.RLock()
	oi, ok := b.orders[orderID]
	b.ordersMutex.RUnlock()
	return oi, ok
}

func (b *TradeExchange) getLocalOrderInfo(localID string) (*OrderInfo, bool) {
	b.ordersMutex.RLock()
	oi, ok := b.localOrderIndex[localID]
	b.ordersMutex.RUnlock()
	return oi, ok
}

func (b *TradeExchange) enqueueAction(act TradeAction) {
	if atomic.LoadInt32(&b.stopped) == 1 {
		log.Warnf("TradeExchange stopped, ignore action: %s %s", act.ID, act.Action)
		return
	}
	select {
	case b.actChan <- act:
	case <-b.stopCh:
		log.Warnf("TradeExchange stopping, ignore action: %s %s", act.ID, act.Action)
	default:
		log.Errorf("TradeExchange action queue full, drop action: id=%s action=%s symbol=%s price=%f amount=%f", act.ID, act.Action, act.Symbol, act.Price, act.Amount)
	}
}

func (b *TradeExchange) recvDatas() {
	var posTime int64
	var o *OrderInfo
	bFirst := true
	var err error
	var tFirstLastStart int64
Out:
	for data := range b.datas {
		switch value := data.(type) {
		case *Candle:
			if bFirst {
				bFirst = false
				param := b.candleParam
				param.End = value.Time().Add(-1 * time.Second)
				tFirstLastStart, err = b.emitRecentCandles(param)
				if err != nil {
					log.Errorf("TradeExchange recv data load recent candles failed: %s", err.Error())
					// on error, skip dedup filter and forward all incoming candles
				} else if value.Start <= tFirstLastStart {
					continue
				}
			}
			b.SendWithExtra("candle", EventCandle, value, b.candleParam.BinSize)
		case *Balance:
			b.Send(b.exchangeName, EventBalance, value)
		case *Position:
			if value.Symbol != b.symbol {
				log.Infof("TradeExchange ignore event: %#v, exchange symbol: %s, data symbol: %s", value, b.symbol, value.Symbol)
				continue
			}
			b.pos = *value
			posTime = time.Now().Unix()
			atomic.StoreInt64(&b.positionUpdate, posTime)
			b.Send(value.Symbol, EventPosition, value)
		case *Order:
			if value.Symbol != b.symbol {
				log.Infof("TradeExchange ignore event: %#v, exchange symbol: %s, data symbol: %s", value, b.symbol, value.Symbol)
				continue
			}
			b.ordersMutex.Lock()
			o = b.orders[value.OrderID]
			if o == nil || o.Filled {
				b.ordersMutex.Unlock()
				continue Out
			}
			o.Order = *value
			if value.Status == OrderStatusFilled {
				o.Filled = true
			}
			b.ordersMutex.Unlock()
			if value.Status != OrderStatusFilled {
				continue Out
			}
			tr := Trade{ID: o.LocalID,
				Action: o.Action,
				Time:   o.Time,
				Price:  o.Price,
				Amount: o.Amount,
				Side:   o.Side,
				Remark: o.OrderID}
			b.Send(o.OrderID, EventTrade, &tr)
		case *Depth:
			b.Send(b.exchangeName, EventDepth, value)
		case *Trade:
			b.onEventTradeMarket(value)
			b.Send(b.exchangeName, EventTradeMarket, value)
		default:
			log.Errorf("unsupport exchange data: %##v", value)
		}
	}
}

func (b *TradeExchange) onEventCandleParam(e *Event) (err error) {
	wParam, ok := e.GetData().(*WatchParam)
	if !ok {
		err = fmt.Errorf("event not watch %s %#v", e.Name, e.Data)
		return
	}
	cParam, _ := wParam.Data.(*CandleParam)
	if cParam == nil {
		err = fmt.Errorf("event not CandleParam %s %#v", e.Name, e.Data)
		return
	}
	go b.emitCandles(*cParam)
	return
}

func (b *TradeExchange) onEventOrder(e *Event) (err error) {
	act := e.GetData().(*TradeAction)
	b.enqueueAction(*act)
	return
}

func (b *TradeExchange) onEventTradeMarket(trade *Trade) {
	if !b.localStopOrder || b.pos.Hold == 0 {
		return
	}
	var deleteOrders []string
	b.stopOrders.Range(func(key, value any) bool {
		id := key.(string)
		act := value.(TradeAction)
		if b.pos.Hold > 0 && act.Action == StopLong && trade.Price < act.Price {
			// do stop long
			newAct := TradeAction{
				ID:     id + "_stop",
				Action: CloseLong,
				Amount: act.Amount,
				Price:  act.Price,
				Time:   act.Time,
				Symbol: act.Symbol,
			}
			log.Infof("TradeEvent local stopLong order trigger: %#v", newAct)
			deleteOrders = append(deleteOrders, id)
			b.enqueueAction(newAct)
			return true
		}
		if b.pos.Hold < 0 && act.Action == StopShort && trade.Price > act.Price {
			// do stop short
			newAct := TradeAction{
				ID:     id + "_stop",
				Action: CloseShort,
				Amount: act.Amount,
				Price:  act.Price,
				Time:   act.Time,
				Symbol: act.Symbol,
			}
			log.Infof("TradeEvent local stopShort order trigger: %#v", newAct)
			deleteOrders = append(deleteOrders, id)
			b.enqueueAction(newAct)
			return true
		}
		return true
	})
	for _, v := range deleteOrders {
		b.stopOrders.Delete(v)
	}
}

func (b *TradeExchange) onEventWatch(e *Event) (err error) {
	if e.Name == "candle" {
		return b.onEventCandleParam(e)
	}

	param := e.GetData().(*WatchParam)
	switch param.Type {
	case EventTradeMarket:
		b.impl.Watch(exchange.WatchParam{Type: exchange.WatchTypeTradeMarket, Param: map[string]string{"symbol": param.Extra.(string)}}, func(data interface{}) {
			b.datas <- data
		})
	case EventDepth:
		b.impl.Watch(exchange.WatchParam{Type: exchange.WatchTypeDepth, Param: map[string]string{"symbol": param.Extra.(string)}}, func(data interface{}) {
			b.datas <- data
		})
	default:
		log.Errorf("TradeExchange OnEventWatch unsupport type: %s %##v", param.Type, param)
	}
	return
}

// orderRoutine process order routine
func (b *TradeExchange) orderRoutine() {
	var err error
	var ret interface{}
	var exist bool
	for {
		select {
		case <-b.stopCh:
			// drain remaining actions before exit
			for {
				select {
				case v := <-b.actChan:
					b.processAction(v, &err, &ret, &exist)
				default:
					return
				}
			}
		case v := <-b.actChan:
			b.processAction(v, &err, &ret, &exist)
		}
	}
}

func (b *TradeExchange) processAction(v TradeAction, errp *error, retp *interface{}, existp *bool) {
	var err error
	var ret interface{}
	// hook the stop order when localStopOrder enabled
	if v.Action.IsStop() && b.localStopOrder {
		b.stopOrders.Store(v.ID, v)
		return
	} else if v.Action == trademodel.CancelAll {
		b.cancelAllOrder()
		b.stopOrders = sync.Map{}
		return
	} else if v.Action == trademodel.CancelOne {
		_, exist := b.stopOrders.LoadAndDelete(v.ID)
		if exist {
			return
		}
		oi, ok := b.getLocalOrderInfo(v.ID)
		if !ok {
			log.Errorf("local order: %s not found", v.ID)
			return
		}
		_, err = doOrderWithRetry(10, func() (interface{}, error) {
			return b.impl.CancelOrder(&oi.Order)
		})
		if err != nil {
			log.Errorf("cancel order local %s, id %s failed: %s", oi.LocalID, oi.OrderID, err.Error())
		}
		return
	}
	ret, err = doOrderWithRetry(10, func() (interface{}, error) {
		order, e := b.impl.ProcessOrder(v)
		return order, e
	})
	if err == nil {
		od := ret.(*Order)
		b.storeOrderInfo(v.ID, od, v.Action)
	} else {
		tr := Trade{ID: v.ID,
			Action: v.Action,
			Time:   v.Time,
			Price:  v.Price,
			Amount: v.Amount,
			Remark: "failed:" + err.Error()}
		b.Send(v.ID, EventTrade, &tr)
	}
}

func (b *TradeExchange) cancelAllOrder() {
	ret, err := doOrderWithRetry(10, func() (interface{}, error) {
		orders, err := b.impl.CancelAllOrders()
		return orders, err
	})
	if err != nil {
		log.Errorf("cancel allorder error %s", err.Error())
		return
	}
	log.Info("cancel order:", ret)
}

func (b *TradeExchange) emitCandles(param CandleParam) {
	if param.BinSize != "1m" {
		log.Info("TradeExchange emit candle binsize not 1m:", param)
		return
	}
	watchParam := exchange.WatchCandle(param.Symbol, param.BinSize)
	b.candleParam = param
	err := b.impl.Watch(watchParam, func(data interface{}) {
		candle := data.(*Candle)
		b.datas <- candle
	})
	if err != nil {
		log.Errorf("emitCandles watchKline failed: %s", err.Error())
		return
	}
}

func (b *TradeExchange) emitRecentCandles(param CandleParam) (tLast int64, err error) {
	klines, errCh := exchange.KlineChan(b.impl, param.Symbol, param.BinSize, param.Start, param.End)
	for v := range klines {
		tLast = v.Start
		b.SendWithExtra("recent", EventCandle, v, param.BinSize)
	}
	err = <-errCh
	return
}
