WORKSPACE=$(cd $(dirname $0) && pwd -P)
mkdir -p output/bin/
cp -rf ./config output/
go build -o output/bin/noah-server -tags="rocksdb" github.com/lnhote/noah/server/noahserver
