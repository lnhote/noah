[![License](http://img.shields.io/badge/license-mit-blue.svg?style=flat-square)](https://raw.githubusercontent.com/lnhote/noah/master/LICENSE)


# What is noah?
noah is a distributed consensus kv store system.

## Key features:
* Raft protocol for keeping values consensus among servers in cluster
* Rocksdb for local store

## Design

## Getting started

### 1. Install Rocksdb library
https://github.com/facebook/rocksdb/blob/master/INSTALL.md
Notice: go install with tag "rocksdb"

```bash
brew install rocksdb
```

### 2. Install vendor packages using glide

```bash
brew install glide

glide install

glide update

sh build.sh

cd output/

./bin/noah-server -data /tmp/noah/data
```

TODO
> -d: daemon mode  
> -h: help info  
> -v: show version info  
> -c: path to configaration file, e.g., /etc/noah/server.conf  

### glide manual

```bash
brew install glide

glide mirror set https://golang.org/x/net https://github.com/golang/net --vcs git
glide mirror set google.golang.org/grpc https://github.com/grpc/grpc-go --vcs git

glide init

glide install

glide update

glide get package/xxx
```

### test

```bash
go test -tags="rocksdb" ./...
```