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

package routes

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/ldmonster/cossacks-game-server/internal/domain/player"
	"github.com/ldmonster/cossacks-game-server/internal/transport/gsc"
	tconn "github.com/ldmonster/cossacks-game-server/internal/transport/tconn"
)

// ErrJoinPlayerNoRoom signals that the requested player does not host
// any room.
var ErrJoinPlayerNoRoom = errors.New("routes: join_pl_cmd: no room")

// ErrRoomStarted signals that the requested room has already started.
var ErrRoomStarted = errors.New("routes: room already started")

// JoinPlayerImpl implements Open join_pl_cmd. The reference behaviour first
// short-circuits with an empty response when the caller is already in
// a room, otherwise it delegates to room_info_dgl using the host's
// room.
func (r *Routes) JoinPlayerImpl(
	ctx context.Context, conn *tconn.Connection, _ *gsc.Stream, p map[string]string,
) ([]gsc.Command, error) {
	if conn.Session != nil && conn.Session.PlayerID != 0 {
		if _, ok := r.deps.Lobby.FindByHost(conn.Session.PlayerID); ok {
			return []gsc.Command{}, nil
		}
	}

	vePlayer := strings.TrimSpace(p["VE_PLAYER"])

	joinedID, playerErr := strconv.ParseUint(vePlayer, 10, 32)
	if playerErr != nil {
		return nil, ErrInvalidPlayerArg{Raw: vePlayer}
	}

	room, ok := r.deps.Lobby.FindByHost(player.PlayerID(uint32(joinedID)))
	if !ok || room == nil {
		return nil, fmt.Errorf("%w: host=%d", ErrJoinPlayerNoRoom, uint32(joinedID))
	}

	if room.Started {
		return r.renderAlert(room.Ver, "Error", "Game alredy started"),
			fmt.Errorf("%w: id=%d", ErrRoomStarted, room.ID)
	}

	return r.RoomInfoImpl(ctx, conn, &gsc.Stream{Ver: room.Ver}, map[string]string{
		"VE_RID": fmt.Sprintf("%d", room.ID),
	})
}
