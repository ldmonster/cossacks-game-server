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

// Package room owns the room-lifecycle service. The
// handler package depends on Repository (interface).
package lobby

import (
	"sync"

	"github.com/ldmonster/cossacks-game-server/internal/domain/lobby"
	"github.com/ldmonster/cossacks-game-server/internal/domain/player"
)

// Repository is the persistence contract for rooms. It is small on
// purpose so test doubles can implement it without dragging in the
// rest of the state.Store surface.
type Repository interface {
	FindByID(id lobby.RoomID) (*lobby.Room, bool)
	FindByHost(playerID player.PlayerID) (*lobby.Room, bool)
	FindBySum(sum uint32) (*lobby.Room, bool)

	IndexByID(r *lobby.Room)
	IndexByHost(playerID player.PlayerID, r *lobby.Room)
	IndexBySum(r *lobby.Room)

	UnindexByID(id lobby.RoomID)
	UnindexByHost(playerID player.PlayerID)
	UnindexBySum(sum uint32)

	NextRoomID() uint32

	// RoomMu returns the per-room mutex for the given id, allocating on
	// first use. Supersedes the Mu field that previously lived on
	// the domain Room aggregate.
	RoomMu(id lobby.RoomID) *sync.RWMutex
}

// Service is the room-lifecycle facade. Construct via NewService.
type Service struct {
	repo Repository
}

// NewService returns a Service wrapping repo.
func NewService(repo Repository) *Service { return &Service{repo: repo} }

// Repo returns the wrapped Repository (nil-safe).
func (s *Service) Repo() Repository {
	if s == nil {
		return nil
	}

	return s.repo
}

// FindByID is a thin pass-through provided for callers that only need
// repository read access.
func (s *Service) FindByID(id lobby.RoomID) (*lobby.Room, bool) {
	if s == nil || s.repo == nil {
		return nil, false
	}

	return s.repo.FindByID(id)
}

// FindByHost looks up the room currently hosted by the given player.
func (s *Service) FindByHost(playerID player.PlayerID) (*lobby.Room, bool) {
	if s == nil || s.repo == nil {
		return nil, false
	}

	return s.repo.FindByHost(playerID)
}

// NextID allocates a fresh RoomID via the underlying repository.
func (s *Service) NextID() uint32 {
	if s == nil || s.repo == nil {
		return 0
	}

	return s.repo.NextRoomID()
}
