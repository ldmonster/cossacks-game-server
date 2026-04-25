package commands

import (
	"strings"
	"testing"

	"cossacksgameserver/golang/internal/protocol/gsc"
	"cossacksgameserver/golang/internal/server/model"
	"cossacksgameserver/golang/internal/server/state"
)

func TestRegNewRoomRowShapeCsParity(t *testing.T) {
	c := newControllerForJoinTests()
	c.Config.HolePort = 34000
	c.Config.HoleInterval = 3000
	req := &gsc.Stream{Ver: 2}
	c.Store.Players[1] = &state.Player{ID: 1, Nick: "host"}
	conn := &model.Connection{
		IP:    "10.0.0.1",
		IntIP: 12345,
		Data:  map[string]any{"id": uint32(1), "nick": "host"},
	}

	out := c.regNewRoom(conn, req, map[string]string{
		"ASTATE":    "1",
		"VE_TITLE":  "Room",
		"VE_PASSWD": "secret",
		"VE_MAX_PL": "6",
		"VE_LEVEL":  "2",
	})
	if len(out) != 1 || out[0].Name != "LW_show" {
		t.Fatalf("expected one LW_show response, got %#v", out)
	}
	room := c.Store.RoomsByID[1]
	if room == nil {
		t.Fatalf("expected created room")
	}
	// Perl CS row: [id, lock, title, nick, level, players, ver, int_ip, 0HEX]
	if len(room.Row) != 9 {
		t.Fatalf("expected CS row len=9, got %d (%#v)", len(room.Row), room.Row)
	}
	if room.Row[1] != "#" || room.Row[4] != "Normal" || room.Row[5] != "1/8" || room.Row[7] != "12345" {
		t.Fatalf("unexpected room row shape: %#v", room.Row)
	}
	if !strings.HasPrefix(room.Row[8], "0") {
		t.Fatalf("expected perl-like hex anti-id suffix, got %q", room.Row[8])
	}
}

func TestRegNewRoomTitleTruncateThenTrimParity(t *testing.T) {
	c := newControllerForJoinTests()
	req := &gsc.Stream{Ver: 2}
	c.Store.Players[1] = &state.Player{ID: 1, Nick: "host"}
	conn := &model.Connection{
		IP:    "10.0.0.1",
		IntIP: 12345,
		Data:  map[string]any{"id": uint32(1), "nick": "host"},
	}

	// Perl order is substr first, then trim spaces.
	raw := strings.Repeat("A", 59) + " "
	_ = c.regNewRoom(conn, req, map[string]string{
		"ASTATE":   "1",
		"VE_TITLE": raw + "TRAIL",
	})
	room := c.Store.RoomsByID[1]
	if room == nil {
		t.Fatalf("expected room to be created")
	}
	if len(room.Title) != 59 {
		t.Fatalf("expected trailing space removed after truncation, got len=%d title=%q", len(room.Title), room.Title)
	}
}

func TestRegNewRoomGameIDPrefixFromVEType(t *testing.T) {
	c := newControllerForJoinTests()
	req := &gsc.Stream{Ver: 2}
	c.Store.Players[1] = &state.Player{ID: 1, Nick: "host"}
	conn := &model.Connection{
		IP:    "10.0.0.1",
		IntIP: 12345,
		Data:  map[string]any{"id": uint32(1), "nick": "host"},
	}
	out := c.regNewRoom(conn, req, map[string]string{
		"ASTATE":   "1",
		"VE_TITLE": "Room",
		"VE_TYPE":  "HB",
	})
	if len(out) != 1 || !strings.Contains(out[0].Args[0], "HB1") {
		t.Fatalf("expected HB-prefixed id in payload, got %#v", out)
	}
}

func TestRegNewRoomRowShapeAcAddsVETypeColumn(t *testing.T) {
	c := newControllerForJoinTests()
	c.Config.HolePort = 34000
	c.Config.HoleInterval = 3000
	req := &gsc.Stream{Ver: 8}
	c.Store.Players[1] = &state.Player{ID: 1, Nick: "host"}
	conn := &model.Connection{
		IP:    "10.0.0.1",
		IntIP: 12345,
		Data:  map[string]any{"id": uint32(1), "nick": "host"},
	}
	_ = c.regNewRoom(conn, req, map[string]string{
		"ASTATE":    "1",
		"VE_TITLE":  "ACRoom",
		"VE_MAX_PL": "6",
		"VE_LEVEL":  "1",
		"VE_TYPE":   "AmericanConquest",
	})
	room := c.Store.RoomsByID[1]
	if room == nil {
		t.Fatalf("expected created room")
	}
	// AC row: [id, lock, title, nick, VE_TYPE, level, players, ver, int_ip, 0HEX] (len 10)
	if len(room.Row) != 10 {
		t.Fatalf("expected AC row len=10, got %d (%#v)", len(room.Row), room.Row)
	}
	if room.Row[4] != "AmericanConquest" {
		t.Fatalf("expected VE_TYPE in row index 4, got %#v", room.Row)
	}
	if room.Row[5] != "Easy" {
		t.Fatalf("expected level label at index 5, got %#v", room.Row)
	}
	if room.Ver != 8 {
		t.Fatalf("expected room.Ver from request, got %d", room.Ver)
	}
}
