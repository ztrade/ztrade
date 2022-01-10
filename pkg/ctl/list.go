package ctl

import (
	"sort"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/ztrade/base/common"
	"github.com/ztrade/trademodel"
	"github.com/ztrade/ztrade/pkg/process/dbstore"
)

type LocalDataInfo struct {
	dbstore.TableInfo

	Start time.Time
	End   time.Time
}

type LocalData struct {
	db *dbstore.DBStore
}

func NewLocalData(db *dbstore.DBStore) (l *LocalData, err error) {
	l = new(LocalData)
	l.db = db
	return
}

func (l *LocalData) ListAll() (infos []LocalDataInfo, err error) {
	tbls, err := l.db.GetKlineTables()
	if err != nil {
		return
	}
	var temp []LocalDataInfo
	for _, v := range tbls {
		if v.Exchange == "DCE" {
			continue
		}
		temp, err = l.checkOne(v)
		if err != nil {
			log.Errorf("check table %s_%s_%s failed", v.Exchange, v.Symbol, v.BinSize)
			continue
		}
		infos = append(infos, temp...)
	}
	sort.Slice(infos, func(i, j int) bool {
		infoA := infos[i]
		infoB := infos[j]
		if infoA.Exchange == infoB.Exchange {
			if infoA.Symbol == infoB.Symbol {
				tA, _ := common.GetBinSizeDuration(infoA.BinSize)
				tB, _ := common.GetBinSizeDuration(infoB.BinSize)
				return tA < tB
			}
			return infoA.Symbol < infoB.Symbol
		}
		return infoA.Exchange < infoB.Exchange
	})
	return
}

func (l *LocalData) checkOne(tbl dbstore.TableInfo) (infos []LocalDataInfo, err error) {
	ktbl := l.db.GetKlineTbl(tbl.Exchange, tbl.Symbol, tbl.BinSize)
	tEnd := ktbl.GetNewest()
	tStart := ktbl.GetOldest()
	nCount, _ := ktbl.Count()
	dur, err := common.GetBinSizeDuration(tbl.BinSize)
	if err != nil {
		return
	}
	nDur := int64(tEnd.Sub(tStart)/dur) + 1
	if nDur == nCount {
		infos = []LocalDataInfo{LocalDataInfo{Start: tStart, End: tEnd, TableInfo: tbl}}
		return
	}
	if nCount == 0 {
		return
	}
	return l.checkOneRaw(ktbl, tStart, tEnd, dur, tbl)
}
func (l *LocalData) checkOneRaw(ktbl *dbstore.KlineTbl, tStart, tEnd time.Time, nDur time.Duration, tbl dbstore.TableInfo) (infos []LocalDataInfo, err error) {
	binSize := tbl.BinSize
	datas, err := ktbl.DataChan(tStart, tEnd, binSize)
	if err != nil {
		return
	}
	var i time.Duration
	var resetStart bool
	tempStart := tStart
	tempEnd := tEnd
	for d := range datas {
		for _, v := range d {
			c, _ := v.(*trademodel.Candle)
			if resetStart {
				tempStart = c.Time()
			}
			if c.Time().Sub(tempStart) != i*nDur {
				infos = append(infos, LocalDataInfo{Start: tempStart, End: tempEnd, TableInfo: tbl})
				resetStart = true
				i = 0
			} else {
				resetStart = false
				i++
			}
			tempEnd = c.Time()
		}
	}
	if tempStart != tempEnd {
		infos = append(infos, LocalDataInfo{Start: tempStart, End: tempEnd, TableInfo: tbl})
	}
	return
}
