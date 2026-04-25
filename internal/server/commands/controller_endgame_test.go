package commands

import (
	"testing"

	"cossacksgameserver/golang/internal/server/model"
	"cossacksgameserver/golang/internal/server/state"
)

func TestParseEndgameSignedPlayerIDAndWinLabel(t *testing.T) {
	c := newControllerForJoinTests()
	c.Store.Players[42] = &state.Player{ID: 42, Nick: "p42"}
	c.Store.RoomsByID[10] = &state.Room{ID: 10, HostID: 7, Title: "room10"}
	conn := &model.Connection{Data: map[string]any{"id": uint32(7)}}

	ev, ok := c.parseEndgame(conn, []string{"game=10", "pid=42", "result=2"})
	if !ok {
		t.Fatalf("expected parse success")
	}
	if ev.GameID != 10 || ev.PlayerID != 42 {
		t.Fatalf("unexpected ids: %+v", ev)
	}
	if ev.Result != "win" {
		t.Fatalf("expected win label, got %q", ev.Result)
	}
	if ev.Nick != "p42" {
		t.Fatalf("expected nick p42, got %q", ev.Nick)
	}
	if ev.Own != "his " {
		t.Fatalf("expected host ownership marker, got %q", ev.Own)
	}
	if ev.Title != " room10" {
		t.Fatalf("expected room title suffix, got %q", ev.Title)
	}
}

func TestParseEndgameNegativePlayerIDReinterpretedUint32(t *testing.T) {
	c := newControllerForJoinTests()
	conn := &model.Connection{Data: map[string]any{}}

	ev, ok := c.parseEndgame(conn, []string{"1", "-1", "5"})
	if !ok {
		t.Fatalf("expected parse success")
	}
	if ev.PlayerID != ^uint32(0) {
		t.Fatalf("expected unsigned reinterpretation of -1, got %d", ev.PlayerID)
	}
	if ev.Result != "disconnect" {
		t.Fatalf("expected disconnect label, got %q", ev.Result)
	}
	if ev.Nick != "." {
		t.Fatalf("expected missing-player nick '.', got %q", ev.Nick)
	}
}

func TestParseEndgameUnknownResultLabel(t *testing.T) {
	c := newControllerForJoinTests()
	conn := &model.Connection{Data: map[string]any{}}

	ev, ok := c.parseEndgame(conn, []string{"1", "2", "9"})
	if !ok {
		t.Fatalf("expected parse success")
	}
	if ev.Result != "?9?" {
		t.Fatalf("expected unknown result token, got %q", ev.Result)
	}
}
