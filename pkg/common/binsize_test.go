package common

import (
	"testing"
	"time"
)

func TestGetBinSizeDuration(t *testing.T) {
	testMap := map[string]time.Duration{
		"1s":  time.Second,
		"5s":  5 * time.Second,
		"m":   time.Minute,
		"1m":  time.Minute,
		"5m":  5 * time.Minute,
		"15m": 15 * time.Minute,
		"30m": 30 * time.Minute,
		"1h":  time.Hour,
		"4h":  4 * time.Hour,
		"6h":  6 * time.Hour,
		"1d":  Day,
		"7d":  Week,
		"1w":  Week,
		"1":   time.Minute,
		"15":  15 * time.Minute,
		"60":  time.Hour,
	}

	var temp time.Duration
	var err error
	for k, v := range testMap {
		temp, err = GetBinSizeDuration(k)
		if err != nil {
			t.Fatalf("parse %s failed:%s", k, err.Error())
		}
		if temp != v {
			t.Fatalf("GetBinSizeDuration failed:%s %s", temp, v)
		}
	}
}
