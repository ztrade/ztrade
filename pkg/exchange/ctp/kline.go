package ctp

import (
	"time"

	. "github.com/ztrade/trademodel"
	. "github.com/ztrade/ztrade/pkg/core"
)

type CTPKline struct {
	cur       *Candle
	datas     chan *ExchangeData
	symbol    string
	lastTrade time.Time
	close     chan bool
}

func newCTPKline(datas chan *ExchangeData, symbol string, closed chan bool) *CTPKline {
	k := new(CTPKline)
	k.symbol = symbol
	k.datas = datas
	k.close = closed
	go k.loop()
	return k
}

func (k *CTPKline) loop() {
	ticker := time.NewTicker(time.Minute)
	for {
		select {
		case <-ticker.C:
			if time.Since(k.lastTrade) > time.Minute {
				k.Flush()
			}
		case <-k.close:
			return
		}

	}
}

func (k *CTPKline) Update(t time.Time, price, volume, turnover float64) {
	if k.cur == nil || t.Sub(k.cur.Time()) >= time.Minute {
		k.lastTrade = time.Now()
		k.sendCandle(k.cur)
		tStart := (t.Unix() / 60) * 60
		k.cur = &Candle{
			ID:       0,
			Start:    tStart,
			Open:     price,
			High:     price,
			Low:      price,
			Close:    price,
			Turnover: turnover,
			Volume:   volume,
			Trades:   int64(volume),
		}
		return
	}
	k.cur.Close = price
	k.cur.Volume += volume
	k.cur.Turnover += turnover
	k.cur.Trades = int64(k.cur.Volume)

	if price > k.cur.High {
		k.cur.High = price
	}
	if price < k.cur.Low {
		k.cur.Low = price
	}
	return
}

func (k *CTPKline) sendCandle(c *Candle) {
	if c == nil {
		return
	}
	d := NewExchangeData("candle", EventCandle, c)
	d.Symbol = k.symbol
	d.Data.Extra = "1m"
	k.datas <- d
}

func (k *CTPKline) Flush() {
	k.sendCandle(k.cur)
	k.cur = nil
	return
}
