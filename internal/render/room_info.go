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

// Room-info dialog template-variable builders. Extracted from the
// handler package.

package render

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/ldmonster/cossacks-game-server/internal/domain/lobby"
)

// RoomInfoBackto computes the "backto" template variable used by
// room_info_dgl when the user came from user_details. Returns "" when
// no backto link is required.
func RoomInfoBackto(backto string, viewerPlayerID uint32) string {
	if backto != "user_details" {
		return ""
	}

	if viewerPlayerID == 0 {
		return ""
	}

	return fmt.Sprintf("open&user_details.dcml&ID=%d", viewerPlayerID)
}

// BuildStartedRoomInfoVars assembles the template variables for
// `started_room_info.tmpl` (and its `statcols` partial). The caller
// is responsible for selecting the template name and supplying
// already-normalised `page` / `res` values.
func BuildStartedRoomInfoVars(
	dev bool,
	room *lobby.Room,
	page, res, backto string,
) map[string]string {
	activePlayers, exitedPlayers := StartedPlayerNames(room)

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
		"room_time":          RoomTimeInterval(room),
		"active_players":     strings.Join(activePlayers, ", "),
		"exited_players":     strings.Join(exitedPlayers, ", "),
		"has_exited_players": exited,
		"backto":             backto,
		"page":               page,
		"res":                res,
	}
	MergeRoomDottedVars(dev, room, vars)

	return vars
}

// BuildRoomInfoVars assembles the template variables for the earlier
// `room_info_dgl.tmpl` (lobby room view).
func BuildRoomInfoVars(dev bool, room *lobby.Room, backto string) map[string]string {
	vars := map[string]string{
		"room_id":       fmt.Sprintf("%d", room.ID),
		"room_name":     room.Title,
		"room_players":  fmt.Sprintf("%d/%d", room.PlayersCount, room.MaxPlayers),
		"room_host":     room.HostAddr,
		"room_ctime":    strconv.FormatInt(room.Ctime.Unix(), 10),
		"room_started":  strconv.FormatBool(room.Started),
		"room_max_pl":   fmt.Sprintf("%d", room.MaxPlayers),
		"room_pl_count": fmt.Sprintf("%d", room.PlayersCount),
		"room_time":     RoomTimeInterval(room),
		"backto":        backto,
	}
	MergeRoomDottedVars(dev, room, vars)

	return vars
}
