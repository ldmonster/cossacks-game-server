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

package lobby

import (
	"sync"

	"github.com/ldmonster/cossacks-game-server/internal/domain/lobby"
	"github.com/ldmonster/cossacks-game-server/internal/domain/player"
)

// MemoryRepository is the in-process Repository implementation.
// It is independent of state.Store and exists so tests can construct a
// room.Service without standing up the full game state.
type MemoryRepository struct {
	mu     sync.RWMutex
	byID   map[lobby.RoomID]*lobby.Room
	byHost map[player.PlayerID]*lobby.Room
	bySum  map[uint32]*lobby.Room
	nextID uint32

	roomLocksMu sync.Mutex
	roomLocks   map[lobby.RoomID]*sync.RWMutex
}

// NewMemoryRepository constructs an empty MemoryRepository.
func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		byID:      map[lobby.RoomID]*lobby.Room{},
		byHost:    map[player.PlayerID]*lobby.Room{},
		bySum:     map[uint32]*lobby.Room{},
		roomLocks: map[lobby.RoomID]*sync.RWMutex{},
	}
}

// RoomMu returns the per-room mutex for the given id, allocating on first
// use. Callers acquire / release the lock directly. It supersedes the Mu
// field that previously lived on the domain Room aggregate.
func (r *MemoryRepository) RoomMu(id lobby.RoomID) *sync.RWMutex {
	r.roomLocksMu.Lock()
	defer r.roomLocksMu.Unlock()

	mu, ok := r.roomLocks[id]
	if !ok {
		mu = &sync.RWMutex{}
		r.roomLocks[id] = mu
	}

	return mu
}

func (r *MemoryRepository) FindByID(id lobby.RoomID) (*lobby.Room, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	v, ok := r.byID[id]

	return v, ok
}

func (r *MemoryRepository) FindByHost(playerID player.PlayerID) (*lobby.Room, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	v, ok := r.byHost[playerID]

	return v, ok
}

func (r *MemoryRepository) FindBySum(sum uint32) (*lobby.Room, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	v, ok := r.bySum[sum]

	return v, ok
}

func (r *MemoryRepository) IndexByID(rm *lobby.Room) {
	if rm == nil {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.byID[lobby.RoomID(rm.ID)] = rm
}

func (r *MemoryRepository) IndexByHost(playerID player.PlayerID, rm *lobby.Room) {
	if rm == nil {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.byHost[playerID] = rm
}

func (r *MemoryRepository) IndexBySum(rm *lobby.Room) {
	if rm == nil {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.bySum[rm.CtlSum] = rm
}

func (r *MemoryRepository) UnindexByID(id lobby.RoomID) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.byID, id)
}

func (r *MemoryRepository) UnindexByHost(playerID player.PlayerID) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.byHost, playerID)
}

func (r *MemoryRepository) UnindexBySum(sum uint32) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.bySum, sum)
}

func (r *MemoryRepository) NextRoomID() uint32 {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.nextID++

	return r.nextID
}
