package ctl

import (
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/ztrade/exchange"
	"github.com/ztrade/trademodel"
	"github.com/ztrade/ztrade/pkg/process/dbstore"
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
	d.stop = make(chan bool, 1)
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
	d.stop = make(chan bool, 1)
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
	select {
	case d.stop <- true:
	default:
	}
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
		start = tmTemp.Add(-time.Minute)
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
	exchangeType := d.cfg.GetString(fmt.Sprintf("exchanges.%s.type", d.exchange))
	fmt.Println(d.exchange, exchangeType)
	ex, err := exchange.NewExchange(exchangeType, exchange.WrapViper(d.cfg), d.exchange)
	if err != nil {
		return
	}
	tbl := d.db.GetKlineTbl(d.exchange, d.symbol, d.binSize)
	klines, errChan := exchange.KlineChan(ex, d.symbol, d.binSize, start, end)
	var t time.Time
	cache := make([]interface{}, 1024)
	i := 0

	for {
		select {
		case <-d.stop:
			err = fmt.Errorf("download stopped")
			return
		case v, ok := <-klines:
			if !ok {
				goto flush
			}
			cache[i] = v
			i++
			t = time.Now()
			if i >= len(cache) {
				err = tbl.WriteDatas(cache)
				if err != nil {
					fmt.Printf("write %s - %s error: %s\n", cache[0].(*trademodel.Candle).Time(), cache[i-1].(*trademodel.Candle).Time(), err.Error())
					log.Errorf("%s write error: %s value: %#v %s", time.Now().Format(time.RFC3339), time.Since(t), v, err.Error())
					return
				}
				fmt.Printf("write %s - %s success\n", cache[0].(*trademodel.Candle).Time(), cache[i-1].(*trademodel.Candle).Time())
				i = 0
			}
		}
	}

flush:
	if i > 0 {
		err = tbl.WriteDatas(cache[0:i])
		if err != nil {
			fmt.Printf("write %s - %s error: %s\n", cache[0].(*trademodel.Candle).Time(), cache[i-1].(*trademodel.Candle).Time(), err.Error())
			log.Errorf("%s write error: %s value: %#v %s", time.Now().Format(time.RFC3339), time.Since(t), len(cache), err.Error())
			return
		}
		fmt.Printf("write %s - %s success\n", cache[0].(*trademodel.Candle).Time(), cache[i-1].(*trademodel.Candle).Time())
	}

	err = <-errChan
	return
}

// Progress return the progress of current backtest
func (d *DataDownload) Progress() (progress int) {
	return 0
}

// Result return the result of current backtest
// must call after end of the backtest
func (d *DataDownload) Result() (err error) {

	return
}
