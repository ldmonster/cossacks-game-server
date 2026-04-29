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

// Package session provides the typed in-memory implementation of
// port.SessionStore.
package connectivity

import (
	"sync"

	"github.com/ldmonster/cossacks-game-server/internal/domain/player"
	"github.com/ldmonster/cossacks-game-server/internal/domain/session"
)

// Store is the concrete in-memory SessionStore. It is safe for
// concurrent use. The zero value is not usable; call NewStore.
type Store struct {
	mu       sync.RWMutex
	sessions map[player.ConnectionID]*session.Session
}

// NewStore constructs an empty session store.
func NewStore() *Store {
	return &Store{sessions: map[player.ConnectionID]*session.Session{}}
}

// Get returns the session for the given connection id, or (nil, false)
// when none exists.
func (s *Store) Get(id player.ConnectionID) (*session.Session, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sess, ok := s.sessions[id]

	return sess, ok
}

// Set stores sess under id, overwriting any existing entry.
func (s *Store) Set(id player.ConnectionID, sess *session.Session) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.sessions[id] = sess
}

// Delete removes the session for id (no-op if absent). Called by the
// dispatcher on connection close.
func (s *Store) Delete(id player.ConnectionID) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.sessions, id)
}
