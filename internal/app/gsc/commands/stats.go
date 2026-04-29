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
	"fmt"
	"sync"

	"go.uber.org/zap"

	"github.com/ldmonster/cossacks-game-server/internal/app/match"
	"github.com/ldmonster/cossacks-game-server/internal/domain/lobby"
	matchdom "github.com/ldmonster/cossacks-game-server/internal/domain/match"
	"github.com/ldmonster/cossacks-game-server/internal/domain/player"
	"github.com/ldmonster/cossacks-game-server/internal/port"
	"github.com/ldmonster/cossacks-game-server/internal/transport/gsc"
	tconn "github.com/ldmonster/cossacks-game-server/internal/transport/tconn"
)

// StatsRoomLookup resolves rooms and exposes the per-room mutex used
// to serialize stat updates within a single room. *rooms.Store
// satisfies this interface.
type StatsRoomLookup interface {
	GetRoom(id uint32) *lobby.Room
	RoomMu(id uint32) *sync.RWMutex
}

// StatsRunner is the slim port that applies a STAT payload to a
// player's state. *match.Service and port.GameService both satisfy it.
type StatsRunner interface {
	UpdateStat(pl *player.Player, room *lobby.Room, raw []byte) matchdom.StatRejection
}

// StatsAliveDriver refreshes the alive timer for the connection that
// produced the STAT.
type StatsAliveDriver interface {
	RefreshAlive(conn *tconn.Connection)
}

// Stats handles the per-tick STAT command. It is dispatched outside of
// the global state mutex by the Controller because it acquires a
// per-room lock internally to allow stats from different rooms to be
// processed concurrently.
type Stats struct {
	Alive StatsAliveDriver
	Rooms StatsRoomLookup
	Match StatsRunner
	Log   *zap.Logger
}

// Name returns the GSC command name handled by this command.
func (Stats) Name() string { return "stats" }

// Handle applies a STAT payload to the player's room state. The
// per-room mutex is acquired internally; callers must NOT hold the
// Controller's global stateMu when invoking Handle.
func (s Stats) Handle(
	_ context.Context,
	conn *tconn.Connection,
	_ *gsc.Stream,
	args []string,
) port.HandleResult {
	s.Alive.RefreshAlive(conn)

	if len(args) < 2 {
		return port.HandleResult{HasResponse: false, Err: ErrStatsBadArgs{}}
	}

	roomID, err := parseUint32Arg(args[1])
	if err != nil {
		return port.HandleResult{HasResponse: false, Err: ErrStatsBadArgs{}}
	}

	room := s.Rooms.GetRoom(roomID)
	if room == nil {
		return port.HandleResult{HasResponse: false, Err: ErrStatsUnknownRoom{ID: roomID}}
	}

	if conn.Session == nil || conn.Session.PlayerID == 0 {
		return port.HandleResult{HasResponse: false, Err: ErrStatsNoSession{}}
	}

	userID := uint32(conn.Session.PlayerID)

	mu := s.Rooms.RoomMu(roomID)
	mu.Lock()
	defer mu.Unlock()

	pl := room.Players[userID]
	if pl == nil {
		return port.HandleResult{HasResponse: false, Err: ErrStatsNotInRoom{}}
	}

	if s.Match.UpdateStat(pl, room, []byte(args[0])) == matchdom.StatRejectTickAhead {
		s.Log.Warn("player stat tick ahead of payload",
			zap.Uint32("player_time_tick", pl.TimeTick),
		)
	}

	return port.HandleResult{HasResponse: false}
}

// ErrStatsBadArgs signals that the per-tick STAT command had fewer
// arguments than the wire format requires.
type ErrStatsBadArgs struct{}

// Error implements the error interface.
func (ErrStatsBadArgs) Error() string { return "handler: stats: insufficient args" }

// ErrStatsUnknownRoom signals that a STAT carried a room id that does
// not exist.
type ErrStatsUnknownRoom struct{ ID uint32 }

// Error implements the error interface.
func (e ErrStatsUnknownRoom) Error() string {
	return fmt.Sprintf("handler: stats: unknown room %d", e.ID)
}

// ErrStatsNotInRoom signals that a STAT was received from a session
// whose player is not a member of the referenced room.
type ErrStatsNotInRoom struct{}

// Error implements the error interface.
func (ErrStatsNotInRoom) Error() string { return "handler: stats: caller not in room" }

// ErrStatsNoSession signals that the STAT command arrived on a
// connection without an authenticated session.
type ErrStatsNoSession struct{}

// Error implements the error interface.
func (ErrStatsNoSession) Error() string { return "handler: session expired" }

func parseUint32Arg(v string) (uint32, error) {
	i, err := match.IntArg(v)
	if err != nil {
		return 0, err
	}

	return uint32(i), nil
}
