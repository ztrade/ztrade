package dbstore

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	// . "ztrade/pkg/define"
	// . "ztrade/pkg/event"

	// . "github.com/SuperGod/trademodel"
	"github.com/go-xorm/xorm"
	log "github.com/sirupsen/logrus"
)

// TimeData data with time info
type TimeData interface {
	GetStart() int64
	Time() time.Time
	GetTable() string
	SetTable(string)
}

type DataCreator interface {
	Sing() TimeData
	Slice() interface{}
	GetSlice(interface{}) []interface{}
}

// TimeTbl tbl with time info
type TimeTbl struct {
	db       *DBStore
	exchange string
	symbol   string
	binSize  string
	table    string
	creator  DataCreator
	closeCh  chan bool
}

// NewTimeTbl create new time table
func NewTimeTbl(db *DBStore, creator DataCreator, exchange, symbol, binSize, extName string) (t *TimeTbl) {
	t = new(TimeTbl)
	t.db = db
	t.creator = creator
	t.exchange = exchange
	t.symbol = symbol
	t.binSize = binSize

	t.table = fmt.Sprintf("%s_%s_%s", exchange, symbol, binSize)
	if extName != "" {
		t.table += "_" + extName
	}
	if creator.Sing == nil || creator.Slice == nil {
		log.Error("TimeTbl data creator can't be nil")
		t = nil
	}
	return
}

func (t *TimeTbl) SetCloseCh(closeCh chan bool) {
	t.closeCh = closeCh
}

func (t *TimeTbl) getTable() (sess *xorm.Session) {
	data := t.creator.Sing()
	sess = t.db.GetTableSession(t.table, data)
	return
}

func (t *TimeTbl) GetSymbol() string {
	return t.symbol
}

func (t *TimeTbl) GetTable() string {
	return t.table
}

func (t *TimeTbl) GetDatas(since, end time.Time, limit int) (datas []interface{}, err error) {
	return t.getDatasWithParam(since, end, limit, 0)
}

func (t *TimeTbl) getDatasWithParam(since, end time.Time, limit, offset int) (datas []interface{}, err error) {
	ret := t.creator.Slice()
	sess := t.getTable()
	defer sess.Close()
	err = sess.Asc("start").Where("start>=? and start<?", since.Unix(), end.Unix()).Limit(limit, offset).Find(ret)
	if err != nil {
		return
	}
	datas = t.creator.GetSlice(ret)
	return
}

func (t *TimeTbl) DataRecent(recent int32, bSize string) (klines []interface{}, err error) {
	if bSize != t.binSize {
		err = fmt.Errorf("kline tbl %s binsize error: %s", t.table, bSize)
		return
	}
	ret := t.creator.Slice()
	sess := t.getTable()
	defer sess.Close()
	err = sess.Desc("start").Limit(int(recent), 0).Find(ret)
	if err != nil {
		return
	}
	datas := t.creator.GetSlice(ret)
	klines = make([]interface{}, len(datas))
	for k, v := range datas {
		klines[len(klines)-k-1] = v
	}
	return
}

func (t *TimeTbl) DataChan(start, end time.Time, bSize string) (klines chan []interface{}, err error) {
	if bSize != t.binSize {
		err = fmt.Errorf("kline tbl %s binsize error: %s", t.table, bSize)
		return
	}
	klines = make(chan []interface{}, 10)
	go func() {
		nOffset := 0
		once := 500
		var err1 error
		var data []interface{}
		for {
			data, err1 = t.getDatasWithParam(start, end, once, nOffset)
			if err1 != nil {
				break
			}
			if len(data) == 0 {
				break
			}
			nOffset += len(data)
			klines <- data
			if len(data) < once {
				break
			}
		}
		if err1 != nil {
			log.Error("TimeTbl DataChan getDatas failed:", err1.Error())
		}
		close(klines)
	}()
	return
}

func (tbl *TimeTbl) IsEmpty() (isEmpty bool) {
	isEmpty = true
	sess := tbl.getTable()
	defer sess.Close()
	n, err := sess.Count()
	if err != nil {
		log.Errorf("table:%s count failed:%s", tbl.table, err.Error())
		return
	}
	if n > 0 {
		isEmpty = false
	}
	return
}

