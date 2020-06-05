package ctl

import (
	"errors"
	"fmt"
	"sync"
	"time"
	. "ztrade/pkg/define"
	"ztrade/pkg/event"
	"ztrade/pkg/process/exchange"
	"ztrade/pkg/process/goscript"
	"ztrade/pkg/process/rpt"
	"ztrade/pkg/process/wxworkbot"

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
	rpt          Reporter
	proc         *event.Processers
	engine       Scripter
	r            *rpt.RPTProcesser
	wg           sync.WaitGroup
}

// NewTrade constructor of Trade
func NewTrade(exchange, symbol string) (b *Trade, err error) {
	b = new(Trade)
	b.exchangeName = exchange
	b.symbol = symbol
	b.exchangeType = cfg.GetString(fmt.Sprintf("exchanges.%s.type", b.exchangeName))
	gEngine, err := goscript.NewDefaultGoEngine()
	if err != nil {
		return
	}
	b.engine = gEngine
	// fmt.Println("exchange type:", b.exchangeType, fmt.Sprintf("%s.type", b.exchangeName))
	return
}

func (b *Trade) SetReporter(rpt Reporter) {
	b.rpt = rpt
}

func (b *Trade) AddScript(name, scriptFile string, param map[string]interface{}) (err error) {
	err = b.engine.AddScript(name, scriptFile, param)
	return
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
	if b.exchangeType != "bitmex" {
		err = fmt.Errorf("unsupport exchange: %s", b.exchangeType)
		return
	}
	bm, err := exchange.NewBitmexTradeWithSymbol(cfg, b.exchangeName, b.symbol)
	if err != nil {
		err = fmt.Errorf("creat bitmex trade failed:%s", err.Error())
		return
	}
	notify, err := wxworkbot.NewWXWork(true)
	if err != nil {
		log.Errorf("creat wxworkbot failed:%s", err.Error())
		err = nil
	}
	b.proc = event.NewProcessers()
	procs := []event.Processer{param, bm, b.engine}
	if notify != nil {
		procs = append(procs, notify)
	}
	if b.rpt != nil {
		r := NewRpt(b.rpt)
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
		Start: time.Now(),
		// End:     b.end,
		Symbol:  b.symbol,
		BinSize: "1m",
	}
	log.Info("real trade candle param:", candleParam)
	param.Send("trade", EventCandleParam, candleParam)
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
