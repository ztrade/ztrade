package vex

import (
	"container/list"
	"fmt"
	"math"
	"reflect"
	"sync"
	"time"

	"github.com/ztrade/base/common"
	"github.com/ztrade/trademodel"
	. "github.com/ztrade/ztrade/pkg/core"
	. "github.com/ztrade/ztrade/pkg/event"

	log "github.com/sirupsen/logrus"
	. "github.com/ztrade/trademodel"
)

// VExchange Virtual exchange impl FuturesBaseExchanger
type VExchange struct {
	BaseProcesser
	candle   *Candle
	trades   []Trade
	orders   *list.List
	position float64
	symbol   string
	balance  *common.VBalance
	// order index in same candle
	orderIndex int
	orderMutex sync.Mutex
}

func NewVExchange(symbol string) *VExchange {
	ex := new(VExchange)
	ex.Name = "VExchange"
	ex.orders = list.New()
	ex.symbol = symbol
	ex.balance = common.NewVBalance()
	return ex
}

func (b *VExchange) Init(bus *Bus) (err error) {
	b.BaseProcesser.Init(bus)
	b.Subscribe(EventCandle, b.onEventCandle)
	b.Subscribe(EventOrder, b.onEventOrder)
	b.Subscribe(EventBalanceInit, b.onEventBalanceInit)
	return
}

func (ex *VExchange) Start() (err error) {
	ex.Send(ex.symbol, EventBalance, &Balance{Balance: ex.balance.Get()})
	return
}
func (ex *VExchange) processCandle(candle Candle) (err error) {
	if ex.orders.Len() == 0 {
		return
	}
	ex.orderMutex.Lock()
	defer ex.orderMutex.Unlock()
	var posChange bool
	var deleteElems []*list.Element
	virtualTime := candle.Time()
	var trades []*Event
	var pos Position
	var orderFilled bool
	var side string
	var price float64
	for elem := ex.orders.Front(); elem != nil; elem = elem.Next() {
		orderFilled = false
		v, ok := elem.Value.(TradeAction)
		if !ok {
			log.Errorf("order items type error:%##v", elem.Value)
			continue
		}
		if !v.Action.IsOpen() {
			// stop order not works if position is zero
			if ex.position == 0 {
				continue
			} else if ex.position > 0 && v.Action.IsLong() {
				continue
			} else if ex.position < 0 && !v.Action.IsLong() {
				continue
			}
		}
		// order can only be filled after next candle
		price = v.Price

		switch v.Action {
		case StopShort:
			if v.Price <= candle.High {
				side = "buy"
				orderFilled = true
				if v.Price < candle.Low {
					price = candle.Low
				}
			}
		case StopLong:
			if v.Price >= candle.Low {
				side = "sell"
				orderFilled = true
				if v.Price > candle.High {
					price = candle.High
				}
			}
		case OpenLong, CloseShort:
			if v.Price >= candle.Low {
				side = "buy"
				orderFilled = true
				if v.Price > candle.High {
					price = candle.High
				}
			}
		case OpenShort, CloseLong:
			if v.Price <= candle.High {
				side = "sell"
				orderFilled = true
				if v.Price < candle.Low {
					price = candle.Low
				}
			}
		default:
			log.Warnf("unsupport ActionType: %s", v.Action.String())
			continue
		}

		//		fmt.Println("action:", v.Action.String(), v.Price, candle.High, candle.Low, candle.Time(), orderFilled)
		if !orderFilled {
			continue
		}

		virtualTime = virtualTime.Add(time.Second)
		tr := Trade{ID: fmt.Sprintf("%d", len(ex.trades)),
			Action: v.Action,
			Time:   virtualTime,
			Price:  price,
			Amount: v.Amount,
			Side:   side,
			Remark: ""}
		// fix size
		_, _, err = ex.balance.AddTrade(tr)
		if err != nil {
			// log.Errorf("vexchange balance AddTrade error:%s %f %f", err.Error(), v.Price, v.Amount)
			return
		}
		ex.trades = append(ex.trades, tr)
		tradeEvent := ex.CreateEvent("trade", EventTrade, &tr)
		trades = append(trades, tradeEvent)

		posChange = true
		pos.Price = tr.Price
		deleteElems = append(deleteElems, elem)
	}
	for _, v := range deleteElems {
		ex.orders.Remove(v)
	}
	// keep trade time order
	if len(trades) != 0 {
		for i := len(trades) - 1; i >= 0; i-- {
			ex.Bus.Send(trades[i])
		}
	}
	if posChange {
		ex.position = ex.balance.Pos()
		pos.Symbol = ex.symbol
		pos.Hold = ex.position
		//		ex.Send(ex.symbol, EventCurPosition, pos)
		ex.Send(ex.symbol, EventPosition, &pos)
		if pos.Hold == 0 {
			ex.Send(ex.symbol, EventBalance, &Balance{Currency: ex.symbol, Balance: ex.balance.Get()})
		}
	}
	return nil
}

