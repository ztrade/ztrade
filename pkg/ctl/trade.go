package ctl

import (
	"errors"
	"fmt"
	"sync"
	"time"

	. "github.com/ztrade/ztrade/pkg/core"
	"github.com/ztrade/ztrade/pkg/event"
	"github.com/ztrade/ztrade/pkg/process/exchange"
	"github.com/ztrade/ztrade/pkg/process/goscript"
	"github.com/ztrade/ztrade/pkg/process/rpt"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

var (
	cfg *viper.Viper
)

func SetConfig(c *viper.Viper) {
	cfg = c
}

// Trade trade with multi scripts
type Trade struct {
	exchangeType string
	exchangeName string
	symbol       string
	running      bool
	stop         chan bool
	rpt          rpt.Reporter
	proc         *event.Processers
	engine       *goscript.GoEngine
	r            *rpt.Rpt
	wg           sync.WaitGroup
	loadRecent   time.Duration
	statusCh     chan *goscript.Status
}

// NewTrade constructor of Trade
func NewTrade(exchange, symbol string) (b *Trade, err error) {
	b = new(Trade)
	b.exchangeName = exchange
	b.symbol = symbol
	b.exchangeType = cfg.GetString(fmt.Sprintf("exchanges.%s.type", b.exchangeName))
	gEngine, err := goscript.NewGoEngine(symbol)
	if err != nil {
		return
	}
	b.engine = gEngine
	b.loadRecent = time.Hour * 24
	return
}

func (b *Trade) SetLoadRecent(recent time.Duration) {
	b.loadRecent = recent
}

func (b *Trade) SetStatusCh(ch chan *goscript.Status) {
	b.engine.SetStatusCh(ch)
}

func (b *Trade) SetReporter(rpt rpt.Reporter) {
	b.rpt = rpt
}

func (b *Trade) AddScript(name, scriptFile, param string) (err error) {
	err = b.engine.AddScript(name, scriptFile, param)
	return
}

func (b *Trade) ScriptCount() int {
	return b.engine.ScriptCount()
}

func (b *Trade) RemoveScript(name string) (err error) {
	if !b.running {
		err = errors.New("Trade is not working,must start it first")
		return
	}
	err = b.engine.RemoveScript(name)
	return
}

// Start start backtest
func (b *Trade) Start() (err error) {
	if b.running {
		return
	}
	b.running = true
	err = b.init()
	if err != nil {
		b.running = false
		return
	}
	b.wg.Add(1)
	go b.Run()
	return
}

// Stop stop backtest
func (b *Trade) Stop() (err error) {
	b.proc.Stop()
	b.stop <- true
	return
}

func (b *Trade) init() (err error) {
	b.stop = make(chan bool)
	param := event.NewBaseProcesser("param")
	ex, err := exchange.GetTradeExchange(b.exchangeType, cfg, b.exchangeName, b.symbol)
	if err != nil {
		err = fmt.Errorf("creat exchange trade %s failed:%s", b.exchangeName, err.Error())
		return
	}
	// notify, err := wxworkbot.NewWXWork(true)
	// if err != nil {
	// log.Errorf("creat wxworkbot failed:%s", err.Error())
	// err = nil
	// }
	b.proc = event.NewProcessers()
	procs := []event.Processer{param, ex, b.engine}
	// if notify != nil {
	// procs = append(procs, notify)
	// }
	if b.rpt != nil {
		r := rpt.NewRpt(b.rpt)
		procs = append(procs, r)
	}

	err = b.proc.Adds(procs...)
	if err != nil {
		log.Error("add processers error:", err.Error())
		return
	}
	err = b.proc.Start()
	if err != nil {
		log.Error("start processers error:", err.Error())
		return
	}
	candleParam := CandleParam{
		Start:   time.Now().Add(-1 * b.loadRecent),
		Symbol:  b.symbol,
		BinSize: "1m",
	}
	log.Info("real trade candle param:", candleParam)
	param.Send("candle", EventWatch, NewWatchCandle(&candleParam))

	log.Info("real trade watch trade_market")
	param.Send("trade", EventWatch, &WatchParam{Type: EventTradeMarket, Extra: b.symbol, Data: map[string]interface{}{"name": "market"}})
	log.Info("real trade watch depth")
	param.Send("trade", EventWatch, &WatchParam{Type: EventDepth, Extra: b.symbol, Data: map[string]interface{}{"name": "depth"}})
	return
}

func (b *Trade) Wait() (err error) {
	b.wg.Wait()
	return
}

func (b *Trade) Run() (err error) {
	defer b.wg.Done()
	// TODO wait for finish
	<-b.stop
	b.proc.WaitClose(time.Second * 10)
	return
}
