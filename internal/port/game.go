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
	"github.com/ldmonster/cossacks-game-server/internal/domain/match"
	"github.com/ldmonster/cossacks-game-server/internal/domain/player"
)

// GameService is the contract for in-game stat / endgame / start logic.
// The production implementation is game.Service; tests may substitute a
// stub without importing the game package.
type GameService interface {
	// UpdateStat decodes a raw STAT payload and applies it to the player's
	// state. Returns match.StatApplied on success or a rejection code.
	UpdateStat(player *player.Player, room *lobby.Room, raw []byte) match.StatRejection
	// ApplyStartedUsers parses the host-side start payload and populates
	// room.StartedUsers.
	ApplyStartedUsers(room *lobby.Room, args []string)
	// BuildStartAccountPayload assembles the payload posted to the account
	// endpoint when a host starts a game. Returns nil when inputs are
	// malformed.
	BuildStartAccountPayload(
		room *lobby.Room,
		args []string,
		sav, mapName string,
		findPlayer func(uint32) *player.Player,
	) map[string]any
	// ParseEndgame builds an EndgameEvent from raw GSC args. Returns
	// (EndgameEvent{}, false) when the payload is malformed.
	ParseEndgame(
		args []string,
		connectedPlayerID uint32,
		getPlayer func(uint32) *player.Player,
		getRoom func(uint32) *lobby.Room,
	) (match.EndgameEvent, bool)
}
