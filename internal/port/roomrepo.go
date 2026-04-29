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

package port

import (
	"github.com/ldmonster/cossacks-game-server/internal/domain/lobby"
	"github.com/ldmonster/cossacks-game-server/internal/domain/player"
)

// RoomRepository is the read/write contract for room lookups by the various
// index keys used by the game server (primary ID, hosting player, control
// sum). The in-memory *state.Store satisfies this interface via
// RoomRepoAdapter; room.MemoryRepository also satisfies it for tests.
type RoomRepository interface {
	// RoomByID returns (room, true) when present; (nil, false) when absent.
	RoomByID(id lobby.RoomID) (*lobby.Room, bool)
	// RoomByPlayerID returns the room currently hosted by the player, if any.
	RoomByPlayerID(id player.PlayerID) (*lobby.Room, bool)
}
