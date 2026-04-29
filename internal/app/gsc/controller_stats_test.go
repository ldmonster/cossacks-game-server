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

package gsc

import (
	"encoding/binary"
	"testing"
	"time"

	"github.com/ldmonster/cossacks-game-server/internal/domain/player"
	"github.com/ldmonster/cossacks-game-server/internal/domain/session"
	tconn "github.com/ldmonster/cossacks-game-server/internal/transport/tconn"
)

func encodeRawStat(
	tick uint32,
	pc uint8,
	playerID uint32,
	status uint8,
	scores uint16,
	population uint16,
	wood uint32,
	gold uint32,
	stone uint32,
	food uint32,
	iron uint32,
	coal uint32,
	peasants uint16,
	units uint16,
) string {
	buf := make([]byte, 42)
	binary.LittleEndian.PutUint32(buf[0:4], tick)
	buf[4] = pc
	binary.LittleEndian.PutUint32(buf[5:9], playerID)
	buf[9] = status
	binary.LittleEndian.PutUint16(buf[10:12], scores)
	binary.LittleEndian.PutUint16(buf[12:14], population)
	binary.LittleEndian.PutUint32(buf[14:18], wood)
	binary.LittleEndian.PutUint32(buf[18:22], gold)
	binary.LittleEndian.PutUint32(buf[22:26], stone)
	binary.LittleEndian.PutUint32(buf[26:30], food)
	binary.LittleEndian.PutUint32(buf[30:34], iron)
	binary.LittleEndian.PutUint32(buf[34:38], coal)
	binary.LittleEndian.PutUint16(buf[38:40], peasants)
	binary.LittleEndian.PutUint16(buf[40:42], units)
	return string(buf)
}

func TestStatsUpdatesPlayerStatAndAlive(t *testing.T) {
	c := newControllerForJoinTests()
	c.session.SetTTL(time.Second)

	player := &player.Player{ID: 1, Nick: "p1", ConnectedAt: time.Now()}
	c.Store.SetPlayer(player)
	room := makeRoom(c, 301, 1, "stats-room", "")
	room.Players[1] = player
	room.PlayersCount = 1
	c.Store.IndexRoomByID(room)
	c.Store.IndexRoomByHost(1, room)
	conn := &tconn.Connection{Session: &session.Session{PlayerID: 1, Nick: "p1"}}

	raw1 := encodeRawStat(100, 1, 1, 0, 10, 5, 200, 50, 100, 80, 30, 20, 5, 8)
	c.handleStats(conn, []string{raw1, "301"})
	if player.Stat == nil {
		t.Fatalf("expected player stat to be stored")
	}
	if player.Stat.Time != 100 || player.Stat.PlayerID != 1 {
		t.Fatalf("unexpected stat snapshot %+v", player.Stat)
	}
	if !c.session.HasTimer(1) {
		t.Fatalf("expected alive timer to be refreshed by stats")
	}

	raw2 := encodeRawStat(125, 1, 1, 0, 20, 9, 250, 70, 130, 100, 40, 30, 9, 12)
	c.handleStats(conn, []string{raw2, "301"})
	if player.Stat.RealScores == 0 {
		t.Fatalf("expected real_scores to be computed")
	}
	if player.Stat.Population2 != player.Stat.Units+player.Stat.Peasants {
		t.Fatalf("expected population2 compatibility field")
	}
}

func TestStatsRejectsDifferentPlayerID(t *testing.T) {
	c := newControllerForJoinTests()
	player := &player.Player{ID: 1, Nick: "p1", ConnectedAt: time.Now()}
	c.Store.SetPlayer(player)
	room := makeRoom(c, 302, 1, "stats-room", "")
	room.Players[1] = player
	c.Store.IndexRoomByID(room)
	conn := &tconn.Connection{Session: &session.Session{PlayerID: 1, Nick: "p1"}}

	raw := encodeRawStat(100, 1, 2, 0, 10, 5, 200, 50, 100, 80, 30, 20, 5, 8)
	c.handleStats(conn, []string{raw, "302"})
	if player.Stat != nil {
		t.Fatalf("expected stats for mismatched player to be ignored")
	}
}

func TestStatsScoreWrapCycleMatchesPerl(t *testing.T) {
	c := newControllerForJoinTests()
	player := &player.Player{ID: 1, Nick: "p1", ConnectedAt: time.Now()}
	c.Store.SetPlayer(player)
	room := makeRoom(c, 303, 1, "stats-wrap", "")
	room.Players[1] = player
	c.Store.IndexRoomByID(room)
	conn := &tconn.Connection{Session: &session.Session{PlayerID: 1, Nick: "p1"}}

	raw1 := encodeRawStat(100, 1, 1, 0, 65530, 10, 200, 50, 100, 80, 30, 20, 5, 8)
	c.handleStats(conn, []string{raw1, "303"})
	raw2 := encodeRawStat(120, 1, 1, 0, 5, 10, 200, 55, 100, 80, 30, 20, 5, 8)
	c.handleStats(conn, []string{raw2, "303"})

	if player.Stat == nil {
		t.Fatalf("expected stat after wrap update")
	}
	if player.StatCycle.Scores != 1 {
		t.Fatalf("expected score cycle increment, got %d", player.StatCycle.Scores)
	}
	if player.Stat.RealScores != 65541 {
		t.Fatalf("expected real_scores 65541, got %d", player.Stat.RealScores)
	}
}

func TestStatsResourceDeltaUsesSignedDifference(t *testing.T) {
	c := newControllerForJoinTests()
	player := &player.Player{ID: 1, Nick: "p1", ConnectedAt: time.Now()}
	c.Store.SetPlayer(player)
	room := makeRoom(c, 304, 1, "stats-delta", "")
	room.Players[1] = player
	c.Store.IndexRoomByID(room)
	conn := &tconn.Connection{Session: &session.Session{PlayerID: 1, Nick: "p1"}}

	raw1 := encodeRawStat(200, 1, 1, 0, 10, 10, 200, 300, 150, 120, 90, 70, 10, 10)
	raw2 := encodeRawStat(220, 1, 1, 0, 15, 10, 195, 280, 140, 115, 80, 65, 10, 10)
	c.handleStats(conn, []string{raw1, "304"})
	c.handleStats(conn, []string{raw2, "304"})

	if player.Stat == nil {
		t.Fatalf("expected stat after delta update")
	}
	if player.Stat.ChangeGold >= 0 {
		t.Fatalf("expected negative gold change, got %f", player.Stat.ChangeGold)
	}
	if player.Stat.ChangeIron >= 0 {
		t.Fatalf("expected negative iron change, got %f", player.Stat.ChangeIron)
	}
	if player.Stat.ChangeCoal >= 0 {
		t.Fatalf("expected negative coal change, got %f", player.Stat.ChangeCoal)
	}
}
