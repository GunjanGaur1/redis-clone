package store

import (
	"encoding/json"
	"errors"
	"os"
	"sort"
	"sync"
	"time"
)

type entry struct {
	Kind    string            `json:"kind"`
	Value   string            `json:"value,omitempty"`
	Hash    map[string]string `json:"hash,omitempty"`
	Set     map[string]bool   `json:"set,omitempty"`
	Expires int64             `json:"expires,omitempty"` // Unix nanoseconds; zero means no expiry.
}

type Store struct {
	mu   sync.RWMutex
	data map[string]entry
}

func New() *Store { return &Store{data: make(map[string]entry)} }

func (s *Store) getLiveLocked(key string) (entry, bool) {
	e, ok := s.data[key]
	if !ok {
		return entry{}, false
	}
	if e.Expires != 0 && time.Now().UnixNano() >= e.Expires {
		delete(s.data, key)
		return entry{}, false
	}
	return e, true
}

func (s *Store) Set(key, value string, ttl time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e := entry{Kind: "string", Value: value}
	if ttl > 0 {
		e.Expires = time.Now().Add(ttl).UnixNano()
	}
	s.data[key] = e
}

func (s *Store) Get(key string) (string, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.getLiveLocked(key)
	if !ok {
		return "", false, nil
	}
	if e.Kind != "string" {
		return "", false, errors.New("WRONGTYPE operation against a key holding the wrong kind of value")
	}
	return e.Value, true, nil
}

func (s *Store) Del(keys ...string) int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	var n int64
	for _, key := range keys {
		if _, ok := s.getLiveLocked(key); ok {
			delete(s.data, key)
			n++
		}
	}
	return n
}

func (s *Store) Exists(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.getLiveLocked(key)
	return ok
}

func (s *Store) Expire(key string, ttl time.Duration) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.getLiveLocked(key)
	if !ok {
		return false
	}
	if ttl <= 0 {
		delete(s.data, key)
		return true
	}
	e.Expires = time.Now().Add(ttl).UnixNano()
	s.data[key] = e
	return true
}

func (s *Store) TTL(key string) int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.getLiveLocked(key)
	if !ok {
		return -2
	}
	if e.Expires == 0 {
		return -1
	}
	seconds := int64(time.Until(time.Unix(0, e.Expires)).Seconds())
	if seconds < 0 {
		return -2
	}
	return seconds
}

func (s *Store) HSet(key, field, value string) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.getLiveLocked(key)
	if ok && e.Kind != "hash" {
		return 0, errors.New("WRONGTYPE operation against a key holding the wrong kind of value")
	}
	if !ok {
		e = entry{Kind: "hash", Hash: make(map[string]string)}
	}
	if e.Hash == nil {
		e.Hash = make(map[string]string)
	}
	_, existed := e.Hash[field]
	e.Hash[field] = value
	s.data[key] = e
	if existed {
		return 0, nil
	}
	return 1, nil
}

func (s *Store) HGet(key, field string) (string, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.getLiveLocked(key)
	if !ok {
		return "", false, nil
	}
	if e.Kind != "hash" {
		return "", false, errors.New("WRONGTYPE operation against a key holding the wrong kind of value")
	}
	v, ok := e.Hash[field]
	return v, ok, nil
}

func (s *Store) HGetAll(key string) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.getLiveLocked(key)
	if !ok {
		return []string{}, nil
	}
	if e.Kind != "hash" {
		return nil, errors.New("WRONGTYPE operation against a key holding the wrong kind of value")
	}
	keys := make([]string, 0, len(e.Hash))
	for k := range e.Hash {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]string, 0, len(keys)*2)
	for _, k := range keys {
		out = append(out, k, e.Hash[k])
	}
	return out, nil
}

func (s *Store) SAdd(key string, members ...string) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.getLiveLocked(key)
	if ok && e.Kind != "set" {
		return 0, errors.New("WRONGTYPE operation against a key holding the wrong kind of value")
	}
	if !ok {
		e = entry{Kind: "set", Set: make(map[string]bool)}
	}
	if e.Set == nil {
		e.Set = make(map[string]bool)
	}
	var added int64
	for _, m := range members {
		if !e.Set[m] {
			e.Set[m] = true
			added++
		}
	}
	s.data[key] = e
	return added, nil
}

func (s *Store) SMembers(key string) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.getLiveLocked(key)
	if !ok {
		return []string{}, nil
	}
	if e.Kind != "set" {
		return nil, errors.New("WRONGTYPE operation against a key holding the wrong kind of value")
	}
	out := make([]string, 0, len(e.Set))
	for m := range e.Set {
		out = append(out, m)
	}
	sort.Strings(out)
	return out, nil
}

func (s *Store) Save(path string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UnixNano()
	cleaned := make(map[string]entry, len(s.data))
	for k, e := range s.data {
		if e.Expires != 0 && now >= e.Expires {
			continue
		}
		cleaned[k] = e
	}
	raw, err := json.MarshalIndent(cleaned, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func (s *Store) Load(path string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var loaded map[string]entry
	if err := json.Unmarshal(raw, &loaded); err != nil {
		return err
	}
	now := time.Now().UnixNano()
	s.mu.Lock()
	defer s.mu.Unlock()
	for k, e := range loaded {
		if e.Expires != 0 && now >= e.Expires {
			continue
		}
		s.data[k] = e
	}
	return nil
}
