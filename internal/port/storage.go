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

// Package port defines the abstract interfaces between the application core
// (server, handlers, services) and infrastructure adapters (in-memory storage,
// future Redis, STUN, etc.). Concrete implementations live in other packages
// and depend on these interfaces — not the other way around (Dependency
// Inversion Principle).
package port

import (
	"context"
	"time"
)

// KVStore is a small key/value store with optional per-key expiration.
//
// Used by STUN (hole-punch data exchange) and health probes. The
// in-memory implementation lives in internal/adapter/kvmemory; future Redis
// or Memcached adapters can satisfy the same contract without touching
// callers.
type KVStore interface {
	// Get returns the value stored under key. Returns ErrKeyNotFound when
	// the key is missing or expired.
	Get(ctx context.Context, key string) (string, error)
	// SetPX stores value under key with optional TTL (zero TTL = no expiry).
	SetPX(ctx context.Context, key, value string, ttl time.Duration) error
	// Ping reports whether the store is operational. Used for readiness probes.
	Ping(ctx context.Context) error
}
