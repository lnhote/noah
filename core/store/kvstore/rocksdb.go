package kvstore

import (
	"fmt"

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
	cfg       *lediscfg.Config
}

func NewRocksDB(dataDir string) *RocksDBStore {
	cfg := lediscfg.NewConfigDefault()
	cfg.DBName = rocksdb.DBName
	cfg.DataDir = dataDir
	return &RocksDBStore{db: nil, ledisInst: nil, dataDir: dataDir, cfg: cfg}
}

func (r *RocksDBStore) Close() error {
	if r.ledisInst != nil {
		r.ledisInst.Close()
		r.ledisInst = nil
	}
	return nil
}

func (r *RocksDBStore) Connect() error {
	var err error
	if r.ledisInst == nil {
		r.ledisInst, err = ledis.Open(r.cfg)
		if err != nil {
			panic(err.Error())
		}
	}
	r.db, err = r.ledisInst.Select(0)
	if err != nil {
		panic(err.Error())
	}
	return nil
}

func (r *RocksDBStore) Get(key string) ([]byte, error) {
	if r.db == nil {
		return nil, fmt.Errorf("RocksDBStore not connected")
	}
	var val, err = r.db.Get([]byte(key))
	if err != nil {
		return nil, fmt.Errorf("DB_GET fail: %s", err.Error())
	}
	return val, nil
}

func (r *RocksDBStore) Set(key string, val []byte) error {
	if r.db == nil {
		return fmt.Errorf("RocksDBStore not connected")
	}
	if err := r.db.Set([]byte(key), val); err != nil {
		return fmt.Errorf("DB_SET fail: %s", err.Error())
	}
	return nil
}
