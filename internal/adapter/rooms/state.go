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

package rooms

import (
	"strings"
	"sync"

	"github.com/ldmonster/cossacks-game-server/internal/domain/lobby"
	"github.com/ldmonster/cossacks-game-server/internal/domain/match"
	"github.com/ldmonster/cossacks-game-server/internal/domain/player"
)

// player.Player / lobby.Room / PlayerStat / PlayerStatCycle / StatHistoryPoint are
// aliases pointing at the canonical domain types.
type (
	Player           = player.Player
	PlayerStat       = match.PlayerStat
	PlayerStatCycle  = match.PlayerStatCycle
	StatHistoryPoint = match.StatHistoryPoint
	Room             = lobby.Room
)

// Store is the in-memory game-state store. Internal maps are unexported
// all access goes through the methods below.
type Store struct {
	mu sync.RWMutex

	lastPlayerID uint32
	lastRoomID   uint32
	players      map[uint32]*player.Player
	roomsByID    map[uint32]*lobby.Room
	roomsByPID   map[uint32]*lobby.Room
	roomsBySum   map[uint32]*lobby.Room

	roomLocksMu sync.Mutex
	roomLocks   map[uint32]*sync.RWMutex
}

func NewStore() *Store {
	return &Store{
		players:    map[uint32]*player.Player{},
		roomsByID:  map[uint32]*lobby.Room{},
		roomsByPID: map[uint32]*lobby.Room{},
		roomsBySum: map[uint32]*lobby.Room{},
		roomLocks:  map[uint32]*sync.RWMutex{},
	}
}

// RoomMu returns the per-room mutex for the given id, allocating on first
// use. The returned mutex is stable for the lifetime of the Store; callers
// acquire / release it directly. It supersedes the Mu field that previously
// lived on the domain Room aggregate.
func (s *Store) RoomMu(id uint32) *sync.RWMutex {
	s.roomLocksMu.Lock()
	defer s.roomLocksMu.Unlock()

	mu, ok := s.roomLocks[id]
	if !ok {
		mu = &sync.RWMutex{}
		s.roomLocks[id] = mu
	}

	return mu
}

// WithRoom locks the per-room mutex for the supplied id, looks up the
// room by id, and invokes fn with the (possibly nil) room. The lock
// is held for the duration of fn and released afterwards. WithRoom is
// the migration target identified by the room aggregate split: as
// callers move to it, the global handler.Controller.stateMu can be
// removed.
//
// fn may return a non-nil error; WithRoom propagates it verbatim.
func (s *Store) WithRoom(id uint32, fn func(*lobby.Room) error) error {
	mu := s.RoomMu(id)
	mu.Lock()
	defer mu.Unlock()

	return fn(s.GetRoom(id))
}

// WithRoomRead is the read-only variant of WithRoom.
func (s *Store) WithRoomRead(id uint32, fn func(*lobby.Room) error) error {
	mu := s.RoomMu(id)

	mu.RLock()
	defer mu.RUnlock()

	return fn(s.GetRoom(id))
}

// NextPlayerID returns ++last_player.
func (s *Store) NextPlayerID() uint32 {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.lastPlayerID++

	return s.lastPlayerID
}

// NextRoomID returns ++last_room .
func (s *Store) NextRoomID() uint32 {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.lastRoomID++

	return s.lastRoomID
}

// LastPlayerID / LastRoomID return the current high-water mark counters.
func (s *Store) LastPlayerID() uint32 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.lastPlayerID
}

func (s *Store) LastRoomID() uint32 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.lastRoomID
}

// SetLastPlayerID / SetLastRoomID set the counters (used by tests that
// reset state between sub-tests).
func (s *Store) SetLastPlayerID(v uint32) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.lastPlayerID = v
}

func (s *Store) SetLastRoomID(v uint32) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.lastRoomID = v
}

// --- player.Player accessors ---

// SetPlayer upserts p into the players map keyed by p.ID.
func (s *Store) SetPlayer(p *player.Player) {
	if p == nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.players[p.ID] = p
}

// UpsertPlayer is an alias for SetPlayer (kept for older call sites).
func (s *Store) UpsertPlayer(p *player.Player) { s.SetPlayer(p) }

// GetPlayer returns the player or nil if absent.
func (s *Store) GetPlayer(id uint32) *player.Player {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.players[id]
}

// FindPlayer returns (player, ok).
func (s *Store) FindPlayer(id uint32) (*player.Player, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	p, ok := s.players[id]

	return p, ok
}

