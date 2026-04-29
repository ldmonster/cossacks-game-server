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
	"sync"

	"github.com/ldmonster/cossacks-game-server/internal/domain/lobby"
	"github.com/ldmonster/cossacks-game-server/internal/domain/player"
)

// RoomRepoAdapter adapts *Store to the room.Repository interface
// It exists in this package to avoid an
// import cycle: the room package does not depend on state.
type RoomRepoAdapter struct{ s *Store }

// AsRoomRepo returns a *Store-backed adapter implementing the
// room.Repository contract.
func (s *Store) AsRoomRepo() *RoomRepoAdapter { return &RoomRepoAdapter{s: s} }

func (a *RoomRepoAdapter) FindByID(id lobby.RoomID) (*lobby.Room, bool) {
	return a.s.FindRoom(uint32(id))
}

func (a *RoomRepoAdapter) FindByHost(pid player.PlayerID) (*lobby.Room, bool) {
	return a.s.FindRoomByHost(uint32(pid))
}

func (a *RoomRepoAdapter) FindBySum(sum uint32) (*lobby.Room, bool) {
	return a.s.FindRoomBySum(sum)
}

func (a *RoomRepoAdapter) IndexByID(r *lobby.Room) { a.s.IndexRoomByID(r) }
func (a *RoomRepoAdapter) IndexByHost(pid player.PlayerID, r *lobby.Room) {
	a.s.IndexRoomByHost(uint32(pid), r)
}
func (a *RoomRepoAdapter) IndexBySum(r *lobby.Room)          { a.s.IndexRoomBySum(r) }
func (a *RoomRepoAdapter) UnindexByID(id lobby.RoomID)       { a.s.UnindexRoomByID(uint32(id)) }
func (a *RoomRepoAdapter) UnindexByHost(pid player.PlayerID) { a.s.UnindexRoomByHost(uint32(pid)) }
func (a *RoomRepoAdapter) UnindexBySum(sum uint32)           { a.s.UnindexRoomBySum(sum) }
func (a *RoomRepoAdapter) NextRoomID() uint32                { return a.s.NextRoomID() }
func (a *RoomRepoAdapter) RoomMu(id lobby.RoomID) *sync.RWMutex {
	return a.s.RoomMu(uint32(id))
}
