```bash
sudo apt-get install gcc-4.9
sudo apt-get install libgflags-dev
sudo apt-get install libsnappy-dev
sudo apt-get install zlib1g-dev
sudo apt-get install libbz2-dev
sudo apt-get install liblz4-dev
sudo apt-get install libzstd-dev
#make static_lib
git clone git@github.com:facebook/rocksdb.git
cd rocksdb
make shared_lib
```

```bash
brew install rocksdb
```
include path: /usr/local/include/rocksdb/
lib path: /usr/local/lib/

[recommended] make static_lib

```bash
CGO_CFLAGS="-I/usr/local/include/rocksdb" CGO_LDFLAGS="-L/usr/local/lib -lrocksdb -lstdc++ -lm -lz -lbz2 -lsnappy -llz4 -lzstd" go get github.com/tecbot/gorocksdb
```