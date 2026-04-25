package commands

import (
	"testing"
	"time"

	"cossacksgameserver/golang/internal/server/model"
	"cossacksgameserver/golang/internal/server/state"
)

func TestAliveTimerRemovesPlayerFromRoom(t *testing.T) {
	c := newControllerForJoinTests()
	c.aliveTTL = 20 * time.Millisecond
	c.ensureRuntimeMaps()

	c.Store.Players[1] = &state.Player{ID: 1, Nick: "p1", ConnectedAt: time.Now()}
	room := makeRoom(c, 201, 1, "timer-room", "")
	conn := &model.Connection{Data: map[string]any{"id": uint32(1), "nick": "p1"}}
	c.playerConns[1] = conn
	c.Store.RoomsByPID[1] = room

	c.armAliveTimer(1)
	time.Sleep(80 * time.Millisecond)

	c.stateMu.Lock()
	defer c.stateMu.Unlock()
	if _, ok := c.Store.RoomsByPID[1]; ok {
		t.Fatalf("expected player to be removed from room by not_alive timer")
	}
	if conn.Closed {
		t.Fatalf("not_alive should leave room without force-closing socket")
	}
}

func TestOnDisconnectRemovesPlayerAndLeavesHostRoom(t *testing.T) {
	c := newControllerForJoinTests()
	c.ensureRuntimeMaps()

	c.Store.Players[1] = &state.Player{ID: 1, Nick: "host", ConnectedAt: time.Now()}
	_ = makeRoom(c, 301, 1, "dc-room", "")
	conn := &model.Connection{Data: map[string]any{"id": uint32(1), "nick": "host"}}
	c.playerConns[1] = conn

	c.OnDisconnect(conn)

	if _, ok := c.Store.Players[1]; ok {
		t.Fatalf("expected player removed from store")
	}
	if _, ok := c.Store.RoomsByPID[1]; ok {
		t.Fatalf("expected host removed from room mapping")
	}
	if _, ok := c.Store.RoomsByID[301]; ok {
		t.Fatalf("expected room deleted when host disconnects (pre-start)")
	}
	if _, ok := c.playerConns[1]; ok {
		t.Fatalf("expected playerConns cleared")
	}
}

func TestLeaveClearsAliveTimer(t *testing.T) {
	c := newControllerForJoinTests()
	c.aliveTTL = 200 * time.Millisecond
	c.ensureRuntimeMaps()

	c.Store.Players[1] = &state.Player{ID: 1, Nick: "p1", ConnectedAt: time.Now()}
	room := makeRoom(c, 202, 1, "leave-room", "")
	conn := &model.Connection{Data: map[string]any{"id": uint32(1), "nick": "p1"}}
	c.playerConns[1] = conn
	c.Store.RoomsByPID[1] = room
	c.armAliveTimer(1)

	c.leaveRoom(conn)
	time.Sleep(260 * time.Millisecond)

	c.stateMu.Lock()
	defer c.stateMu.Unlock()
	if conn.Closed {
		t.Fatalf("expected timer to be cleared on leave; connection should remain open")
	}
}
