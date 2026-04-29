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
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ldmonster/cossacks-game-server/internal/domain/player"
	"github.com/ldmonster/cossacks-game-server/internal/domain/session"
	tconn "github.com/ldmonster/cossacks-game-server/internal/transport/tconn"
	"github.com/ldmonster/cossacks-game-server/internal/transport/gsc"
)

func TestResizeLargeAndSmallResponses(t *testing.T) {
	c := newControllerForJoinTests()
	req := &gsc.Stream{Ver: 2}
	conn := &tconn.Connection{Session: &session.Session{}}

	large, _ := c.dispatchOpen(nil, conn, req, "resize", map[string]string{"height": "400"})
	if len(large) != 1 || !strings.Contains(large[0].Args[0], "#large") {
		t.Fatalf("expected large resize response, got %#v", large)
	}
	small, _ := c.dispatchOpen(nil, conn, req, "resize", map[string]string{"height": "300"})
	if len(small) != 1 || strings.Contains(small[0].Args[0], "#large") {
		t.Fatalf("expected small resize response, got %#v", small)
	}
}

func TestNewRoomDialogAstateGuardAndSuccess(t *testing.T) {
	c := newControllerForJoinTests()
	req := &gsc.Stream{Ver: 2}
	conn := &tconn.Connection{Session: &session.Session{}}

	guard, _ := c.dispatchOpen(nil, conn, req, "new_room_dgl", map[string]string{"ASTATE": "0"})
	if len(guard) != 1 || !strings.Contains(guard[0].Args[0], "You can not create or join room!") {
		t.Fatalf("expected ASTATE guard error, got %#v", guard)
	}
	ok, _ := c.dispatchOpen(nil, conn, req, "new_room_dgl", map[string]string{"ASTATE": "1"})
	if len(ok) != 1 || !strings.Contains(ok[0].Args[0], "<NGDLG>") {
		t.Fatalf("expected new room dialog payload, got %#v", ok)
	}
}

func TestUserDetailsAcceptsEmbeddedNumericID(t *testing.T) {
	c := newControllerForJoinTests()
	req := &gsc.Stream{Ver: 2}
	conn := &tconn.Connection{Session: &session.Session{}}
	c.Store.SetPlayer(&player.Player{ID: 12, Nick: "Nick12", ConnectedAt: time.Now().Add(-2 * time.Minute)})

	out, _ := c.dispatchOpen(nil, conn, req, "user_details", map[string]string{"ID": "abc12xyz"})
	if len(out) != 1 || !strings.Contains(out[0].Args[0], "Nick12") {
		t.Fatalf("expected details for parsed numeric ID, got %#v", out)
	}
}

func TestUserDetailsIncludesAccountAndRoomActions(t *testing.T) {
	c := newControllerForJoinTests()
	req := &gsc.Stream{Ver: 2}
	conn := &tconn.Connection{Session: &session.Session{}}
	c.Store.SetPlayer(&player.Player{
		ID:             33,
		Nick:           "Nick33",
		ConnectedAt:    time.Now().Add(-time.Minute),
		AccountType:    "LCN",
		AccountProfile: "http://example.com/profile/33",
	})
	room := makeRoom(c, 501, 33, "R501", "")
	c.Store.IndexRoomByHost(33, room)

	out, _ := c.dispatchOpen(nil, conn, req, "user_details", map[string]string{"ID": "33"})
	if len(out) != 1 {
		t.Fatalf("expected one user_details response, got %#v", out)
	}
	body := out[0].Args[0]
	if !strings.Contains(body, "Logon with") || !strings.Contains(body, "http://example.com/profile/33") {
		t.Fatalf("expected account block in user_details body: %q", body)
	}
	if !strings.Contains(body, "join_game.dcml") || !strings.Contains(body, "room_info_dgl.dcml") {
		t.Fatalf("expected room join/info actions in user_details body: %q", body)
	}
}

func TestRoomsTableDglRoutesToStartup(t *testing.T) {
	c := newControllerForJoinTests()
	req := &gsc.Stream{Ver: 2}
	conn := &tconn.Connection{Session: &session.Session{}}
	out, _ := c.dispatchOpen(nil, conn, req, "rooms_table_dgl", map[string]string{})
	if len(out) != 1 || !strings.Contains(out[0].Args[0], "GW|open&new_room_dgl.dcml") {
		t.Fatalf("expected startup/rooms table payload, got %#v", out)
	}
}

func TestOpenStartupWithTrailingNULInURL(t *testing.T) {
	c := newControllerForJoinTests()
	req := &gsc.Stream{Ver: 2}
	conn := &tconn.Connection{Session: &session.Session{}}
	out, err := c.handleOpen(nil, conn, req, []string{"startup.dcml\x00"})
	if err != nil {
		t.Fatalf("handleOpen returned error: %v", err)
	}
	if len(out) != 1 || !strings.Contains(out[0].Args[0], "GW|open&new_room_dgl.dcml") {
		t.Fatalf("expected startup payload for NUL-terminated URL, got %#v", out)
	}
}

func TestStartupWithoutGGCupFileHidesGGCupBanner(t *testing.T) {
	c := newControllerForJoinTests()
	// No gg_cup_file key — loadGGCup returns nil like reference with no file path.
	req := &gsc.Stream{Ver: 2}
	conn := &tconn.Connection{Session: &session.Session{}}
	out, _ := c.dispatchOpen(nil, conn, req, "startup", map[string]string{})
	if len(out) != 1 {
		t.Fatalf("expected one LW_show, got %#v", out)
	}
	if strings.Contains(out[0].Args[0], "Cossacks GG Cup") {
		t.Fatalf("expected no GG Cup marketing block without gg_cup, body: %q", out[0].Args[0])
	}
	if !strings.Contains(out[0].Args[0], "#box[%BOTTOM](x:0,w:100%,y:100%-32,h:32)") {
		t.Fatalf("expected resolved startup bottom panel geometry, got: %q", out[0].Args[0])
	}
}

func TestStartupWithGGCupJSONShowsRegistrationBlock(t *testing.T) {
	c := newControllerForJoinTests()
	dir := t.TempDir()
	path := filepath.Join(dir, "gg_cup.json")
	if err := os.WriteFile(path, []byte(`{
		"id": 42,
		"started": 0,
		"wo_info": 0,
		"players_count": "3",
		"prize_fund": 1000.0
	}`), 0o644); err != nil {
		t.Fatal(err)
	}
	c.Game.GGCupFile = path
	req := &gsc.Stream{Ver: 2}
	conn := &tconn.Connection{Session: &session.Session{}}
	out, _ := c.dispatchOpen(nil, conn, req, "startup", map[string]string{})
	if len(out) != 1 {
		t.Fatalf("expected one LW_show, got %#v", out)
	}
	body := out[0].Args[0]
	if !strings.Contains(body, "Cossacks GG Cup") || !strings.Contains(body, "http://goodgame.ru/cup/") {
		t.Fatalf("expected startup GG Cup block like reference Open startup, got: %q", body)
	}
	// !gg_cup.wo_info sub-block: prize in RUB (share/cs/startup.cml)
	if !strings.Contains(body, "RUB") || !strings.Contains(body, "1000") {
		t.Fatalf("expected prize/players from gg_cup when wo_info is 0, got: %q", body)
	}
}
