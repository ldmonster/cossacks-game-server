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

// Per-tick STAT and end-of-game handlers — thin wrappers around
// internal/app/gsc/commands. Kept around as a thin shim so that
// existing call sites (and the wire tests) keep compiling.

package gsc

import (
	"context"

	gsccmds "github.com/ldmonster/cossacks-game-server/internal/app/gsc/commands"
	"github.com/ldmonster/cossacks-game-server/internal/domain/lobby"
	"github.com/ldmonster/cossacks-game-server/internal/domain/match"
	"github.com/ldmonster/cossacks-game-server/internal/domain/player"
	tconn "github.com/ldmonster/cossacks-game-server/internal/transport/tconn"
)

// handleStats is the test-facing entry point. Production dispatch calls
// handleStatsLocked directly.
func (c *Controller) handleStats(conn *tconn.Connection, args []string) error {
	return c.handleStatsLocked(conn, args)
}

func (c *Controller) handleStatsLocked(conn *tconn.Connection, args []string) error {
	cmd := gsccmds.Stats{
		Alive: c,
		Rooms: c.Store,
		Match: c.game,
		Log:   c.Log,
	}
	res := cmd.Handle(context.Background(), conn, nil, args)

	return res.Err
}

type endgameEvent = match.EndgameEvent

func (c *Controller) parseEndgame(conn *tconn.Connection, args []string) (endgameEvent, bool) {
	var id uint32
	if conn.Session != nil {
		id = uint32(conn.Session.PlayerID)
	}

	return c.game.ParseEndgame(
		args, id,
		func(pid uint32) *player.Player { return c.Store.GetPlayer(pid) },
		func(rid uint32) *lobby.Room { return c.Store.GetRoom(rid) },
	)
}
