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

package commands

import (
	"context"

	"github.com/ldmonster/cossacks-game-server/internal/domain/lobby"
	"github.com/ldmonster/cossacks-game-server/internal/domain/player"
	"github.com/ldmonster/cossacks-game-server/internal/port"
	"github.com/ldmonster/cossacks-game-server/internal/transport/gsc"
	tconn "github.com/ldmonster/cossacks-game-server/internal/transport/tconn"
)

// LeaveAliveTimers is the narrow consumer port the Leave command needs
// from the connectivity subsystem. The production implementation is
// satisfied by *connectivity.Manager.
type LeaveAliveTimers interface {
	// ClearTimer removes any pending alive timer for the player.
	ClearTimer(playerID uint32)
}

// LeaveLobby is the narrow consumer port the Leave command needs from
// the lobby application service. *lobby.Service satisfies it.
type LeaveLobby interface {
	// Leave removes the player from any room they currently occupy and
	// returns the affected room (if any). The Leave command discards
	// the return values; they exist for other callers.
	Leave(id player.PlayerID) (*lobby.Room, bool)
}

// Leave implements the GSC "leave" command. The client signals it is
// leaving any room it currently occupies; the server clears alive
// timers and detaches the player from the lobby.
type Leave struct {
	Alive LeaveAliveTimers
	Lobby LeaveLobby
}

// Name implements gsc.CommandHandler.
func (Leave) Name() string { return "leave" }

// Handle implements gsc.CommandHandler.
func (l Leave) Handle(
	_ context.Context,
	conn *tconn.Connection,
	_ *gsc.Stream,
	_ []string,
) port.HandleResult {
	if conn.Session == nil || conn.Session.PlayerID == 0 {
		return port.HandleResult{HasResponse: false}
	}

	id := uint32(conn.Session.PlayerID)
	l.Alive.ClearTimer(id)
	l.Lobby.Leave(player.PlayerID(id))

	return port.HandleResult{HasResponse: false}
}
