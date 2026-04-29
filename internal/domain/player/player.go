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

// Package player holds the Player aggregate and its identifier value
// objects (PlayerID, ConnectionID).
package player

import (
	"time"

	"github.com/ldmonster/cossacks-game-server/internal/domain/match"
)

// PlayerID is a typed identifier for a player. Using a named type prevents
// accidental mixing with RoomID and other uint32 values throughout the code.
type PlayerID uint32

// ConnectionID is a typed identifier for a TCP connection (used as the
// session key).
type ConnectionID uint64

// Player is the in-memory representation of an authenticated client.
type Player struct {
	ID          uint32
	Nick        string
	ConnectedAt time.Time
	ExitedAt    time.Time
	// Typed account fields replaced the earlier Account map[string]any.
	AccountType    string
	AccountLogin   string
	AccountID      string
	AccountProfile string

	TimeTick    uint32
	Nation      uint32
	Theam       uint32
	Color       uint32
	Zombie      bool
	Stat        *match.PlayerStat
	StatCycle   match.PlayerStatCycle
	StatHistory map[string][]match.StatHistoryPoint
	StatSum     map[string]float64
}
