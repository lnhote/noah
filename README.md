# What is noah?
noah is a distributed consensus kv store system.

## Key features:
* Raft protocol for keeping values consensus among servers in cluster
* Rocksdb for local store

## Design

## Getting started

### client
```bash
noah set {key} {value}
noah get {key}
```

```bash
echo -n "test out the server" | nc localhost 8848
```


### server
```bash
noah-server {start|stop|restart|status}
```

> -d: daemon mode
> -h: help info
> -v: show version info
> -c: path to configaration file, e.g., /etc/noah/server.conf
