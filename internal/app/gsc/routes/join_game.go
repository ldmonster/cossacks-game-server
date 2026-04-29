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
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"go.uber.org/zap"

	"github.com/ldmonster/cossacks-game-server/internal/render"
	"github.com/ldmonster/cossacks-game-server/internal/transport/gsc"
	tconn "github.com/ldmonster/cossacks-game-server/internal/transport/tconn"
)

// ErrRoomFull signals that the requested room is already at its
// maximum-player capacity.
var ErrRoomFull = errors.New("routes: room full")

// ErrBadRoomPassword signals that VE_PASSWD did not match the room's
// configured password.
var ErrBadRoomPassword = errors.New("routes: bad room password")

// JoinGameImpl renders the join_room dialog when a player joins an
// existing room. It is the body of the `join_game` route.
func (r *Routes) JoinGameImpl(
	ctx context.Context, conn *tconn.Connection, req *gsc.Stream, p map[string]string,
) ([]gsc.Command, error) {
	reqVer := uint8(2)
	if req != nil {
		reqVer = req.Ver
	}

	if p["ASTATE"] == "" || p["ASTATE"] == "0" {
		playerID := uint32(0)
		if conn.Session != nil {
			playerID = uint32(conn.Session.PlayerID)
		}

		return r.renderAlert(
			reqVer,
			"Error",
			"You can not create or join room!\nYou are already participate in some room\nPlease disconnect from that room first to create a new one",
		), ErrAlreadyInRoom{PlayerID: playerID}
	}

	if conn.Session == nil || conn.Session.PlayerID == 0 ||
		strings.TrimSpace(conn.Session.Nick) == "" {
		return r.renderAlert(
			reqVer,
			"Error",
			"Your was disconnected from the server. Enter again.",
		), ErrSessionExpired
	}

	playerID := uint32(conn.Session.PlayerID)

	rid, ridErr := strconv.ParseUint(strings.TrimSpace(p["VE_RID"]), 10, 32)
	if ridErr != nil {
		return render.Show("<NGDLG>\n<NGDLG>"), fmt.Errorf("%w: %q", ErrInvalidRoomID, p["VE_RID"])
	}

	room := r.deps.Players.GetRoom(uint32(rid))
	if room == nil {
		return r.renderAlert(
			reqVer,
			"Error",
			"You can not join this room!\nThe room is closed",
		), fmt.Errorf("%w: %d", ErrRoomNotFound, uint32(rid))
	}

	if room.Started {
		return r.renderAlert(
			room.Ver,
			"Error",
			"You can not join this room!\nThe game has already started",
		), fmt.Errorf("%w: id=%d", ErrRoomStarted, room.ID)
	}

	if room.PlayersCount >= room.MaxPlayers {
		return r.renderAlert(
			room.Ver,
			"Error",
			"You can not join this room!\nThe room is full",
		), fmt.Errorf("%w: id=%d", ErrRoomFull, room.ID)
	}

	if room.Password != "" && p["VE_PASSWD"] != room.Password {
		return render.Show(r.render(room.Ver, "confirm_password_dgl.tmpl", map[string]string{
			"id": fmt.Sprintf("%d", room.ID),
		})), fmt.Errorf("%w: id=%d", ErrBadRoomPassword, room.ID)
	}

	if r.deps.Rooms != nil {
		r.deps.Rooms.LeaveByPlayer(playerID)
	}

	r.deps.Lobby.Join(room, r.deps.Players.GetPlayer(playerID))

	ip := room.HostAddr
	port := 0
	storageHit := false

	if r.deps.Storage != nil {
		if raw, err := r.deps.Storage.Get(ctx, strconv.Itoa(int(room.HostID))); err == nil {
			var remote struct {
				Host string `json:"host"`
				Port int    `json:"port"`
			}
			if json.Unmarshal([]byte(raw), &remote) == nil {
				ip = remote.Host
				port = remote.Port
				storageHit = true
			}
		}
	}

	if r.deps.Log != nil {
		r.deps.Log.Debug("join endpoint",
			zap.Uint64("conn_id", conn.ID),
			zap.Uint32("room_id", room.ID),
			zap.Uint32("host_id", room.HostID),
			zap.Bool("storage_hit", storageHit),
			zap.String("ip", ip),
			zap.Int("port", port),
		)
	}

	return render.Show(r.render(room.Ver, "join_room.tmpl", map[string]string{
		"id":     fmt.Sprintf("%d", room.ID),
		"max_pl": fmt.Sprintf("%d", room.MaxPlayers),
		"name":   room.Title,
		"ip":     ip,
		"port":   fmt.Sprintf("%d", port),
	})), nil
}
