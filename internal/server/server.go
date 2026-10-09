package server

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"time"

	"redis-clone/internal/resp"
	"redis-clone/internal/store"
)

type Server struct {
	addr     string
	db       *store.Store
	dataFile string
}

func New(addr string, db *store.Store, dataFile string) *Server {
	return &Server{addr: addr, db: db, dataFile: dataFile}
}

func (s *Server) ListenAndServe() error {
	ln, err := net.Listen("tcp", s.addr)
	if err != nil {
		return err
	}
	defer ln.Close()
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("accept error: %v", err)
			continue
		}
		go s.handleConn(conn)
	}
}

func (s *Server) handleConn(conn net.Conn) {
	defer conn.Close()
	r := bufio.NewReader(conn)
	for {
		args, err := resp.ReadCommand(r)
		if err != nil {
			return
		}
		reply := s.execute(args)
		if _, err := conn.Write(reply); err != nil {
			return
		}
	}
}

func (s *Server) execute(args []string) []byte {
	if len(args) == 0 {
		return resp.Error("empty command")
	}
	cmd := strings.ToUpper(args[0])
	arity := func(n int) bool { return len(args) == n }
	switch cmd {
	case "PING":
		if len(args) == 1 {
			return resp.Simple("PONG")
		}
		if len(args) == 2 {
			return resp.Bulk(args[1])
		}
		return resp.Error("wrong number of arguments for PING")
	case "ECHO":
		if !arity(2) {
			return resp.Error("wrong number of arguments for ECHO")
		}
		return resp.Bulk(args[1])
	case "SET":
		if len(args) != 3 && len(args) != 5 {
			return resp.Error("usage: SET key value [EX seconds]")
		}
		var ttl time.Duration
		if len(args) == 5 {
			if strings.ToUpper(args[3]) != "EX" {
				return resp.Error("only the EX option is supported")
			}
			seconds, err := strconv.Atoi(args[4])
			if err != nil || seconds <= 0 {
				return resp.Error("EX seconds must be a positive integer")
			}
			ttl = time.Duration(seconds) * time.Second
		}
		s.db.Set(args[1], args[2], ttl)
		return resp.Simple("OK")
	case "GET":
		if !arity(2) {
			return resp.Error("wrong number of arguments for GET")
		}
		value, ok, err := s.db.Get(args[1])
		if err != nil {
			return resp.Error(err.Error())
		}
		if !ok {
			return resp.NullBulk()
		}
		return resp.Bulk(value)
	case "DEL":
		if len(args) < 2 {
			return resp.Error("wrong number of arguments for DEL")
		}
		return resp.Integer(s.db.Del(args[1:]...))
	case "EXISTS":
		if !arity(2) {
			return resp.Error("wrong number of arguments for EXISTS")
		}
		if s.db.Exists(args[1]) {
			return resp.Integer(1)
		}
		return resp.Integer(0)
	case "EXPIRE":
		if !arity(3) {
			return resp.Error("usage: EXPIRE key seconds")
		}
		seconds, err := strconv.Atoi(args[2])
		if err != nil {
			return resp.Error("seconds must be an integer")
		}
		if s.db.Expire(args[1], time.Duration(seconds)*time.Second) {
			return resp.Integer(1)
		}
		return resp.Integer(0)
	case "TTL":
		if !arity(2) {
			return resp.Error("wrong number of arguments for TTL")
		}
		return resp.Integer(s.db.TTL(args[1]))
	case "HSET":
		if !arity(4) {
			return resp.Error("usage: HSET key field value")
		}
		n, err := s.db.HSet(args[1], args[2], args[3])
		if err != nil {
			return resp.Error(err.Error())
		}
		return resp.Integer(n)
	case "HGET":
		if !arity(3) {
			return resp.Error("usage: HGET key field")
		}
		v, ok, err := s.db.HGet(args[1], args[2])
		if err != nil {
			return resp.Error(err.Error())
		}
		if !ok {
			return resp.NullBulk()
		}
		return resp.Bulk(v)
	case "HGETALL":
		if !arity(2) {
			return resp.Error("wrong number of arguments for HGETALL")
		}
		values, err := s.db.HGetAll(args[1])
		if err != nil {
			return resp.Error(err.Error())
		}
		return resp.Array(values)
	case "SADD":
		if len(args) < 3 {
			return resp.Error("usage: SADD key member [member ...]")
		}
		n, err := s.db.SAdd(args[1], args[2:]...)
		if err != nil {
			return resp.Error(err.Error())
		}
		return resp.Integer(n)
	case "SMEMBERS":
		if !arity(2) {
			return resp.Error("wrong number of arguments for SMEMBERS")
		}
		members, err := s.db.SMembers(args[1])
		if err != nil {
			return resp.Error(err.Error())
		}
		return resp.Array(members)
	case "SAVE":
		if !arity(1) {
			return resp.Error("SAVE takes no arguments")
		}
		if err := s.db.Save(s.dataFile); err != nil {
			return resp.Error("snapshot failed: " + err.Error())
		}
		return resp.Simple("OK")
	case "QUIT":
		return resp.Simple("OK")
	default:
		return resp.Error(fmt.Sprintf("unknown command '%s'", cmd))
	}
}
