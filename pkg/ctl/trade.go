package ctl

import (
	"errors"
	"fmt"
	"sync"
	"time"

	zexchange "github.com/ztrade/exchange"
	. "github.com/ztrade/ztrade/pkg/core"
	"github.com/ztrade/ztrade/pkg/event"
	"github.com/ztrade/ztrade/pkg/process/exchange"
	"github.com/ztrade/ztrade/pkg/process/goscript"
	"github.com/ztrade/ztrade/pkg/process/notify"
	"github.com/ztrade/ztrade/pkg/process/risk"
	"github.com/ztrade/ztrade/pkg/process/rpt"

	log "github.com/sirupsen/logrus"
)

var (
	globalCfg zexchange.Config
)

// SetConfig sets the global config for backward compatibility.
// Prefer using NewTradeWithConfig for explicit config injection.
func SetConfig(c zexchange.Config) {
	globalCfg = c
}

// Trade trade with multi scripts
type Trade struct {
	cfg          zexchange.Config
	exchangeType string
	exchangeName string
	symbol       string
	running      bool
	stop         chan bool
	errorCh      chan error
	rpt          rpt.Reporter
	proc         *event.Processers
	engine       *goscript.GoEngine
	wg           sync.WaitGroup
	loadRecent   time.Duration
	riskConfig   *risk.RiskConfig
}

// NewTrade constructor of Trade, uses global config set by SetConfig.
// Prefer NewTradeWithConfig for explicit config injection.
func NewTrade(exchange, symbol string) (b *Trade, err error) {
	return NewTradeWithConfig(globalCfg, exchange, symbol)
}

// NewTradeWithConfig constructor of Trade with explicit config injection.
func NewTradeWithConfig(cfg zexchange.Config, exchange, symbol string) (b *Trade, err error) {
	if cfg == nil {
		err = errors.New("config is not initialized, please set --config or put ztrade.yaml in config paths")
		return
	}
	b = new(Trade)
	b.cfg = cfg
	b.exchangeName = exchange
	b.symbol = symbol
	b.exchangeType = b.cfg.GetString(fmt.Sprintf("exchanges.%s.type", b.exchangeName))
	if b.exchangeType == "" {
		err = fmt.Errorf("exchange %s type is empty in config", b.exchangeName)
		return
	}
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

func (b *Trade) SetRiskConfig(cfg *risk.RiskConfig) {
	b.riskConfig = cfg
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
	if b.proc != nil {
		err = b.proc.Stop()
	}
	select {
	case b.stop <- true:
	default:
	}
	return
}

func (b *Trade) init() (err error) {
	b.stop = make(chan bool, 1)
	b.errorCh = make(chan error, 1)
	param := event.NewBaseProcesser("param")
	ex, err := exchange.GetTradeExchange(b.exchangeType, b.cfg, b.exchangeName, b.symbol)
	if err != nil {
		err = fmt.Errorf("creat exchange trade %s failed:%s", b.exchangeName, err.Error())
		return
	}
	notify, err := notify.NewNotify(b.cfg)
	if err != nil {
		log.Errorf("creat notify failed:%s", err.Error())
		err = nil
	}
	b.proc = event.NewProcessers()
	procs := []event.Processer{param, ex}
	if b.riskConfig != nil {
		rm := risk.NewRiskManager(b.symbol, *b.riskConfig)
		procs = append(procs, rm)
	}
	procs = append(procs, b.engine)
	if notify != nil {
		procs = append(procs, notify)
	}
	if b.rpt != nil {
		r := rpt.NewRpt(b.rpt)
		procs = append(procs, r)
	}

	err = b.proc.Adds(procs...)
	if err != nil {
		log.Error("add processers error:", err.Error())
		return
	}
	b.proc.SetErrorCallback(func(err error) {
		log.Errorf("Trade processer error: %s", err.Error())
		select {
		case b.errorCh <- err:
		default:
		}
	})
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
	param.Send("depth", EventWatch, &WatchParam{Type: EventDepth, Extra: b.symbol, Data: map[string]interface{}{"name": "depth"}})
	return
}

func (b *Trade) Wait() (err error) {
	b.wg.Wait()
	return
}

func (b *Trade) Run() (err error) {
	defer b.wg.Done()
	select {
	case <-b.stop:
		log.Info("Trade received stop signal")
	case err = <-b.errorCh:
		log.Errorf("Trade exiting due to error: %v", err)
		b.proc.Stop()
	}
	b.proc.WaitClose(time.Second * 10)
	b.running = false
	return
}
