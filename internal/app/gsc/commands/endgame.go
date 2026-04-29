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

	"go.uber.org/zap"

	"github.com/ldmonster/cossacks-game-server/internal/domain/lobby"
	"github.com/ldmonster/cossacks-game-server/internal/domain/match"
	"github.com/ldmonster/cossacks-game-server/internal/domain/player"
	"github.com/ldmonster/cossacks-game-server/internal/port"
	"github.com/ldmonster/cossacks-game-server/internal/transport/gsc"
	tconn "github.com/ldmonster/cossacks-game-server/internal/transport/tconn"
)

// ErrEndgameBadArgs signals that the GSC "endgame" payload could not
// be parsed. Mirrors the earlier errEndgameBadArgs sentinel previously
// defined in the handler package.
type ErrEndgameBadArgs struct{}

// Error implements the error interface.
func (ErrEndgameBadArgs) Error() string { return "gsc: endgame: bad args" }

// EndgameParser is the narrow consumer port the Endgame command
// requires from the match service. *match.Service satisfies it.
type EndgameParser interface {
	// ParseEndgame builds an EndgameEvent from raw GSC args. Returns
	// (zero, false) when the payload is malformed.
	ParseEndgame(
		args []string,
		connectedPlayerID uint32,
		getPlayer func(uint32) *player.Player,
		getRoom func(uint32) *lobby.Room,
	) (match.EndgameEvent, bool)
}

// EndgamePlayerLookup resolves a player by id. *rooms.Store satisfies
// it via its existing GetPlayer method.
type EndgamePlayerLookup interface {
	GetPlayer(id uint32) *player.Player
}

// EndgameRoomLookup resolves a room by id. *rooms.Store satisfies it
// via its existing GetRoom method.
type EndgameRoomLookup interface {
	GetRoom(id uint32) *lobby.Room
}

// Endgame implements the GSC "endgame" command. The host transmits
// the final game outcome (winner, score, map) so the server can log
// it for ranking and observability.
type Endgame struct {
	Match   EndgameParser
	Players EndgamePlayerLookup
	Rooms   EndgameRoomLookup
	Log     *zap.Logger
}

// Name implements gsc.CommandHandler.
func (Endgame) Name() string { return "endgame" }

// Handle implements gsc.CommandHandler.
func (e Endgame) Handle(
	_ context.Context,
	conn *tconn.Connection,
	_ *gsc.Stream,
	args []string,
) port.HandleResult {
	var connectedID uint32
	if conn.Session != nil {
		connectedID = uint32(conn.Session.PlayerID)
	}

	ev, ok := e.Match.ParseEndgame(
		args, connectedID,
		func(pid uint32) *player.Player { return e.Players.GetPlayer(pid) },
		func(rid uint32) *lobby.Room { return e.Rooms.GetRoom(rid) },
	)
	if !ok {
		return port.HandleResult{HasResponse: false, Err: ErrEndgameBadArgs{}}
	}

	if e.Log != nil {
		e.Log.Info("send game result",
			zap.String("nick", ev.Nick),
			zap.Uint32("player_id", ev.PlayerID),
			zap.String("result", ev.Result),
			zap.String("own", ev.Own),
			zap.Int("game_id", ev.GameID),
			zap.String("title", ev.Title),
		)
	}

	return port.HandleResult{HasResponse: false}
}