func (tbl *TimeTbl) GetNewest() (t time.Time) {
	sess := tbl.getTable()
	defer sess.Close()
	data := tbl.creator.Sing()
	_, err := sess.Desc("start").Limit(1, 0).Get(data)
	if err != nil {
		log.Errorf("TimeTbl get newest %s failed:%s", tbl.table, err.Error())
		return
	}
	t = data.Time()
	return
}

func (tbl *TimeTbl) GetOldest() (t time.Time) {
	sess := tbl.getTable()
	defer sess.Close()
	data := tbl.creator.Sing()
	_, err := sess.Asc("start").Limit(1, 0).Get(data)
	if err != nil {
		log.Errorf("TimeTbl get newest %s failed:%s", tbl.table, err.Error())
		return
	}
	t = data.Time()
	return
}

// Exists check if data's time exists
func (t *TimeTbl) Exists(data interface{}) (bRet bool, err error) {
	sess := t.getTable()
	if sess == nil {
		err = errors.New("no such table")
		return
	}
	defer sess.Close()
	v, ok := data.(TimeData)
	if !ok {
		err = fmt.Errorf("UpdateData type error:%s", reflect.TypeOf(v))
		return
	}
	bRet, err = sess.Table(t.table).Where("start=?", v.GetStart()).Exist()
	return
}

func (t *TimeTbl) AddOrUpdateData(data interface{}) (err error) {
	err = t.UpdateData(data)
	if err != nil {
		err = t.WriteData(data)
	}
	return
}

// UpdateData update datas
func (t *TimeTbl) UpdateData(data interface{}) (err error) {
	sess := t.getTable()
	if sess == nil {
		err = errors.New("no such table")
		return
	}
	defer sess.Close()
	var v TimeData
	v, ok := data.(TimeData)
	if !ok {
		err = fmt.Errorf("UpdateData type error:%s", reflect.TypeOf(v))
		return
	}
	v.SetTable(t.table)
	n, err := sess.Table(t.table).Where("start=?", v.GetStart()).UseBool().Update(data)
	if err != nil {
		log.Errorf("TimeTbl update data %s error:%s", v, err.Error())
		return
	}
	if n == 0 {
		err = fmt.Errorf("no such data")
	}
	return
}

// WriteData write data
func (t *TimeTbl) WriteData(data interface{}) (err error) {
	sess := t.getTable()
	if sess == nil {
		err = errors.New("no such table")
		return
	}
	defer sess.Close()

	var bRet bool
	var v TimeData

	v, ok := data.(TimeData)
	if !ok {
		err = fmt.Errorf("WriteData type error:%s", reflect.TypeOf(v))
		return
	}
	v.SetTable(t.table)
	bRet, err = sess.Table(t.table).Where("start=?", v.GetStart()).Exist()
	if err != nil {
		log.Errorf("TimeTbl check exist %s error:%s", v, err.Error())
		return
	}
	if bRet {
		log.Debugf("insert %s exist", v)
		return
	}
	_, err = sess.Insert(data)
	if err != nil {
		log.Errorf("TimeTbl insert %s error:%s", v, err.Error())
	}
	return
}

// WriteDatas write datas
func (t *TimeTbl) WriteDatas(datas []interface{}) (err error) {
	sess := t.getTable()
	if sess == nil {
		err = errors.New("no such table")
		return
	}
	defer sess.Close()
	err = sess.Begin()
	if err != nil {
		return
	}

	var v TimeData
	for _, data := range datas {
		v = data.(TimeData)
		v.SetTable(t.table)
		_, err = sess.Insert(data)
		if err != nil {
			if strings.Contains(err.Error(), "Duplicate entry") {
				log.Debugf("TimeTbl insert %s error:%#v", v, err)
			} else {
				log.Errorf("TimeTbl insert %s error:%#v", v, err)
			}
		}
		err = nil
	}
	sess.Commit()
	return
}
