package common

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	DefaultBinSizes = "1m, 5m, 15m, 30m, 1h, 4h, 1d"
)

var (
	Day  = time.Hour * 24
	Week = time.Hour * 24 * 7
)

// ParseBinStrs parse binsizes to strs
func ParseBinStrs(str string) (strs []string) {
	bins := strings.Split(str, ",")
	var temp string
	for _, v := range bins {
		temp = strings.Trim(v, " ")
		strs = append(strs, temp)
	}
	return
}

// ParseBinSizes parse binsizes
func ParseBinSizes(str string) (durations []time.Duration, err error) {
	strs := ParseBinStrs(str)
	var t time.Duration
	for _, v := range strs {
		t, err = GetBinSizeDuration(v)
		if err != nil {
			return
		}
		durations = append(durations, t)
	}
	return
}

// GetBinSizeDuration get duration of the binsize
func GetBinSizeDuration(str string) (duration time.Duration, err error) {
	if len(str) == 0 {
		err = errors.New("binsize is empty")
		return
	}
	n, err := strconv.ParseInt(str, 10, 64)
	if err == nil {
		duration = time.Duration(n) * time.Minute
		return
	}
	err = nil
	char := str[len(str)-1]
	switch char {
	case 's', 'S':
		duration = time.Second
	case 'm':
		duration = time.Minute
	case 'h':
		duration = time.Hour
	case 'd', 'D':
		duration = Day
	case 'w', 'W':
		duration = Week
	default:
		err = fmt.Errorf("unsupport binsize: %s", str)
		return
	}
	if len(str) == 1 {
		return
	}
	value := str[0 : len(str)-1]
	n, err = strconv.ParseInt(value, 10, 64)
	if err != nil {
		err = fmt.Errorf("parse binsize error:%s", err.Error())
		return
	}
	duration = time.Duration(n) * duration
	return
}
