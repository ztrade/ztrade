package common

import (
	"testing"
	"time"

	. "github.com/SuperGod/trademodel"
)

func getTestData(source, dst time.Duration) (candles CandleList) {
	nSourceSec := int64(source / time.Second)
	nStart := nSourceSec * (time.Now().Add(0-source*20).Unix() / nSourceSec)
	candle := Candle{
		Start:  nStart,
		Open:   100,
		High:   200,
		Low:    50,
		Close:  110,
		VWP:    1,
		Volume: 1,
		Trades: 10,
	}
	for i := 0; i != 10; i++ {
		temp := candle
		temp.Open = candle.Open + float64(i)
		temp.High = candle.High + float64(i)
		temp.Low = candle.Low + float64(i)
		temp.Close = candle.Close + float64(i)
		temp.VWP = candle.VWP + float64(i)
		temp.Volume = candle.Volume + float64(i)
		temp.Trades = candle.Trades + int64(i)
		candles = append(candles, &temp)
		candle.Start += nSourceSec
	}
	return
}

func TestMergeKline(t *testing.T) {
	source := time.Minute * 5
	dst := time.Minute * 15
	candles := getTestData(source, dst)
	m := NewKlineMerge(source, dst)
	for _, v := range candles {
		t.Log("candle:", v)
	}
	var ret interface{}
	for _, v := range candles {
		ret = m.Update(v)
		if ret == nil {
			continue
		}
		t.Log("ret:", ret)
	}
}

func TestMergeKlineChan(t *testing.T) {
	klines := make(chan []interface{}, 10)
	source := time.Minute * 5
	dst := time.Minute * 15
	candles := getTestData(source, dst)
	go func() {
		datas := make([]interface{}, len(candles))
		for k, v := range candles {
			t.Log(v)
			datas[k] = v
		}
		klines <- datas
		close(klines)
	}()
	ret := MergeKlineChan(klines, source, dst)
	for v := range ret {
		for _, d := range v {
			t.Log("ret:", d)
		}
	}
}
