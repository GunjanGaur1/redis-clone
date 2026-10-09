# Redis Clone in Go

An educational Redis-inspired TCP key-value server written from scratch in Go. It uses a subset of RESP so clients such as `redis-cli` can send commands.

## Implemented

- TCP server with concurrent client connections (goroutines)
- RESP array/bulk-string command parsing and simple inline commands
- `PING`, `ECHO`, `SET`, `GET`, `DEL`, `EXISTS`
- Expiration with `SET key value EX seconds`, `EXPIRE`, and `TTL`
- Hashes: `HSET`, `HGET`, `HGETALL`
- Sets: `SADD`, `SMEMBERS`
- Thread-safe shared store
- JSON snapshot persistence with `SAVE` and startup loading
- Graceful shutdown snapshot
- No Docker required

## Not implemented

This is not a complete Redis replacement. Replication, Pub/Sub, Redis ACLs, transactions, full RESP edge cases, eviction policies, and Redis-compatible persistence formats are not implemented.

## Requirements

- Go 1.22+
- Optional: `redis-cli` for convenient testing

## Run

```bash
go run . 
```

The server listens on `127.0.0.1:6379` (all interfaces on port 6379 by default) and stores snapshots in `dump.json`.

To choose another address or snapshot path:

```bash
go run . -addr 127.0.0.1:6380 -data ./data.json
```

## Test with redis-cli

In a second terminal:

```bash
redis-cli -p 6379 PING
redis-cli -p 6379 SET name Gunjan
redis-cli -p 6379 GET name
redis-cli -p 6379 SET session abc EX 30
redis-cli -p 6379 TTL session
redis-cli -p 6379 HSET user name Gunjan
redis-cli -p 6379 HGET user name
redis-cli -p 6379 SADD languages Go Python
redis-cli -p 6379 SMEMBERS languages
redis-cli -p 6379 SAVE
```

If `redis-cli` is not installed, you can use `nc localhost 6379` for simple inline commands, one command per line:

```text
PING
SET name Gunjan
GET name
```

## Project structure

```text
redis-clone/
├── main.go
├── go.mod
├── README.md
└── internal/
    ├── resp/       # RESP command parser and response encoders
    ├── server/     # TCP listener and command dispatch
    └── store/      # Thread-safe data structures and snapshots
```

## Important notes

- Data is primarily in memory. Use `SAVE` to write a JSON snapshot.
- Snapshot saving is explicit plus a final save on graceful shutdown; this is not an append-only log.
- Do not expose this educational server to untrusted networks.
