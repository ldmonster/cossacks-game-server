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
	"strings"

	"github.com/ldmonster/cossacks-game-server/internal/domain/lobby"
	"github.com/ldmonster/cossacks-game-server/internal/domain/player"
	"github.com/ldmonster/cossacks-game-server/internal/port"
	"github.com/ldmonster/cossacks-game-server/internal/transport/gsc"
	tconn "github.com/ldmonster/cossacks-game-server/internal/transport/tconn"
)

// StartRoomLookup resolves the room owned by a given host. *rooms.Store
// satisfies this interface.
type StartRoomLookup interface {
	GetRoomByHost(hostID uint32) *lobby.Room
}

// StartLobby marks a room as started. *lobby.Service satisfies it.
type StartLobby interface {
	MarkStarted(room *lobby.Room, sav, mapName string)
}

// StartAliveDriver arms the per-player alive timer.
type StartAliveDriver interface {
	ArmAliveTimer(playerID uint32)
}

// StartMatch hosts the wire-critical match payload helpers.
type StartMatch interface {
	ApplyStartedUsers(room *lobby.Room, args []string)
	BuildStartAccountPayload(
		room *lobby.Room,
		args []string,
		sav, mapName string,
		lookup func(id uint32) *player.Player,
	) map[string]any
}

// StartPlayerLookup resolves a player by id for payload enrichment.
type StartPlayerLookup interface {
	GetPlayer(id uint32) *player.Player
}

// StartAccountPoster posts the start_account action to the configured
// account endpoint. The Controller satisfies this via
// postAccountAction; in production the call is fire-and-forget.
type StartAccountPoster interface {
	PostAccountAction(conn *tconn.Connection, action string, payload map[string]any)
}

// Start implements the GSC `start` command. It promotes the host's
// room to "started" state, arms alive timers for all members, and
// posts the start_account action when configured.
type Start struct {
	Rooms   StartRoomLookup
	Lobby   StartLobby
	Alive   StartAliveDriver
	Match   StartMatch
	Players StartPlayerLookup
	Account StartAccountPoster
}

// Name returns the GSC command name handled by this command.
func (Start) Name() string { return "start" }

// Handle executes the start workflow, returning an empty response.
func (s Start) Handle(
	_ context.Context,
	conn *tconn.Connection,
	_ *gsc.Stream,
	args []string,
) port.HandleResult {
	if conn.Session == nil || conn.Session.PlayerID == 0 {
		return port.HandleResult{HasResponse: false, Err: ErrStartNoSession{}}
	}

	playerID := uint32(conn.Session.PlayerID)

	room := s.Rooms.GetRoomByHost(playerID)
	if room == nil {
		return port.HandleResult{HasResponse: false, Err: ErrStartNotHost{}}
	}

	sav := ""
	mapName := ""

	if len(args) > 0 {
		sav = strings.TrimRight(args[0], "\x00")
	}

	if len(args) > 1 {
		mapName = strings.TrimRight(args[1], "\x00")
	}

	s.Lobby.MarkStarted(room, sav, mapName)

	for pid := range room.Players {
		s.Alive.ArmAliveTimer(pid)
	}

	if playerID == room.HostID {
		s.Match.ApplyStartedUsers(room, args)
	}

	if s.Account != nil {
		payload := s.Match.BuildStartAccountPayload(room, args, sav, mapName, s.Players.GetPlayer)
		if payload != nil {
			s.Account.PostAccountAction(conn, "start", payload)
		}
	}

	return port.HandleResult{HasResponse: false}
}

// ErrStartNoSession signals that the GSC `start` command was invoked
// from a connection without an authenticated session.
type ErrStartNoSession struct{}

// Error implements the error interface.
func (ErrStartNoSession) Error() string {
	return "handler: start: no session"
}

// ErrStartNotHost signals that the GSC `start` command was invoked by
// a player who does not host any room.
type ErrStartNotHost struct{}

// Error implements the error interface.
func (ErrStartNotHost) Error() string {
	return "handler: start: caller is not a room host"
}
