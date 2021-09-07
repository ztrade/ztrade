package ctl

import (
	"encoding/json"
	"time"

	"github.com/ztrade/base/common"
	. "github.com/ztrade/ztrade/pkg/define"
	"github.com/ztrade/ztrade/pkg/event"
	"github.com/ztrade/ztrade/pkg/process/dbstore"
	"github.com/ztrade/ztrade/pkg/process/rpt"
	"github.com/ztrade/ztrade/pkg/process/vex"

	// . "github.com/SuperGod/trademodel"
	log "github.com/sirupsen/logrus"
)

type Backtest struct {
	progress    int
	exchange    string
	symbol      string
	paramData   common.ParamData
	start       time.Time
	end         time.Time
	running     bool
	stop        chan bool
	db          *dbstore.DBStore
	scriptFile  string
	rpt         rpt.Reporter
	balanceInit float64
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
	b.paramData = make(common.ParamData)
	if param != "" {
		err = json.Unmarshal([]byte(param), &b.paramData)
	}
	return
}

func (b *Backtest) SetBalanceInit(balanceInit float64) {
	b.balanceInit = balanceInit
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
	tbl := b.db.GetKlineTbl(b.exchange, b.symbol, bSize)
	tbl.SetLoadDataMode(true)
	tbl.SetCloseCh(closeCh)
	ex := vex.NewVExchange(b.symbol)
	engine, err := NewScript(b.scriptFile, b.paramData)
	if err != nil {
		return
	}
	r := rpt.NewRpt(b.rpt)
	processers := event.NewProcessers()
	processers.Add(param)
	processers.Add(tbl)
	processers.Add(ex)
	processers.Add(engine)
	processers.Add(r)
	processers.Start()

	param.Send("balance_init", EventBalanceInit, BalanceInfo{Balance: b.balanceInit})
	candleParam := CandleParam{
		Start:   b.start,
		End:     b.end,
		Symbol:  b.symbol,
		BinSize: bSize,
	}
	log.Info("backtest candle param:", candleParam)
	param.Send("load_candle", EventCandleParam, candleParam)
	// TODO wait for finish
	<-closeCh
	processers.WaitClose(time.Second * 10)
	return
}

// Progress return the progress of current backtest
func (b *Backtest) Progress() (progress int) {
	return b.Progress()
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
