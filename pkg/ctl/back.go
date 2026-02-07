package ctl

import (
	"errors"
	"sync"
	"time"

	"github.com/ztrade/base/common"
	. "github.com/ztrade/ztrade/pkg/core"
	"github.com/ztrade/ztrade/pkg/event"
	"github.com/ztrade/ztrade/pkg/process/dbstore"
	"github.com/ztrade/ztrade/pkg/process/rpt"
	"github.com/ztrade/ztrade/pkg/process/vex"

	// . "github.com/ztrade/trademodel"
	log "github.com/sirupsen/logrus"
)

type Backtest struct {
	progress    int
	exchange    string
	symbol      string
	paramData   string
	start       time.Time
	end         time.Time
	running     bool
	stop        chan bool
	db          *dbstore.DBStore
	scriptFile  string
	rpt         rpt.Reporter
	balanceInit float64
	loadDBOnce  int
	fee         float64
	lever       float64

	closeAllWhenFinished bool
}

// NewBacktest constructor of Backtest
func NewBacktest(db *dbstore.DBStore, exchange, symbol, param string, start time.Time, end time.Time) (b *Backtest, err error) {
	b = new(Backtest)
	b.start = start
	b.end = end
	b.exchange = exchange
	b.symbol = symbol
	b.db = db
	b.balanceInit = 100000
	b.loadDBOnce = 50000
	b.paramData = param
	b.stop = make(chan bool, 1)
	return
}

func (b *Backtest) CloseAllWhenFinished(bCloseAll bool) {
	b.closeAllWhenFinished = bCloseAll
}

func (b *Backtest) SetLoadDBOnce(loadOnce int) {
	b.loadDBOnce = loadOnce
}

func (b *Backtest) SetBalanceInit(balanceInit, fee float64) {
	b.balanceInit = balanceInit
	b.fee = fee
}

func (b *Backtest) SetLever(lever float64) {
	b.lever = lever
}

func (b *Backtest) SetScript(scriptFile string) {
	b.scriptFile = scriptFile
}

func (b *Backtest) SetReporter(rpt rpt.Reporter) {
	b.rpt = rpt
}

// Start start backtest
func (b *Backtest) Start() (err error) {
	b.running = true
	go b.Run()
	return
}

// Stop stop backtest
func (b *Backtest) Stop() (err error) {
	b.stop <- true
	return
}

// Run !TODO need support multi binsizes
func (b *Backtest) Run() (err error) {
	defer func() {
		b.running = false
	}()
	closeCh := make(chan bool)
	param := event.NewBaseProcesser("param")
	bSize := "1m"
	tbl := b.db.NewKlineTbl(b.exchange, b.symbol, bSize)
	tbl.SetLoadOnce(b.loadDBOnce)
	tbl.SetLoadDataMode(true)
	tbl.SetCloseCh(closeCh)
	ex := vex.NewVExchange(b.symbol)
	engine, err := NewScript(b.scriptFile, b.paramData, b.symbol)
	if err != nil {
		return
	}
	r := rpt.NewRpt(b.rpt)
	processers := event.NewSyncProcessers()
	processers.Add(param)
	processers.Add(tbl)
	processers.Add(ex)
	processers.Add(engine)
	processers.Add(r)

	var stopOnce sync.Once
	errorCh := make(chan bool)
	processers.SetErrorCallback(func(err error) {
		if errors.Is(err, common.ErrNoBalance) {
			stopOnce.Do(func() {
				log.Errorf("got error: %s, just exit", err.Error())
				processers.Stop()
				errorCh <- true
			})
		}
	})

	err = processers.Start()
	if err != nil {
		return
	}

	param.Send("balance_init", EventBalanceInit, &BalanceInfo{Balance: b.balanceInit, Fee: b.fee})
	param.Send("risk_init", EventRiskLimit, &RiskLimit{Lever: b.lever})
	candleParam := CandleParam{
		Start:   b.start,
		End:     b.end,
		Symbol:  b.symbol,
		BinSize: bSize,
	}

	log.Info("backtest candle param:", candleParam)
	param.Send("load_candle", EventWatch, NewWatchCandle(&candleParam))
	// TODO wait for finish
	select {
	case <-closeCh:
	case <-errorCh:
		// FIXME: tbl maybe not close
	}
	if b.closeAllWhenFinished {
		time.Sleep(time.Second * 10)
		ex.CloseAll()
	}
	processers.WaitClose(time.Second * 10)
	return
}

// Progress return the progress of current backtest
func (b *Backtest) Progress() (progress int) {
	return b.progress
}

// IsRunning return if the backtest is running
func (b *Backtest) IsRunning() (ret bool) {
	return b.running
}

// Result return the result of current backtest
// must call after end of the backtest
func (b *Backtest) Result() (err error) {

	return
}
