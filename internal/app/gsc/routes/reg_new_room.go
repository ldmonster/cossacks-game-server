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

	"go.uber.org/zap"

	lobbyapp "github.com/ldmonster/cossacks-game-server/internal/app/lobby"
	"github.com/ldmonster/cossacks-game-server/internal/render"
	"github.com/ldmonster/cossacks-game-server/internal/transport/gsc"
	tconn "github.com/ldmonster/cossacks-game-server/internal/transport/tconn"
)

// ErrSessionExpired signals that the requested route requires an
// authenticated session but the connection has none.
var ErrSessionExpired = errors.New("routes: session expired")

// ErrIllegalRoomTitle signals that VE_TITLE failed lobby title
// validation.
var ErrIllegalRoomTitle = errors.New("routes: illegal room title")

// RegNewRoomImpl renders the reg_new_room dialog and registers a new
// room in the lobby. It is the body of the `reg_new_room` route; the
// flow-port wrapper Routes.RegNewRoom will delegate here once the
// handler-side method is removed.
func (r *Routes) RegNewRoomImpl(
	_ context.Context, conn *tconn.Connection, req *gsc.Stream, p map[string]string,
) ([]gsc.Command, error) {
	if p["ASTATE"] == "" || p["ASTATE"] == "0" {
		playerID := uint32(0)
		if conn.Session != nil {
			playerID = uint32(conn.Session.PlayerID)
		}

		return r.renderAlert(
			req.Ver,
			"Error",
			"You can not create or join room!\nYou are already participate in some room\nPlease disconnect from that room first to create a new one",
		), ErrAlreadyInRoom{PlayerID: playerID}
	}

	if conn.Session == nil || conn.Session.PlayerID == 0 || conn.Session.Nick == "" {
		return r.renderAlert(
			req.Ver,
			"Error",
			"Your was disconnected from the server. Enter again.",
		), ErrSessionExpired
	}

	playerID := uint32(conn.Session.PlayerID)
	nickStr := conn.Session.Nick

	rawTitle := p["VE_TITLE"]
	if !lobbyapp.ValidateTitle(rawTitle) {
		return render.Show(r.render(req.Ver, "confirm_dgl.tmpl", map[string]string{
			"header":  "Error",
			"text":    "Illegal title!\nPress Edit button to check title",
			"ok_text": "Edit",
			"height":  "180",
			"command": "GW|open&new_room_dgl.dcml&ASTATE=<%ASTATE>",
		})), ErrIllegalRoomTitle
	}

	title := lobbyapp.NormalizeTitle(rawTitle)

	if r.deps.Rooms != nil {
		r.deps.Rooms.LeaveByPlayer(playerID)
	}

	maxPlayers := 8
	if v, err := strconv.Atoi(p["VE_MAX_PL"]); err == nil {
		maxPlayers = v + 2
	}

	level := 0
	if v, err := strconv.Atoi(p["VE_LEVEL"]); err == nil {
		level = v
	}

	levelLabel := lobbyapp.LevelLabel(level)

	lockMark := ""
	if p["VE_PASSWD"] != "" {
		lockMark = "#"
	}

	hostName := r.deps.Server.HostName
	if hostName == "" {
		hostName = conn.IP
	}

	hostPlayer := r.deps.Players.GetPlayer(playerID)

	room := r.deps.Lobby.RegisterNew(lobbyapp.RegisterNewParams{
		Host:        hostPlayer,
		HostAddr:    conn.IP,
		HostAddrInt: conn.IntIP,
		Ver:         req.Ver,
		Title:       title,
		Password:    p["VE_PASSWD"],
		MaxPlayers:  maxPlayers,
		Level:       level,
		LevelLabel:  levelLabel,
		LockMark:    lockMark,
		NickStr:     nickStr,
		VEType:      p["VE_TYPE"],
		IsAC:        render.IsAC(req.Ver),
	})

	if r.deps.Rooms != nil {
		r.deps.Rooms.ArmAliveTimer(playerID)
	}

	gameID := fmt.Sprintf("%d", room.ID)
	if p["VE_TYPE"] != "" {
		gameID = "HB" + gameID
	}

	if r.deps.Log != nil {
		r.deps.Log.Debug("createGame stun endpoint",
			zap.Uint32("player_id", playerID),
			zap.String("hole_host", hostName),
			zap.Int("hole_port", r.deps.Server.HolePort),
			zap.Int("hole_int", r.deps.Game.HoleInterval),
		)
	}

	return render.Show(r.render(req.Ver, "reg_new_room.tmpl", map[string]string{
		"player_id": fmt.Sprintf("%d", playerID),
		"hole_port": strconv.Itoa(r.deps.Server.HolePort),
		"hole_host": hostName,
		"hole_int":  strconv.Itoa(r.deps.Game.HoleInterval),
		"id":        gameID,
		"name":      title,
		"max_pl":    fmt.Sprintf("%d", maxPlayers),
	})), nil
}
