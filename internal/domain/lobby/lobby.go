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

// Package lobby holds Room and its identifier value objects.
package lobby

import (
	"time"

	"github.com/ldmonster/cossacks-game-server/internal/domain/player"
)

// RoomID is a typed identifier for a room.
type RoomID uint32

// Room is the in-memory representation of a game room (lobby + match).
//
// Concurrency primitives do not live on the domain entity. Per-room locks
// are provided by the room store / repository (see state.Store.RoomMu and
// room.MemoryRepository.RoomMu).
type Room struct {
	ID           uint32
	Title        string
	HostID       uint32
	HostAddr     string
	HostAddrInt  uint32
	Ver          uint8
	Level        int
	Password     string
	MaxPlayers   int
	PlayersCount int
	Players      map[uint32]*player.Player
	PlayersTime  map[uint32]time.Time
	Started      bool
	StartedAt    time.Time
	StartPlayers int
	CtlSum       uint32
	Row          []string
	Ctime        time.Time
	Map          string
	SaveFrom     int
	TimeTick     uint32
	StartedUsers []*player.Player
}