func (ex *VExchange) onEventCandle(e *Event) (err error) {
	candle, ok := e.GetData().(*Candle)
	if !ok {
		err = fmt.Errorf("VExchange candle type error:%s", reflect.TypeOf(e.GetData()))
		return
	}
	// fmt.Println("candle:", e.Name, e.GetType(), e.GetData())
	binSize := e.GetExtra().(string)
	if binSize != "1m" {
		return
	}

	ex.candle = candle
	ex.orderIndex = 0
	err = ex.processCandle(*candle)
	return
}

func (ex *VExchange) onEventOrder(e *Event) (err error) {
	ex.orderMutex.Lock()
	defer ex.orderMutex.Unlock()
	act := e.GetData().(*TradeAction)
	if act == nil {
		log.Errorf("decode tradeaction error: %##v", e.GetData())
		return
	}
	if act.Action == trademodel.CancelAll {
		ex.orders = list.New()
		return
	}
	if ex.candle != nil {
		act.Time = ex.candle.Time().Add(time.Second * time.Duration(ex.orderIndex))
		if act.Action == StopLong && act.Price >= ex.candle.Close {
			log.Warnf("invalid stop long order,action: %#v, candle: %s", *act, *ex.candle)
			return
		} else if act.Action == StopShort && act.Price <= ex.candle.Close {
			log.Warnf("invalid stop short order,action: %#v, candle: %s", *act, *ex.candle)
			return
		}
	}
	ex.orderIndex++
	ex.orders.PushBack(*act)
	return
}

func (ex *VExchange) onEventBalanceInit(e *Event) (err error) {
	balance := e.GetData().(*BalanceInfo)
	ex.balance.Set(balance.Balance)
	ex.balance.SetFee(balance.Fee)
	ex.Send(ex.symbol, EventBalance, &Balance{Currency: ex.symbol, Balance: ex.balance.Get()})
	return
}

func (ex *VExchange) CloseAll() (err error) {
	if ex.position == 0 {
		return
	}
	virtualTime := ex.candle.Time().Add(time.Second)
	var tr Trade
	if ex.position > 0 {
		tr = Trade{ID: fmt.Sprintf("%d", len(ex.trades)),
			Action: CloseLong,
			Time:   virtualTime,
			Price:  ex.candle.Close,
			Amount: math.Abs(ex.position),
			Side:   "sell",
			Remark: ""}
	} else {
		tr = Trade{ID: fmt.Sprintf("%d", len(ex.trades)),
			Action: CloseShort,
			Time:   virtualTime,
			Price:  ex.candle.Close,
			Amount: math.Abs(ex.position),
			Side:   "buy",
			Remark: ""}
	}
	tradeEvent := ex.CreateEvent("trade", EventTrade, &tr)
	ex.Bus.Send(tradeEvent)
	_, _, err = ex.balance.AddTrade(tr)
	if err != nil {
		log.Errorf("vexchange CloseALll balance AddTrade error:%s %f %f", err.Error(), tr.Price, tr.Amount)
		return
	}
	ex.position = ex.balance.Pos()
	var pos Position
	pos.Symbol = ex.symbol
	pos.Hold = ex.position
	//		ex.Send(ex.symbol, EventCurPosition, pos)
	ex.Send(ex.symbol, EventPosition, &pos)
	if pos.Hold == 0 {
		ex.Send(ex.symbol, EventBalance, &Balance{Currency: ex.symbol, Balance: ex.balance.Get()})
	}
	return
}
