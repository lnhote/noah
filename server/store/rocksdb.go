package store

import (
	"fmt"

	lediscfg "github.com/siddontang/ledisdb/config"
	"github.com/siddontang/ledisdb/ledis"
	"github.com/siddontang/ledisdb/store/rocksdb"
)

var (
	db *ledis.DB
)

func init() {
	cfg := lediscfg.NewConfigDefault()
	cfg.DBName = rocksdb.DBName
	cfg.DataDir = "/tmp/noah"
	l, err := ledis.Open(cfg)
	if err != nil {
		panic(err.Error())
	}
	db, err = l.Select(0)
	if err != nil {
		panic(err.Error())
	}
}

func DBGet(key string) ([]byte, error) {
	var val, err = db.Get([]byte(key))
	if err != nil {
		return nil, fmt.Errorf("DB_GET fail: %s", err.Error())
	}
	return val, nil
}

func DBSet(key string, val []byte) error {
	if err := db.Set([]byte(key), val); err != nil {
		return fmt.Errorf("DB_SET fail: %s", err.Error())
	}
	return nil
}
