package main

import (
	"fmt"
	"time"

	//"time"

	lediscfg "github.com/siddontang/ledisdb/config"
	"github.com/siddontang/ledisdb/ledis"
	"github.com/siddontang/ledisdb/store/rocksdb"
)

func main() {
	cfg := lediscfg.NewConfigDefault()
	cfg.DBName = rocksdb.DBName
	l, err := ledis.Open(cfg)
	if err != nil {
		panic(err.Error())
	}
	db, err := l.Select(0)
	if err != nil {
		panic(err.Error())
	}

	if err := db.Set([]byte("name"), []byte(time.Now().String())); err != nil {
		fmt.Errorf("set fail: err = %s", err.Error())
		return
	}

	var val []byte
	val, err = db.Get([]byte("name"))
	if err != nil {
		fmt.Errorf("get fail: err = %s", err.Error())
		return
	}
	fmt.Printf("result = %v", string(val))
}
