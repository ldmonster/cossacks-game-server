package commands

import (
	"encoding/binary"
	"testing"
	"time"

	"cossacksgameserver/golang/internal/server/model"
	"cossacksgameserver/golang/internal/server/state"
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
	c.aliveTTL = time.Second
	c.ensureRuntimeMaps()

	player := &state.Player{ID: 1, Nick: "p1", ConnectedAt: time.Now()}
	c.Store.Players[1] = player
	room := makeRoom(c, 301, 1, "stats-room", "")
	room.Players[1] = player
	room.PlayersCount = 1
	c.Store.RoomsByID[301] = room
	c.Store.RoomsByPID[1] = room
	conn := &model.Connection{Data: map[string]any{"id": uint32(1), "nick": "p1"}}

	raw1 := encodeRawStat(100, 1, 1, 0, 10, 5, 200, 50, 100, 80, 30, 20, 5, 8)
	c.handleStats(conn, []string{raw1, "301"})
	if player.Stat == nil {
		t.Fatalf("expected player stat to be stored")
	}
	if player.Stat.Time != 100 || player.Stat.PlayerID != 1 {
		t.Fatalf("unexpected stat snapshot %+v", player.Stat)
	}
	if _, ok := c.aliveTimers[1]; !ok {
		t.Fatalf("expected alive timer to be refreshed by stats")
	}

	raw2 := encodeRawStat(125, 1, 1, 0, 20, 9, 250, 70, 130, 100, 40, 30, 9, 12)
	c.handleStats(conn, []string{raw2, "301"})
	if player.Stat.RealScores == 0 {
		t.Fatalf("expected real_scores to be computed")
	}
	if player.Stat.Population2 != player.Stat.Units+player.Stat.Peasants {
		t.Fatalf("expected population2 parity field")
	}
}

func TestStatsRejectsDifferentPlayerID(t *testing.T) {
	c := newControllerForJoinTests()
	player := &state.Player{ID: 1, Nick: "p1", ConnectedAt: time.Now()}
	c.Store.Players[1] = player
	room := makeRoom(c, 302, 1, "stats-room", "")
	room.Players[1] = player
	c.Store.RoomsByID[302] = room
	conn := &model.Connection{Data: map[string]any{"id": uint32(1), "nick": "p1"}}

	raw := encodeRawStat(100, 1, 2, 0, 10, 5, 200, 50, 100, 80, 30, 20, 5, 8)
	c.handleStats(conn, []string{raw, "302"})
	if player.Stat != nil {
		t.Fatalf("expected stats for mismatched player to be ignored")
	}
}

func TestStatsScoreWrapCycleMatchesPerl(t *testing.T) {
	c := newControllerForJoinTests()
	player := &state.Player{ID: 1, Nick: "p1", ConnectedAt: time.Now()}
	c.Store.Players[1] = player
	room := makeRoom(c, 303, 1, "stats-wrap", "")
	room.Players[1] = player
	c.Store.RoomsByID[303] = room
	conn := &model.Connection{Data: map[string]any{"id": uint32(1), "nick": "p1"}}

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
	player := &state.Player{ID: 1, Nick: "p1", ConnectedAt: time.Now()}
	c.Store.Players[1] = player
	room := makeRoom(c, 304, 1, "stats-delta", "")
	room.Players[1] = player
	c.Store.RoomsByID[304] = room
	conn := &model.Connection{Data: map[string]any{"id": uint32(1), "nick": "p1"}}

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
