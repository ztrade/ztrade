package ctl

import (
	"fmt"
	"time"

	// . "github.com/ztrade/ztrade/pkg/define"

	"github.com/SuperGod/trademodel"
	"github.com/ztrade/ztrade/pkg/process/dbstore"
	"github.com/ztrade/ztrade/pkg/process/exchange"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type DataDownload struct {
	exchange string
	start    time.Time
	end      time.Time
	binSize  string
	symbol   string
	running  bool
	stop     chan bool
	bInit    bool
	db       *dbstore.DBStore
	cfg      *viper.Viper
	isAuto   bool
}

// NewDataDownload constructor of DataDownload
func NewDataDownloadAuto(cfg *viper.Viper, db *dbstore.DBStore, exchange, symbol, binSize string) (d *DataDownload) {
	d = new(DataDownload)
	d.cfg = cfg
	d.exchange = exchange
	d.symbol = symbol
	d.binSize = binSize
	d.db = db
	d.isAuto = true
	return
}

// NewDataDownload constructor of DataDownload
func NewDataDownload(cfg *viper.Viper, db *dbstore.DBStore, exchange, symbol, binSize string, start time.Time, end time.Time) (d *DataDownload) {
	d = new(DataDownload)
	d.cfg = cfg
	d.start = start
	d.end = end
	d.exchange = exchange
	d.symbol = symbol
	d.binSize = binSize
	d.db = db
	return
}

func (d *DataDownload) SetBinSize(binSize string) {
	d.binSize = binSize
}

// Start start backtest
func (d *DataDownload) Start() (err error) {
	d.running = true
	go d.Run()
	return
}

// Stop stop backtest
func (d *DataDownload) Stop() (err error) {
	d.stop <- true
	return
}
func (d *DataDownload) AutoRun() (err error) {
	tbl := d.db.GetKlineTbl(d.exchange, d.symbol, d.binSize)
	var invalidTime time.Time
	var tmTemp, start time.Time
	start = time.Now()
	tmTemp = tbl.GetNewest()
	if tmTemp == invalidTime {
		err = fmt.Errorf("no start found in db,you must set start time")
		return
	}
	// log.Info(k, "temp time newest:", tmTemp)
	if tmTemp.Sub(start) < 0 {
		start = tmTemp
	}
	end := time.Now()
	log.Debugf("autorun start:%s, end:%s", start, end)
	err = d.download(start, end)
	return
}

// Run run backtest and wait for finish
func (d *DataDownload) Run() (err error) {
	if d.isAuto {
		err = d.AutoRun()
	} else {
		err = d.download(d.start, d.end)
	}
	return
}

func (d *DataDownload) download(start, end time.Time) (err error) {
	log.Info("begin download candle:", start, end, d.symbol, d.binSize)
	ex, err := exchange.NewExchange(d.exchange, d.cfg, d.exchange, d.symbol)
	if err != nil {
		return
	}
	tbl := d.db.GetKlineTbl(d.exchange, d.symbol, d.binSize)
	klines, errChan := ex.KlineChan(start, end, d.symbol, d.binSize)
	var t time.Time
	cache := make([]interface{}, 1024)
	i := 0
	for v := range klines {
		cache[i] = v
		i++
		t = time.Now()
		if i >= 1024 {

			err = tbl.WriteDatas(cache)
			if err != nil {
				fmt.Printf("write %s - %s error: %s\n", cache[0].(*trademodel.Candle).Time(), cache[i-1].(*trademodel.Candle).Time(), err.Error())
				log.Errorf("%s write error: %s value: %#v %s", time.Now().Format(time.RFC3339), time.Since(t), v, err.Error())
				return
			} else {
				fmt.Printf("write %s - %s success\n", cache[0].(*trademodel.Candle).Time(), cache[i-1].(*trademodel.Candle).Time())
			}
			i = 0
		}

		// log.Infof("%s write finish: %s len: %d ", time.Now().Format(time.RFC3339), time.Since(t), len(v))
	}
	if i > 0 {

		err = tbl.WriteDatas(cache[0:i])
		if err != nil {
			fmt.Printf("write %s - %s error: %s\n", cache[0].(*trademodel.Candle).Time(), cache[i-1].(*trademodel.Candle).Time(), err.Error())
			log.Errorf("%s write error: %s value: %#v %s", time.Now().Format(time.RFC3339), time.Since(t), len(cache), err.Error())
			return
		} else {
			fmt.Printf("write %s - %s success\n", cache[0].(*trademodel.Candle).Time(), cache[i-1].(*trademodel.Candle).Time())
		}
	}
	err = <-errChan
	// log.Debugf("%s-%s %s %s %s data total %d stored\n", gStart,
	// 	lastStart,
	// 	d.source,
	// 	d.symbol,
	// 	d.binSize,
	// 	total)
	return
}

// Progress return the progress of current backtest
func (d *DataDownload) Progress() (progress int) {
	return d.Progress()
}

// Result return the result of current backtest
// must call after end of the backtest
func (d *DataDownload) Result() (err error) {

	return
}
