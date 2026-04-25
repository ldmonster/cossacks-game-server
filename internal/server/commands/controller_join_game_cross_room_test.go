package commands

import (
	"testing"

	"cossacksgameserver/golang/internal/protocol/gsc"
	"cossacksgameserver/golang/internal/server/model"
	"cossacksgameserver/golang/internal/server/state"
)

func TestJoinGameLeavesPreviousRoomBeforeJoiningAnother(t *testing.T) {
	c := newControllerForJoinTests()
	c.Store.Players[1] = &state.Player{ID: 1, Nick: "host1"}
	c.Store.Players[2] = &state.Player{ID: 2, Nick: "guest"}
	c.Store.Players[3] = &state.Player{ID: 3, Nick: "host2"}
	roomA := makeRoom(c, 1, 1, "room-a", "")
	roomB := makeRoom(c, 2, 3, "room-b", "")
	roomA.Players[2] = c.Store.Players[2]
	roomA.PlayersTime[2] = roomA.PlayersTime[1]
	roomA.PlayersCount = 2
	roomA.Row = setRoomPlayersColumn(roomA.Row, 2, roomA.MaxPlayers)
	roomA.CtlSum = state.RoomControlSum(roomA.Row)
	c.Store.RoomsBySum[roomA.CtlSum] = roomA
	c.Store.RoomsByPID[2] = roomA

	conn := &model.Connection{Data: map[string]any{"id": uint32(2), "nick": "guest"}}
	req := &gsc.Stream{Ver: 2}
	out := c.joinGame(nil, conn, req, map[string]string{"VE_RID": "2", "ASTATE": "1"})
	if len(out) != 1 {
		t.Fatalf("expected join response, got %#v", out)
	}
	if c.Store.RoomsByPID[2] != roomB {
		t.Fatalf("expected guest in room B, got %#v", c.Store.RoomsByPID[2])
	}
	if c.Store.RoomsByPID[1] != roomA {
		t.Fatalf("expected host still mapped to room A")
	}
	if roomA.PlayersCount != 1 {
		t.Fatalf("expected room A count back to 1 after guest left, got %d", roomA.PlayersCount)
	}
	if _, ok := roomA.Players[2]; ok {
		t.Fatalf("expected guest removed from room A players map")
	}
}
