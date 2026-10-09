# Redis Clone in Go

I built this project to understand how Redis works under the hood, from handling TCP connections to storing data in memory and processing commands.

The goal is to build a lightweight, Redis-inspired key-value store from scratch using Go, without relying on Docker or an existing database implementation.

## What it can do

- Accept client connections over TCP and handle multiple clients concurrently using goroutines.
- Process Redis-style commands using a subset of the RESP protocol.
- Store and retrieve values with `SET` and `GET`.
- Delete keys and check whether they exist using `DEL` and `EXISTS`.
- Set expiration times on keys using `EXPIRE`, `TTL`, and `SET ... EX`.
- Work with hashes using `HSET`, `HGET`, and `HGETALL`.
- Work with sets using `SADD` and `SMEMBERS`.
- Save the current database state to a JSON file and load it again when the server restarts.

## Built with

- **Go** for the server and storage logic
- **TCP sockets** for client-server communication
- **RESP** for Redis-style command parsing and responses
- **Goroutines and mutexes** for concurrent connections and thread-safe storage
- **JSON** for saving and restoring database snapshots

## Getting started

Make sure Go is installed on your machine.

Clone the repository and enter the project directory:

```bash
git clone https://github.com/GunjanGaur1/redis-clone.git
cd redis-clone
```

Start the server:

```bash
go run .
```

By default, the server listens on port `6379`.

## Trying it out

Open another terminal and connect to the server using `nc`:

```bash
nc localhost 6379
```

You can now enter commands directly:

```text
PING
SET name Gunjan
GET name
SET language Go
GET language
SAVE
```

The server returns Redis-style responses. For example, `PING` returns `+PONG`, and `GET name` returns the stored value.

If you have `redis-cli` installed, you can use it instead.

## Project structure

```text
redis-clone/
├── main.go
├── go.mod
├── README.md
└── internal/
    ├── resp/
    │   └── resp.go
    ├── server/
    │   └── server.go
    └── store/
        └── store.go
```

- `main.go` starts the server and handles shutdown.
- `internal/server` manages TCP connections and dispatches commands.
- `internal/resp` parses incoming commands and formats responses.
- `internal/store` manages the in-memory database, expiration, and snapshots.

## Current limitations

This is a learning project, not a full replacement for Redis. Replication, Pub/Sub, transactions, and append-only file persistence are not implemented yet. Persistence currently uses JSON snapshots rather than Redis's native persistence formats.

## Why I built it

I wanted to get hands-on experience with networking, concurrency, data structures, and persistence instead of treating a database as a black box. Building each part has helped me explore how a client request travels through a server and how data is stored, retrieved, and recovered.
