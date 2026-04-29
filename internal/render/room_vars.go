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

// Room-info template variable assembly. Extracted from the handler
// package  — DRY builders own
// LW response and template-var construction so the handler stays a
// thin orchestrator.

package render

import (
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/ldmonster/cossacks-game-server/internal/domain/lobby"
)

// SetRoomPlayersColumn returns a copy of `row` with the players-count
// column updated to "P/M". Row layout places the
// players column at len-4; for shorter rows we fall back to len-1.
func SetRoomPlayersColumn(row []string, playersCount, maxPlayers int) []string {
	if len(row) == 0 {
		return row
	}

	out := append([]string(nil), row...)

	idx := len(out) - 4
	if idx < 0 {
		idx = len(out) - 1
	}

	out[idx] = fmt.Sprintf("%d/%d", playersCount, maxPlayers)

	return out
}

// StartedPlayerNames returns (active, exited) player nicks from a
// started room, ordered by join time.
func StartedPlayerNames(room *lobby.Room) ([]string, []string) {
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

// RoomTimeInterval formats the elapsed time since the room was created
// (or since it started, if applicable).
func RoomTimeInterval(room *lobby.Room) string {
	base := room.Ctime
	if room.Started && !room.StartedAt.IsZero() {
		base = room.StartedAt
	}

	secs := int(time.Since(base).Seconds())

	return TimeIntervalFromElapsedSec(secs)
}

// MergeRoomDottedVars writes the earlier TT "room.*" dotted-key
// variables into `vars` for templates that approximate TT
// `room` object.
func MergeRoomDottedVars(dev bool, room *lobby.Room, vars map[string]string) {
	if vars == nil {
		return
	}

	if dev {
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
	// No AI field on Room yet; keep stable sentinel for the template.
	vars["room.ai"] = "0"

	vars["room.passwd"] = room.Password
	if p := room.Players[room.HostID]; p != nil {
		vars[fmt.Sprintf("room.players.%d.nick", room.HostID)] = p.Nick
	}

	vars["room.ctime"] = room.Ctime.UTC().Format("2006-01-02 15:04:05 UTC")
}
