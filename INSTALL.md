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
[recommended] make static_lib

```bash
brew install rocksdb
```