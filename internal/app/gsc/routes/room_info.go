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
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/ldmonster/cossacks-game-server/internal/domain/lobby"
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
// started_room_info variant when the room has been launched). It is
// the body of the `room_info_dgl` route.
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

	if room.Started &&
		((conn.Session != nil && conn.Session.Dev) || r.deps.Game.ShowStartedRoomInfo) {
		tpl := "started_room_info.tmpl"
		if p["part"] == "statcols" {
			tpl = "started_room_info/statcols.tmpl"
		}

		page := normalizePageRoutes(p["page"])
		res := normalizeResRoutes(p["res"])
		activePlayers, exitedPlayers := startedPlayerNamesRoutes(room)

		exited := "0"
		if len(exitedPlayers) > 0 {
			exited = "1"
		}

		roomTicks := room.TimeTick
		if roomTicks == 0 && !room.StartedAt.IsZero() {
			roomTicks = uint32(time.Since(room.StartedAt).Seconds() * 25)
		}

		vars := map[string]string{
			"room_id":            fmt.Sprintf("%d", room.ID),
			"room_name":          room.Title,
			"room_players":       fmt.Sprintf("%d/%d", room.PlayersCount, room.MaxPlayers),
			"room_players_start": fmt.Sprintf("%d", room.StartPlayers),
			"room_host":          room.HostAddr,
			"room_ctime":         strconv.FormatInt(room.Ctime.Unix(), 10),
			"room_started":       strconv.FormatBool(room.Started),
			"room.id":            fmt.Sprintf("%d", room.ID),
			"room.title":         room.Title,
			"room.time":          fmt.Sprintf("%d", roomTicks),
			"room.level":         fmt.Sprintf("%d", room.Level),
			"room.map":           room.Map,
			"room_max_pl":        fmt.Sprintf("%d", room.MaxPlayers),
			"room_pl_count":      fmt.Sprintf("%d", room.PlayersCount),
			"room_time":          roomTimeIntervalRoutes(room),
			"active_players":     strings.Join(activePlayers, ", "),
			"exited_players":     strings.Join(exitedPlayers, ", "),
			"has_exited_players": exited,
			"backto":             backto,
			"page":               page,
			"res":                res,
		}
		mergeRoomDottedVarsRoutes(conn, room, vars)

		return render.Show(r.render(room.Ver, tpl, vars)), nil
	}

	roomVars := map[string]string{
		"room_id":       fmt.Sprintf("%d", room.ID),
		"room_name":     room.Title,
		"room_players":  fmt.Sprintf("%d/%d", room.PlayersCount, room.MaxPlayers),
		"room_host":     room.HostAddr,
		"room_ctime":    strconv.FormatInt(room.Ctime.Unix(), 10),
		"room_started":  strconv.FormatBool(room.Started),
		"room_max_pl":   fmt.Sprintf("%d", room.MaxPlayers),
		"room_pl_count": fmt.Sprintf("%d", room.PlayersCount),
		"room_time":     roomTimeIntervalRoutes(room),
		"backto":        backto,
	}
	mergeRoomDottedVarsRoutes(conn, room, roomVars)

	return render.Show(r.render(room.Ver, "room_info_dgl.tmpl", roomVars)), nil
}

// normalizePageRoutes mirrors handler.normalizePage; constrained to
// "1"/"2"/"3" with default "1".
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

// normalizeResRoutes mirrors handler.normalizeRes.
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

// startedPlayerNamesRoutes returns active and exited player nicks
// preserving join order.
func startedPlayerNamesRoutes(room *lobby.Room) ([]string, []string) {
	active := []string{}
	exited := []string{}

	type pair struct {
		id uint32
		t  time.Time
	}

	ordered := make([]pair, 0, len(room.PlayersTime))
	for id, t := range room.PlayersTime {
		ordered = append(ordered, pair{id: id, t: t})
	}

	sort.Slice(ordered, func(i, j int) bool { return ordered[i].t.Before(ordered[j].t) })

	for _, it := range ordered {
		pl := room.Players[it.id]
		if pl == nil {
			continue
		}

		if !pl.ExitedAt.IsZero() {
			exited = append(exited, pl.Nick)
		} else {
			active = append(active, pl.Nick)
		}
	}

	return active, exited
}

// roomTimeIntervalRoutes mirrors handler.roomTimeInterval.
func roomTimeIntervalRoutes(room *lobby.Room) string {
	base := room.Ctime
	if room.Started && !room.StartedAt.IsZero() {
		base = room.StartedAt
	}

	secs := int(time.Since(base).Seconds())

	return render.TimeIntervalFromElapsedSec(secs)
}

// mergeRoomDottedVarsRoutes mirrors handler.mergeRoomDottedVars.
func mergeRoomDottedVarsRoutes(conn *tconn.Connection, room *lobby.Room, vars map[string]string) {
	if vars == nil {
		return
	}

	if conn.Session != nil && conn.Session.Dev {
		vars["h.connection.data.dev"] = "1"
	} else {
		vars["h.connection.data.dev"] = ""
	}

	vars["room.id"] = fmt.Sprintf("%d", room.ID)
	vars["room.title"] = room.Title
	vars["room.host_id"] = fmt.Sprintf("%d", room.HostID)
	vars["room.host_addr_int"] = fmt.Sprintf("%d", room.HostAddrInt)

	vars["room.level"] = fmt.Sprintf("%d", room.Level)
	if room.Started {
		vars["room.started"] = "1"
	} else {
		vars["room.started"] = "0"
	}

	vars["room.start_players_count"] = strconv.Itoa(room.StartPlayers)
	vars["room.players_count"] = strconv.Itoa(room.PlayersCount)
	vars["room.max_players"] = strconv.Itoa(room.MaxPlayers)
	vars["room.map"] = room.Map
	vars["room.ai"] = "0"

	vars["room.passwd"] = room.Password
	if p := room.Players[room.HostID]; p != nil {
		vars[fmt.Sprintf("room.players.%d.nick", room.HostID)] = p.Nick
	}

	vars["room.ctime"] = room.Ctime.UTC().Format("2006-01-02 15:04:05 UTC")
}
