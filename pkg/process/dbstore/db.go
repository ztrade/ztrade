package dbstore

import (
	"fmt"
	"reflect"
	"regexp"
	"sync"

	"github.com/ztrade/base/common"
	. "github.com/ztrade/ztrade/pkg/core"

	_ "github.com/go-sql-driver/mysql"
	"github.com/go-xorm/xorm"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
	log "github.com/sirupsen/logrus"
)

var (
	tblRegexp = regexp.MustCompile(`^([A-Za-z0-9]+)_([A-Za-z0-9]+)_([A-Za-z0-9]+)$`)
)

type TableInfo struct {
	Exchange string
	Symbol   string
	BinSize  string
}

type DBStore struct {
	dbType string
	dbPath string
	table  string
	engine *xorm.Engine
	tbls   sync.Map
}

// NewDBStore support sqlite,mysql,pg
func NewDBStore(dbType, dbURI string) (dr *DBStore, err error) {
	dr = new(DBStore)
	dr.dbType = dbType
	dr.dbPath = dbURI
	err = dr.initDB()
	return
}

// Close close db
func (dr *DBStore) Close() (err error) {
	if dr.engine != nil {
		err = dr.engine.Close()
	}
	return
}

// SetDebug set debug mode
func (dr *DBStore) SetDebug(bDebug bool) {
	dr.engine.ShowSQL(bDebug)
}

func (dr *DBStore) initDB() (err error) {
	if dr.engine != nil {
		dr.engine.Close()
	}
	dr.engine, err = xorm.NewEngine(dr.dbType, dr.dbPath)
	if err != nil {
		err = fmt.Errorf("init db failed:%s", err.Error())
		return
	}
	err = dr.engine.Sync2(&SymbolInfo{})
	return
}

// GetTableSession get table,if not exsit, create the table
func (dr *DBStore) GetTableSession(tbl string, data TimeData) (sess *xorm.Session) {
	bExit, err := dr.engine.IsTableExist(tbl)
	if err != nil {
		log.Error("dbstore get table failed:", err.Error())
	}
	if !bExit {
		log.Debugf("create table %s ", dr.table, reflect.TypeOf(data))
		data.SetTable(tbl)
		fmt.Println(tbl, reflect.TypeOf(data))
		dr.engine.Sync2(data)
	}
	sess = dr.engine.NewSession()
	sess = sess.Table(tbl)
	return
}

func (dr *DBStore) getTblSess(tbl string) (sess *xorm.Session) {
	sess = dr.engine.NewSession()
	sess = sess.Table(tbl)
	return
}

// GetKlineTbl get kline table
func (dr *DBStore) GetKlineTbl(exchange, symbol, binSize string) *KlineTbl {
	key := fmt.Sprintf("%s_%s_%s", exchange, symbol, binSize)
	v, ok := dr.tbls.Load(key)
	if ok {
		return v.(*KlineTbl)
	}
	t := NewKlineTbl(dr, exchange, symbol, binSize)
	dr.tbls.Store(key, t)
	return t
}

// WriteKlines write klines
func (d *DBStore) WriteKlines(exchange, symbol, binSize string, datas []interface{}) (err error) {
	err = d.GetKlineTbl(exchange, symbol, binSize).WriteDatas(datas)
	return
}

func (dr *DBStore) GetTables() (tbls []string, err error) {
	allTbls, err := dr.engine.DBMetas()
	if err != nil {
		return
	}
	for _, v := range allTbls {
		tbls = append(tbls, v.Name)
	}
	return
}

func (dr *DBStore) GetKlineTables() (tbls []TableInfo, err error) {
	tblNames, err := dr.GetTables()
	if err != nil {
		return
	}
	for _, v := range tblNames {

		ret := tblRegexp.FindAllStringSubmatch(v, -1)
		if len(ret) != 1 {
			continue
		}
		if len(ret[0]) != 4 {
			continue
		}
		_, err = common.GetBinSizeDuration(ret[0][3])
		if err != nil {
			err = nil
			continue
		}
		tbls = append(tbls, TableInfo{Exchange: ret[0][1], Symbol: ret[0][2], BinSize: ret[0][3]})
	}
	return
}
