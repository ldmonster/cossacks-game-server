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

// Room lifecycle: register, join, info, join-by-player, leave, plus connection
// OnDisconnect cleanup.

package gsc

import (
	"context"

	"github.com/ldmonster/cossacks-game-server/internal/domain/player"
	"github.com/ldmonster/cossacks-game-server/internal/transport/gsc"
	tconn "github.com/ldmonster/cossacks-game-server/internal/transport/tconn"
)

func (c *Controller) OnDisconnect(conn *tconn.Connection) {
	// reference ConnectionController::_close leaves room and removes player.
	if conn.Session == nil || conn.Session.PlayerID == 0 {
		return
	}

	id := uint32(conn.Session.PlayerID)

	c.clearAliveTimer(id)
	c.session.Unregister(id)
	c.leaveRoomByID(id)
	c.Store.DeletePlayer(id)
}

func (c *Controller) joinGame(
	ctx context.Context,
	conn *tconn.Connection,
	req *gsc.Stream,
	p map[string]string,
) ([]gsc.Command, error) {
	return c.routes.JoinGameImpl(ctx, conn, req, p)
}

// roomInfo delegates to the routes-package implementation. It remains
// here as a thin shim so existing tests and callers keep working
// during the migration.
func (c *Controller) roomInfo(
	conn *tconn.Connection,
	req *gsc.Stream,
	p map[string]string,
) ([]gsc.Command, error) {
	return c.routes.RoomInfoImpl(context.Background(), conn, req, p)
}

// joinPlayer implements Open join_pl_cmd (Open.pm). VE_PLAYER is a player id;
// the room is rooms_by_player{VE_PLAYER} (Store.RoomsByPID), then the same as
// room_info_dgl with VE_RID => that room, or an error if started, or no response
// if there is no room, or an empty response if the caller is already in a room.
// joinPlayer delegates to the routes-package implementation. It
// remains here as a thin shim so existing tests and callers keep
// working during the migration.
func (c *Controller) joinPlayer(
	conn *tconn.Connection,
	p map[string]string,
) ([]gsc.Command, error) {
	return c.routes.JoinPlayerImpl(context.Background(), conn, nil, p)
}

func (c *Controller) leaveRoom(conn *tconn.Connection) {
	if conn.Session == nil || conn.Session.PlayerID == 0 {
		return
	}

	c.leaveRoomByID(uint32(conn.Session.PlayerID))
}

func (c *Controller) leaveRoomByID(playerID uint32) {
	c.clearAliveTimer(playerID)
	c.rooms.Leave(player.PlayerID(playerID))
}
