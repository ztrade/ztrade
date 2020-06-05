package common

import (
	"fmt"
	"time"

	. "github.com/SuperGod/trademodel"
	log "github.com/sirupsen/logrus"
)

// MergeKlineChan merge kline data
func MergeKlineChan(klines chan []interface{}, srcDuration, dstDuration time.Duration) (rets chan []interface{}) {
	rets = make(chan []interface{}, len(klines))
	go func() {
		km := NewKlineMerge(srcDuration, dstDuration)
		var temp interface{}
		for v := range klines {
			tempDatas := []interface{}{}
			for _, d := range v {
				temp = km.Update(d)
				if temp != nil {
					tempDatas = append(tempDatas, temp)
				}
			}
			if len(tempDatas) != 0 {
				rets <- tempDatas
			}
		}
		close(rets)
	}()
	return
}

// KlineMerge merge kline to new duration
type KlineMerge struct {
	src    int64      // src kline seconds
	dst    int64      // dst kline seconds
	ratio  int        // dst/src kline ration
	cache  CandleList // kline cache
	bFirst bool
}

// NewKlineMergeStr new KlineMerge with string duration
func NewKlineMergeStr(src, dst string) *KlineMerge {
	srcDur, err := time.ParseDuration(src)
	if err != nil {
		log.Errorf("NewKlineMergeStr parse src %s error: %s", src, err.Error())
		return nil
	}
	dstDur, err := time.ParseDuration(dst)
	if err != nil {
		log.Errorf("NewKlineMergeStr parse dst %s error: %s", dst, err.Error())
		return nil
	}
	return NewKlineMerge(srcDur, dstDur)
}

// NewKlineMerge merge kline constructor
func NewKlineMerge(src, dst time.Duration) *KlineMerge {
	km := new(KlineMerge)
	km.src = int64(src / time.Second)
	km.dst = int64(dst / time.Second)
	km.ratio = int(dst / src)
	km.bFirst = true
	return km
}

// IsFirst is first time
func (km *KlineMerge) IsFirst() bool {
	return km.bFirst
}

// NeedMerge is kline need merge
func (km *KlineMerge) NeedMerge() bool {
	return km.ratio != 1
}

// GetSrc return kline source duration secs
func (km *KlineMerge) GetSrc() int64 {
	return km.src
}

// GetSrcDuration get kline source duration
func (km *KlineMerge) GetSrcDuration() time.Duration {
	return time.Duration(km.src) * time.Second
}

// GetDstDuration get kline dst duration
func (km *KlineMerge) GetDstDuration() time.Duration {
	return time.Duration(km.dst) * time.Second
}

// GetDst return kline dst duration secs
func (km *KlineMerge) GetDst() int64 {
	return km.dst
}

// Update update candle, and return new kline candle
// return nil if no new kline candle
func (km *KlineMerge) Update(data interface{}) (ret interface{}) {
	// return if no need to merge
	if km.ratio == 1 {
		ret = data
		return
	}
	candle, ok := data.(*Candle)
	if !ok {
		panic(fmt.Sprintf("KlineMerge data type error:%#v", data))
		return
	}
	n := len(km.cache)
	if n > 0 && candle.Start <= km.cache[n-1].Start {
		return
	}
	if km.bFirst && candle.Start%km.dst != 0 {
		return
	}
	km.bFirst = false
	// add current candle to cache
	index := int(candle.Start%km.dst)/int(km.src) + 1
	km.cache = append(km.cache, candle)
	if index != km.ratio {
		return
	}
	defer func() {
		// reset cache after kline merged
		km.cache = CandleList{}
	}()
	// cache length not match,just skip
	if len(km.cache) != km.ratio {
		log.Infof("cache length not match, skip %d %d", len(km.cache), km.ratio)
		return
	}
	ret = km.cache.Merge()
	return
}
