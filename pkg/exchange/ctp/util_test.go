package ctp

import (
	"testing"
	"time"
)

func TestTradeTime(t *testing.T) {
	datas := map[string]int{
		"2021-11-22 00:00:00": DayMinute(8, 49),
		"2021-11-22 08:00:00": DayMinute(0, 49),
		"2021-11-22 12:00:00": DayMinute(1, 29),
		"2021-11-22 15:30:00": DayMinute(5, 19),
		"2021-11-23 00:00:00": DayMinute(0, 0),
	}
	for k, v := range datas {
		tm, err := time.Parse("2006-01-02 15:04:05", k)
		if err != nil {
			t.Fatal(err.Error())
		}
		dur := TradeTimes.Range[tm.Weekday()].NeedWait(tm)
		t.Log(k, dur)
		n := time.Duration(v) * time.Minute
		if n != dur {
			t.Fatalf("%s wait: %s not match: %s", k, dur, n)
		}
	}
}
