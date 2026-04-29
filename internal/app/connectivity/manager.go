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

package connectivity

import (
	"sync"
	"time"

	tconn "github.com/ldmonster/cossacks-game-server/internal/transport/tconn"
)

// Manager owns the per-connection alive-keep-alive timers and the live
// connection registry keyed by player id. Methods
// are safe for concurrent use; the caller does not need to hold any
// other lock.
type Manager struct {
	mu     sync.Mutex
	timers map[uint32]*time.Timer
	byID   map[uint32]*tconn.Connection
	ttl    time.Duration
}

// NewManager constructs a Manager with the given alive-timer TTL. A
// zero ttl falls back to 150s.
func NewManager(ttl time.Duration) *Manager {
	if ttl <= 0 {
		ttl = 150 * time.Second
	}

	return &Manager{
		timers: map[uint32]*time.Timer{},
		byID:   map[uint32]*tconn.Connection{},
		ttl:    ttl,
	}
}

// TTL returns the configured alive-timer TTL.
func (m *Manager) TTL() time.Duration { return m.ttl }

// ArmTimer (re)starts the keep-alive timeout for playerID. onTimeout
// runs in its own goroutine when the timer fires.
func (m *Manager) ArmTimer(playerID uint32, onTimeout func()) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if t := m.timers[playerID]; t != nil {
		t.Stop()
	}

	m.timers[playerID] = time.AfterFunc(m.ttl, onTimeout)
}

// ClearTimer stops and removes the keep-alive timer for playerID.
func (m *Manager) ClearTimer(playerID uint32) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if t := m.timers[playerID]; t != nil {
		t.Stop()
	}

	delete(m.timers, playerID)
}

// Register associates a live connection with playerID.
func (m *Manager) Register(playerID uint32, conn *tconn.Connection) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.byID[playerID] = conn
}

// Unregister removes the conn mapping for playerID (idempotent).
func (m *Manager) Unregister(playerID uint32) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.byID, playerID)
}

// Conn looks up the live connection for playerID.
func (m *Manager) Conn(playerID uint32) (*tconn.Connection, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	c, ok := m.byID[playerID]

	return c, ok
}

// HasTimer reports whether playerID currently has an armed alive timer.
// Intended for tests and observability.
func (m *Manager) HasTimer(playerID uint32) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	_, ok := m.timers[playerID]

	return ok
}

// SetTTL replaces the alive-timer TTL. Existing timers are not rescheduled.
// Intended for tests; production code passes the desired TTL to NewManager.
func (m *Manager) SetTTL(d time.Duration) {
	if d <= 0 {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.ttl = d
}
