package dbstore

import "github.com/spf13/viper"

// Load init db from config file
func LoadDB(cfg *viper.Viper) (db *DBStore, err error) {
	dbType := cfg.GetString("db.type")
	dbURI := cfg.GetString("db.uri")
	db, err = NewDBStore(dbType, dbURI)
	return
}
