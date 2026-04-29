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

import "github.com/ldmonster/cossacks-game-server/internal/domain/player"

// PlayerRepository is the read/write contract for player lookups. The
// in-memory store (*state.Store) satisfies this interface; test doubles
// can provide a minimal stub without instantiating the full game state.
type PlayerRepository interface {
	// PlayerByID returns (player, true) when present; (nil, false) when absent.
	PlayerByID(id player.PlayerID) (*player.Player, bool)
}
