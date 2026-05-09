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

// Typed-view builders for lobby and in-game rooms. Replaces the
// earlier dotted-key flatteners (`MergeRoomDottedVars`,
// `MergeStartedRoomDottedVars`) so templates address the data with
// idiomatic Go-template syntax (`{{.Room.ID}}`).

package render

import (
	"sort"
	"time"

	"github.com/ldmonster/cossacks-game-server/internal/domain/lobby"
)

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

// BuildRoomView projects a lobby `*lobby.Room` into the typed view
// consumed by `room_info_dgl.tmpl` (and the `.Room.*` references
// inside `started_room_info.tmpl`).
func BuildRoomView(room *lobby.Room, backto string) *RoomView {
	if room == nil {
		return nil
	}

	active, exited := StartedPlayerNames(room)

	v := &RoomView{
		ID:             room.ID,
		Title:          room.Title,
		HostID:         room.HostID,
		HostAddr:       room.HostAddr,
		HostAddrInt:    room.HostAddrInt,
		Level:          room.Level,
		Started:        room.Started,
		StartPlayers:   room.StartPlayers,
		PlayersCount:   room.PlayersCount,
		MaxPlayers:     room.MaxPlayers,
		Map:            room.Map,
		Password:       room.Password,
		Backto:         backto,
		HasExited:      len(exited) > 0,
		CtimeFormatted: room.Ctime.UTC().Format("2006-01-02 15:04:05 UTC"),
		Time:           RoomTimeInterval(room),
		ActivePlayers:  active,
		ExitedPlayers:  exited,
	}

	if hp := room.Players[room.HostID]; hp != nil {
		v.HostNick = hp.Nick
	}

	if len(room.Players) > 0 {
		v.Players = make(map[uint32]*RoomPlayerView, len(room.Players))
		for id, pl := range room.Players {
			if pl == nil {
				continue
			}

			v.Players[id] = &RoomPlayerView{Nick: pl.Nick}
		}
	}

	return v
}

// BuildStartedRoomView projects an in-game room into the typed view
// consumed by `started_room_info.tmpl` and its `statcols` partial.
func BuildStartedRoomView(room *lobby.Room, page, res string) *StartedRoomView {
	if room == nil {
		return nil
	}

	if page == "" {
		page = "1"
	}

	roomTicks := room.TimeTick
	if roomTicks == 0 && !room.StartedAt.IsZero() {
		roomTicks = uint32(time.Since(room.StartedAt).Seconds() * 25)
	}

	v := &StartedRoomView{
		ID:           room.ID,
		Title:        room.Title,
		Map:          room.Map,
		Level:        room.Level,
		RoomTime:     RoomTimeInterval(room),
		TimeTickSecs: roomTicks / 25,
		SmallColW:    49,
		LargeColW:    54,
		Page:         page,
		Res:          res,
	}

	if len(room.StartedUsers) == 0 {
		return v
	}

	v.Players = make([]*StartedPlayerView, 0, len(room.StartedUsers))

	for _, pl := range room.StartedUsers {
		if pl == nil {
			v.Players = append(v.Players, nil)
			continue
		}

		sp := &StartedPlayerView{
			Nick:   pl.Nick,
			Color:  pl.Color,
			Theam:  pl.Theam,
			Nation: pl.Nation,
			Zombie: pl.Zombie,
			Exited: !pl.ExitedAt.IsZero(),
		}

		if pl.Stat != nil {
			sp.Stat = &StartedPlayerStatView{
				RealScores: pl.Stat.RealScores,
				Population: pl.Stat.Population,
			}
		}

		v.Players = append(v.Players, sp)
	}

	return v
}
