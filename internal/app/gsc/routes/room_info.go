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

	"github.com/ldmonster/cossacks-game-server/internal/render"
	"github.com/ldmonster/cossacks-game-server/internal/transport/gsc"
	tconn "github.com/ldmonster/cossacks-game-server/internal/transport/tconn"
)

// ErrInvalidRoomID signals that VE_RID could not be parsed as a
// positive integer.
var ErrInvalidRoomID = errors.New("routes: invalid VE_RID")

// ErrRoomNotFound signals that the requested room ID is not registered.
var ErrRoomNotFound = errors.New("routes: room not found")

// RoomInfoImpl renders the room_info_dgl route (or the
// started_room_info variant when the room has been launched).
func (r *Routes) RoomInfoImpl(
	_ context.Context, conn *tconn.Connection, req *gsc.Stream, p map[string]string,
) ([]gsc.Command, error) {
	reqVer := uint8(2)
	if req != nil {
		reqVer = req.Ver
	}

	veRID := p["VE_RID"]

	rid, ridErr := strconv.ParseUint(veRID, 10, 32)
	if ridErr != nil {
		return render.Show("<NGDLG>\n<NGDLG>"), fmt.Errorf("%w: %q", ErrInvalidRoomID, veRID)
	}

	room := r.deps.Players.GetRoom(uint32(rid))
	if room == nil {
		return r.renderAlert(reqVer, "Error", "The room is closed"),
			fmt.Errorf("%w: %d", ErrRoomNotFound, uint32(rid))
	}

	backto := ""
	if p["BACKTO"] == "user_details" {
		if conn.Session != nil && conn.Session.PlayerID != 0 {
			backto = fmt.Sprintf("open&user_details.dcml&ID=%d", uint32(conn.Session.PlayerID))
		}
	}

	dev := conn.Session != nil && conn.Session.Dev

	if room.Started && (dev || r.deps.Game.ShowStartedRoomInfo) {
		tpl := "started_room_info.tmpl"
		if p["part"] == "statcols" {
			tpl = "started_room_info/statcols.tmpl"
		}

		page := normalizePageRoutes(p["page"])
		res := normalizeResRoutes(p["res"])

		view := &render.View{
			Dev:         dev,
			Room:        render.BuildRoomView(room, backto),
			StartedRoom: render.BuildStartedRoomView(room, page, res),
		}

		return render.Show(r.render(room.Ver, tpl, view)), nil
	}

	view := &render.View{
		Dev:  dev,
		Room: render.BuildRoomView(room, backto),
	}

	return render.Show(r.render(room.Ver, "room_info_dgl.tmpl", view)), nil
}

// normalizePageRoutes constrains `page` to "1"/"2"/"3" with default "1".
func normalizePageRoutes(v string) string {
	s := strings.TrimSpace(v)
	if s == "" {
		return "1"
	}

	if _, err := strconv.ParseUint(s, 10, 64); err != nil {
		return "1"
	}

	if s != "1" && s != "2" && s != "3" {
		return "1"
	}

	return s
}

// normalizeResRoutes coerces a numeric `res` string with default "0".
func normalizeResRoutes(v string) string {
	s := strings.TrimSpace(v)
	if s == "" {
		return "0"
	}

	if _, err := strconv.ParseUint(s, 10, 64); err != nil {
		return "0"
	}

	return s
}
