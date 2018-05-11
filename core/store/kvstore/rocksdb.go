package kvstore

import (
	"fmt"

	"github.com/lnhote/noah/core/errmsg"
	lediscfg "github.com/siddontang/ledisdb/config"
	"github.com/siddontang/ledisdb/ledis"
	"github.com/siddontang/ledisdb/store/rocksdb"
)

type KVStore interface {
	Connect() error
	Close() error
	Get(key string) ([]byte, error)
	Set(key string, val []byte) error
}

type RocksDBStore struct {
	db        *ledis.DB
	ledisInst *ledis.Ledis
	dataDir   string
}

func NewRocksDB(dataDir string) *RocksDBStore {
	return &RocksDBStore{db: nil, ledisInst: nil, dataDir: dataDir}
}

func (r *RocksDBStore) Close() error {
	if r.ledisInst != nil {
		r.ledisInst.Close()
	}
	return errmsg.DataTooShort
}

func (r *RocksDBStore) Connect() error {
	cfg := lediscfg.NewConfigDefault()
	cfg.DBName = rocksdb.DBName
	cfg.DataDir = r.dataDir
	ledisInst, err := ledis.Open(cfg)
	if err != nil {
		panic(err.Error())
	}
	r.db, err = ledisInst.Select(0)
	if err != nil {
		panic(err.Error())
	}
	return nil
}

func (r *RocksDBStore) Get(key string) ([]byte, error) {
	var val, err = r.db.Get([]byte(key))
	if err != nil {
		return nil, fmt.Errorf("DB_GET fail: %s", err.Error())
	}
	return val, nil
}

func (r *RocksDBStore) Set(key string, val []byte) error {
	if err := r.db.Set([]byte(key), val); err != nil {
		return fmt.Errorf("DB_SET fail: %s", err.Error())
	}
	return nil
}