// DeletePlayer removes the player; safe no-op when absent.
func (s *Store) DeletePlayer(id uint32) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.players, id)
}

// AllPlayers returns a snapshot slice of all players.
func (s *Store) AllPlayers() []*player.Player {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]*player.Player, 0, len(s.players))
	for _, p := range s.players {
		out = append(out, p)
	}

	return out
}

// --- lobby.Room accessors ---

// IndexRoomByID registers the room under its primary id.
func (s *Store) IndexRoomByID(r *lobby.Room) {
	if r == nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.roomsByID[r.ID] = r
}

// IndexRoomByHost maps the host player id to the room.
func (s *Store) IndexRoomByHost(pid uint32, r *lobby.Room) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.roomsByPID[pid] = r
}

// IndexRoomBySum registers the room under its current control-sum key.
func (s *Store) IndexRoomBySum(r *lobby.Room) {
	if r == nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.roomsBySum[r.CtlSum] = r
}

// GetRoom returns the room by id or nil.
func (s *Store) GetRoom(id uint32) *lobby.Room {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.roomsByID[id]
}

// FindRoom returns (room, ok).
func (s *Store) FindRoom(id uint32) (*lobby.Room, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	r, ok := s.roomsByID[id]

	return r, ok
}

// GetRoomByHost returns the room hosted by the given player or nil.
func (s *Store) GetRoomByHost(pid uint32) *lobby.Room {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.roomsByPID[pid]
}

// FindRoomByHost returns (room, ok).
func (s *Store) FindRoomByHost(pid uint32) (*lobby.Room, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	r, ok := s.roomsByPID[pid]

	return r, ok
}

// GetRoomBySum returns the room indexed by its control-sum key or nil.
func (s *Store) GetRoomBySum(sum uint32) *lobby.Room {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.roomsBySum[sum]
}

// FindRoomBySum returns (room, ok).
func (s *Store) FindRoomBySum(sum uint32) (*lobby.Room, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	r, ok := s.roomsBySum[sum]

	return r, ok
}

// UnindexRoomByID removes the by-id mapping; no-op when absent.
func (s *Store) UnindexRoomByID(id uint32) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.roomsByID, id)
}

// UnindexRoomByHost removes the by-host mapping; no-op when absent.
func (s *Store) UnindexRoomByHost(pid uint32) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.roomsByPID, pid)
}

// UnindexRoomBySum removes the by-sum mapping; no-op when absent.
func (s *Store) UnindexRoomBySum(sum uint32) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.roomsBySum, sum)
}

// AllRooms returns a snapshot slice of all rooms (by-id index).
func (s *Store) AllRooms() []*lobby.Room {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]*lobby.Room, 0, len(s.roomsByID))
	for _, r := range s.roomsByID {
		out = append(out, r)
	}

	return out
}

// --- Port adapters: strongly-typed accessors satisfying port.PlayerRepository
// and port.RoomRepository. The domain types are
// now canonical; no intermediate any cast is needed.

// PlayerByID satisfies port.PlayerRepository: returns (player, true) when
// present; (nil, false) when absent.
func (s *Store) PlayerByID(id player.PlayerID) (*player.Player, bool) {
	return s.FindPlayer(uint32(id))
}

// RoomByID satisfies port.RoomRepository: returns (room, true) when present;
// (nil, false) when absent.
func (s *Store) RoomByID(id lobby.RoomID) (*lobby.Room, bool) {
	return s.FindRoom(uint32(id))
}

// RoomByPlayerID satisfies port.RoomRepository: returns the room currently
// hosted by the given player, or (nil, false) when none found.
func (s *Store) RoomByPlayerID(id player.PlayerID) (*lobby.Room, bool) {
	return s.FindRoomByHost(uint32(id))
}

// RoomControlSum checksum used by the
// original server to detect concurrent edits to a room's row state.
func RoomControlSum(row []string) uint32 {
	const (
		mod   = 0xFFF1
		chunk = 5552
	)

	s := strings.Join(row, "")
	v1 := uint32(1)
	v2 := uint32(0)

	for i := 0; i < len(s); i += chunk {
		end := i + chunk
		if end > len(s) {
			end = len(s)
		}

		for j := i; j < end; j++ {
			v1 += uint32(s[j])
			v2 += v1
		}

		v1 %= mod
		v2 %= mod
	}

	return (v2 << 16) | v1
}
