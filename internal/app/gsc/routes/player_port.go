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

package routes

import (
	"github.com/ldmonster/cossacks-game-server/internal/domain/lobby"
	"github.com/ldmonster/cossacks-game-server/internal/domain/player"
)

// PlayerLookup is the consumer port for routes that need to resolve a
// player or room by id. The handler-side adapter delegates to
// rooms.Store.
type PlayerLookup interface {
	// GetPlayer returns the player with the given id, or nil if no
	// such player is registered.
	GetPlayer(id uint32) *player.Player
	// GetRoom returns the room with the given id, or nil if no such
	// room exists.
	GetRoom(id uint32) *lobby.Room
	// GetRoomByHost returns the room hosted by the given player id, or
	// nil if no such room exists.
	GetRoomByHost(id uint32) *lobby.Room
	// NextPlayerID returns a fresh, unique player id.
	NextPlayerID() uint32
	// UpsertPlayer inserts or replaces a player record.
	UpsertPlayer(p *player.Player)
}
