// Copyright 2026 Cossacks Game Server Contributors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package integration

import (
	"context"
	"errors"
	"sync"
	"time"
)

var (
	ErrStorageClosed = errors.New("runtime storage is closed")
	ErrKeyNotFound   = errors.New("key not found")
)

type RuntimeStorage struct {
	mu      sync.RWMutex
	closed  bool
	records map[string]storageRecord
}

type storageRecord struct {
	value     string
	expiresAt time.Time
}

func NewRuntimeStorage() *RuntimeStorage {
	return &RuntimeStorage{
		records: make(map[string]storageRecord),
	}
}

func (s *RuntimeStorage) Get(ctx context.Context, key string) (string, error) {
	_ = ctx

	if s == nil {
		return "", ErrStorageClosed
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return "", ErrStorageClosed
	}

	record, ok := s.records[key]
	if !ok {
		return "", ErrKeyNotFound
	}

	if !record.expiresAt.IsZero() && time.Now().After(record.expiresAt) {
		delete(s.records, key)
		return "", ErrKeyNotFound
	}

	return record.value, nil
}

func (s *RuntimeStorage) SetPX(ctx context.Context, key, value string, px time.Duration) error {
	_ = ctx

	if s == nil {
		return ErrStorageClosed
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return ErrStorageClosed
	}

	record := storageRecord{value: value}
	if px > 0 {
		record.expiresAt = time.Now().Add(px)
	}

	s.records[key] = record

	return nil
}

// Ping checks in-memory storage availability (used for readiness probes).
func (s *RuntimeStorage) Ping(ctx context.Context) error {
	_ = ctx

	if s == nil || s.closed {
		return ErrStorageClosed
	}

	return nil
}
