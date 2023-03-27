package exchange

import (
	"fmt"
	"sync/atomic"
	"time"

	"github.com/mitchellh/mapstructure"
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

	closeCh chan bool

	positionUpdate int64
	exchangeName   string
	symbol         string

	candleParam CandleParam
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
	return te
}

func (b *TradeExchange) Init(bus *Bus) (err error) {
	b.BaseProcesser.Init(bus)
	b.Subscribe(EventOrder, b.onEventOrder)
	b.Subscribe(EventWatch, b.onEventWatch)
	return
}

func (b *TradeExchange) Start() (err error) {
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
	err = b.impl.Stop()
	close(b.actChan)
	return
}

func (b *TradeExchange) recvDatas() {
	var ok bool
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
					log.Errorf("TradeExchange recv data:", err.Error())
					panic(err.Error())
				}
				if value.Start <= tFirstLastStart {
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
			posTime = time.Now().Unix()
			atomic.StoreInt64(&b.positionUpdate, posTime)
			b.Send(value.Symbol, EventPosition, value)
		case *Order:
			if value.Symbol != b.symbol {
				log.Infof("TradeExchange ignore event: %#v, exchange symbol: %s, data symbol: %s", value, b.symbol, value.Symbol)
				continue
			}
			o, ok = b.orders[value.OrderID]
			if !ok || o.Filled {
				continue Out
			}
			o.Order = *value
			if value.Status != OrderStatusFilled {
				continue Out
			}
			o.Filled = true
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
	var act TradeAction
	err = mapstructure.Decode(e.GetData(), &act)
	if err != nil {
		return
	}

	b.actChan <- act
	return
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
	for v := range b.actChan {
		if v.Action == trademodel.CancelAll {
			b.cancelAllOrder()
			continue
		} else if v.Action == trademodel.CancelOne {
			oi, ok := b.localOrderIndex[v.ID]
			if !ok {
				log.Errorf("local order: %s not found", v.ID)
				continue
			}
			_, err = doOrderWithRetry(10, func() (interface{}, error) {
				return b.impl.CancelOrder(&oi.Order)
			})
			if err != nil {
				log.Errorf("cancel order local %s, id %s failed: %s", oi.LocalID, oi.OrderID, err.Error())
			}
			continue
		}
		ret, err = doOrderWithRetry(10, func() (interface{}, error) {
			order, e := b.impl.ProcessOrder(v)
			return order, e
		})
		if err == nil {
			od := ret.(*Order)
			oi := &OrderInfo{Order: *od, Action: v.Action, LocalID: v.ID}
			b.orders[od.OrderID] = oi
			b.localOrderIndex[v.ID] = oi
		} else {
			tr := Trade{ID: v.ID,
				Action: v.Action,
				Time:   v.Time,
				Price:  v.Price,
				Amount: v.Amount,
				// Side:   v.Action,
				Remark: "failed:" + err.Error()}
			b.Send(v.ID, EventTrade, &tr)
		}

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
		log.Errorf("emitCandles wathKline failed:", err.Error())
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
