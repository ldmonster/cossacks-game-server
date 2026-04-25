package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"cossacksgameserver/golang/internal/protocol/gsc"
	"cossacksgameserver/golang/internal/server/model"
	"cossacksgameserver/golang/internal/server/state"
)

func TestResizeLargeAndSmallResponses(t *testing.T) {
	c := newControllerForJoinTests()
	req := &gsc.Stream{Ver: 2}
	conn := &model.Connection{Data: map[string]any{}}

	large := c.dispatchOpen(nil, conn, req, "resize", map[string]string{"height": "400"})
	if len(large) != 1 || !strings.Contains(large[0].Args[0], "#large") {
		t.Fatalf("expected large resize response, got %#v", large)
	}
	small := c.dispatchOpen(nil, conn, req, "resize", map[string]string{"height": "300"})
	if len(small) != 1 || strings.Contains(small[0].Args[0], "#large") {
		t.Fatalf("expected small resize response, got %#v", small)
	}
}

func TestNewRoomDialogAstateGuardAndSuccess(t *testing.T) {
	c := newControllerForJoinTests()
	req := &gsc.Stream{Ver: 2}
	conn := &model.Connection{Data: map[string]any{}}

	guard := c.dispatchOpen(nil, conn, req, "new_room_dgl", map[string]string{"ASTATE": "0"})
	if len(guard) != 1 || !strings.Contains(guard[0].Args[0], "You can not create or join room!") {
		t.Fatalf("expected ASTATE guard error, got %#v", guard)
	}
	ok := c.dispatchOpen(nil, conn, req, "new_room_dgl", map[string]string{"ASTATE": "1"})
	if len(ok) != 1 || !strings.Contains(ok[0].Args[0], "<NGDLG>") {
		t.Fatalf("expected new room dialog payload, got %#v", ok)
	}
}

func TestUserDetailsAcceptsEmbeddedNumericID(t *testing.T) {
	c := newControllerForJoinTests()
	req := &gsc.Stream{Ver: 2}
	conn := &model.Connection{Data: map[string]any{}}
	c.Store.Players[12] = &state.Player{ID: 12, Nick: "Nick12", ConnectedAt: time.Now().Add(-2 * time.Minute)}

	out := c.dispatchOpen(nil, conn, req, "user_details", map[string]string{"ID": "abc12xyz"})
	if len(out) != 1 || !strings.Contains(out[0].Args[0], "Nick12") {
		t.Fatalf("expected details for parsed numeric ID, got %#v", out)
	}
}

func TestUserDetailsIncludesAccountAndRoomActions(t *testing.T) {
	c := newControllerForJoinTests()
	req := &gsc.Stream{Ver: 2}
	conn := &model.Connection{Data: map[string]any{}}
	c.Store.Players[33] = &state.Player{
		ID:          33,
		Nick:        "Nick33",
		ConnectedAt: time.Now().Add(-time.Minute),
		Account: map[string]any{
			"type":    "LCN",
			"profile": "http://example.com/profile/33",
		},
	}
	room := makeRoom(c, 501, 33, "R501", "")
	c.Store.RoomsByPID[33] = room

	out := c.dispatchOpen(nil, conn, req, "user_details", map[string]string{"ID": "33"})
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
	conn := &model.Connection{Data: map[string]any{}}
	out := c.dispatchOpen(nil, conn, req, "rooms_table_dgl", map[string]string{})
	if len(out) != 1 || !strings.Contains(out[0].Args[0], "GW|open&new_room_dgl.dcml") {
		t.Fatalf("expected startup/rooms table payload, got %#v", out)
	}
}

func TestOpenStartupWithTrailingNULInURL(t *testing.T) {
	c := newControllerForJoinTests()
	req := &gsc.Stream{Ver: 2}
	conn := &model.Connection{Data: map[string]any{}}
	out := c.handleOpen(nil, conn, req, []string{"startup.dcml\x00"})
	if len(out) != 1 || !strings.Contains(out[0].Args[0], "GW|open&new_room_dgl.dcml") {
		t.Fatalf("expected startup payload for NUL-terminated URL, got %#v", out)
	}
}

func TestStartupWithoutGGCupFileHidesGGCupBanner(t *testing.T) {
	c := newControllerForJoinTests()
	// No gg_cup_file key — loadGGCup returns nil like Perl with no file path.
	req := &gsc.Stream{Ver: 2}
	conn := &model.Connection{Data: map[string]any{}}
	out := c.dispatchOpen(nil, conn, req, "startup", map[string]string{})
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
	c.Config.Raw["gg_cup_file"] = path
	req := &gsc.Stream{Ver: 2}
	conn := &model.Connection{Data: map[string]any{}}
	out := c.dispatchOpen(nil, conn, req, "startup", map[string]string{})
	if len(out) != 1 {
		t.Fatalf("expected one LW_show, got %#v", out)
	}
	body := out[0].Args[0]
	if !strings.Contains(body, "Cossacks GG Cup") || !strings.Contains(body, "http://goodgame.ru/cup/") {
		t.Fatalf("expected startup GG Cup block like Perl Open startup, got: %q", body)
	}
	// !gg_cup.wo_info sub-block: prize in RUB (share/cs/startup.cml)
	if !strings.Contains(body, "RUB") || !strings.Contains(body, "1000") {
		t.Fatalf("expected prize/players from gg_cup when wo_info is 0, got: %q", body)
	}
}
