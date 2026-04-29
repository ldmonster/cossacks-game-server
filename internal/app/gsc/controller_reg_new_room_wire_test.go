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
	"strings"
	"testing"

	"github.com/ldmonster/cossacks-game-server/internal/domain/player"
	"github.com/ldmonster/cossacks-game-server/internal/domain/session"
	tconn "github.com/ldmonster/cossacks-game-server/internal/transport/tconn"
	"github.com/ldmonster/cossacks-game-server/internal/transport/gsc"
)

func TestRegNewRoomRowShapeCs(t *testing.T) {
	c := newControllerForJoinTests()
	c.Server.HolePort = 34000
	c.Game.HoleInterval = 3000
	req := &gsc.Stream{Ver: 2}
	c.Store.SetPlayer(&player.Player{ID: 1, Nick: "host"})
	conn := &tconn.Connection{
		IP:      "10.0.0.1",
		IntIP:   12345,
		Session: &session.Session{PlayerID: 1, Nick: "host"},
	}

	out, _ := c.dispatchOpen(nil, conn, req, "reg_new_room", map[string]string{
		"ASTATE":    "1",
		"VE_TITLE":  "Room",
		"VE_PASSWD": "secret",
		"VE_MAX_PL": "6",
		"VE_LEVEL":  "2",
	})
	if len(out) != 1 || out[0].Name != "LW_show" {
		t.Fatalf("expected one LW_show response, got %#v", out)
	}
	room := c.Store.GetRoom(1)
	if room == nil {
		t.Fatalf("expected created room")
	}
	// reference CS row: [id, lock, title, nick, level, players, ver, int_ip, 0HEX]
	if len(room.Row) != 9 {
		t.Fatalf("expected CS row len=9, got %d (%#v)", len(room.Row), room.Row)
	}
	if room.Row[1] != "#" || room.Row[4] != "Normal" || room.Row[5] != "1/8" || room.Row[7] != "12345" {
		t.Fatalf("unexpected room row shape: %#v", room.Row)
	}
	if !strings.HasPrefix(room.Row[8], "0") {
			t.Fatalf("expected hex anti-id suffix, got %q", room.Row[8])
	}
}

func TestRegNewRoomTitleTruncateThenTrim(t *testing.T) {
	c := newControllerForJoinTests()
	req := &gsc.Stream{Ver: 2}
	c.Store.SetPlayer(&player.Player{ID: 1, Nick: "host"})
	conn := &tconn.Connection{
		IP:      "10.0.0.1",
		IntIP:   12345,
		Session: &session.Session{PlayerID: 1, Nick: "host"},
	}

	// reference order is substr first, then trim spaces.
	raw := strings.Repeat("A", 59) + " "
	_, _ = c.dispatchOpen(nil, conn, req, "reg_new_room", map[string]string{
		"ASTATE":   "1",
		"VE_TITLE": raw + "TRAIL",
	})
	room := c.Store.GetRoom(1)
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
	c.Store.SetPlayer(&player.Player{ID: 1, Nick: "host"})
	conn := &tconn.Connection{
		IP:      "10.0.0.1",
		IntIP:   12345,
		Session: &session.Session{PlayerID: 1, Nick: "host"},
	}
	out, _ := c.dispatchOpen(nil, conn, req, "reg_new_room", map[string]string{
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
	c.Server.HolePort = 34000
	c.Game.HoleInterval = 3000
	req := &gsc.Stream{Ver: 8}
	c.Store.SetPlayer(&player.Player{ID: 1, Nick: "host"})
	conn := &tconn.Connection{
		IP:      "10.0.0.1",
		IntIP:   12345,
		Session: &session.Session{PlayerID: 1, Nick: "host"},
	}
	_, _ = c.dispatchOpen(nil, conn, req, "reg_new_room", map[string]string{
		"ASTATE":    "1",
		"VE_TITLE":  "ACRoom",
		"VE_MAX_PL": "6",
		"VE_LEVEL":  "1",
		"VE_TYPE":   "AmericanConquest",
	})
	room := c.Store.GetRoom(1)
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
