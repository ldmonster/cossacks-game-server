package commands

import (
	"testing"
	"time"

	"cossacksgameserver/golang/internal/config"
	"cossacksgameserver/golang/internal/server/model"
	"cossacksgameserver/golang/internal/server/state"
)

func newTestController() *Controller {
	return &Controller{
		Config: &config.Config{ShowStartedRooms: true, Raw: map[string]string{}},
		Store:  state.NewStore(),
	}
}

func TestLeaveRoomDeletesNonStartedWhenHostLeaves(t *testing.T) {
	c := newTestController()
	host := &state.Player{ID: 1, Nick: "host", ConnectedAt: time.Now()}
	guest := &state.Player{ID: 2, Nick: "guest", ConnectedAt: time.Now()}
	c.Store.Players[1] = host
	c.Store.Players[2] = guest

	room := &state.Room{
		ID:           10,
		Title:        "r",
		HostID:       1,
		MaxPlayers:   8,
		PlayersCount: 2,
		Players:      map[uint32]*state.Player{1: host, 2: guest},
		PlayersTime:  map[uint32]time.Time{1: time.Now(), 2: time.Now()},
		Row:          []string{"10", "", "r", "host", "For all", "2/8", "2"},
		Ctime:        time.Now(),
	}
	room.CtlSum = state.RoomControlSum(room.Row)
	c.Store.RoomsByID[room.ID] = room
	c.Store.RoomsByPID[1] = room
	c.Store.RoomsByPID[2] = room
	c.Store.RoomsBySum[room.CtlSum] = room

	conn := &model.Connection{Data: map[string]any{"id": uint32(1)}}
	c.leaveRoom(conn)

	if _, ok := c.Store.RoomsByID[room.ID]; ok {
		t.Fatalf("expected non-started room to be deleted when host leaves")
	}
}

func TestLeaveRoomKeepsStartedRoomWhenHostLeaves(t *testing.T) {
	c := newTestController()
	host := &state.Player{ID: 1, Nick: "host", ConnectedAt: time.Now()}
	guest := &state.Player{ID: 2, Nick: "guest", ConnectedAt: time.Now()}
	c.Store.Players[1] = host
	c.Store.Players[2] = guest

	room := &state.Room{
		ID:           11,
		Title:        "r2",
		HostID:       1,
		MaxPlayers:   8,
		PlayersCount: 2,
		Players:      map[uint32]*state.Player{1: host, 2: guest},
		PlayersTime:  map[uint32]time.Time{1: time.Now(), 2: time.Now()},
		Started:      true,
		StartedAt:    time.Now(),
		Row:          []string{"11", "\x7f0018", "r2", "host", "For all", "2/8", "1ABCDEF"},
		Ctime:        time.Now(),
	}
	room.CtlSum = state.RoomControlSum(room.Row)
	c.Store.RoomsByID[room.ID] = room
	c.Store.RoomsByPID[1] = room
	c.Store.RoomsByPID[2] = room
	c.Store.RoomsBySum[room.CtlSum] = room

	conn := &model.Connection{Data: map[string]any{"id": uint32(1)}}
	c.leaveRoom(conn)

	got := c.Store.RoomsByID[room.ID]
	if got == nil {
		t.Fatalf("expected started room to remain after host leaves")
	}
	if got.PlayersCount != 1 {
		t.Fatalf("expected players_count to decrement to 1, got %d", got.PlayersCount)
	}
	if got.Players[1] == nil || got.Players[1].ExitedAt.IsZero() {
		t.Fatalf("expected host to be marked exited in started room")
	}
}

func TestLeaveRoomDeletesStartedRoomWhenLastPlayerLeaves(t *testing.T) {
	c := newTestController()
	host := &state.Player{ID: 1, Nick: "host", ConnectedAt: time.Now()}
	c.Store.Players[1] = host

	room := &state.Room{
		ID:           12,
		Title:        "r3",
		HostID:       1,
		MaxPlayers:   8,
		PlayersCount: 1,
		Players:      map[uint32]*state.Player{1: host},
		PlayersTime:  map[uint32]time.Time{1: time.Now()},
		Started:      true,
		StartedAt:    time.Now(),
		Row:          []string{"12", "\x7f0018", "r3", "host", "For all", "1/8", "1ABCDEF"},
		Ctime:        time.Now(),
	}
	room.CtlSum = state.RoomControlSum(room.Row)
	c.Store.RoomsByID[room.ID] = room
	c.Store.RoomsByPID[1] = room
	c.Store.RoomsBySum[room.CtlSum] = room

	conn := &model.Connection{Data: map[string]any{"id": uint32(1)}}
	c.leaveRoom(conn)

	if _, ok := c.Store.RoomsByID[room.ID]; ok {
		t.Fatalf("expected started room to be deleted when last player leaves")
	}
}
