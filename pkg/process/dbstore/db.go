package dbstore

import (
	"fmt"
	"reflect"
	"sync"

	. "github.com/ztrade/ztrade/pkg/define"

	_ "github.com/go-sql-driver/mysql"
	"github.com/go-xorm/xorm"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
	log "github.com/sirupsen/logrus"
)

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
