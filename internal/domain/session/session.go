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

// Package session holds the Session value (per-connection state).
//
// Note: an earlier package layout listed Session under domain/player.
// Placing it there creates an import cycle (player -> lobby for RoomID,
// lobby -> player for *Player). Session is therefore split into its own
// subpackage that imports identity, player, and lobby.
package session

import (
	"time"

	"github.com/ldmonster/cossacks-game-server/internal/domain/identity"
	"github.com/ldmonster/cossacks-game-server/internal/domain/lobby"
	"github.com/ldmonster/cossacks-game-server/internal/domain/player"
)

type Session struct {
	Account  *identity.AccountInfo // nil when not authenticated
	PlayerID player.PlayerID       // 0 until "start" hand-off assigns the in-game id
	Nick     string
	WindowW  int
	WindowH  int
	Dev      bool
	RoomID   lobby.RoomID // zero when not in a room
	AliveAt  time.Time    // last "alive" tick from the client
}

// New constructs a fresh empty session.
func New() *Session { return &Session{} }
