package commands

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"cossacksgameserver/golang/internal/protocol/gsc"
	"cossacksgameserver/golang/internal/server/model"
	"cossacksgameserver/golang/internal/server/state"
)

// lwGoldenMeta is a stable, diff-friendly view of []gsc.Command: names and per-arg byte
// length (CML and binary). Full bodies are *not* snapshotted (template churn).
// Regenerate JSON under testdata/golden: go test ./internal/server/commands -golden
type lwGoldenMeta struct {
	Name    string `json:"name"`
	ArgLens []int  `json:"arg_lens"`
}

func commandsToGoldenMeta(cmds []gsc.Command) []lwGoldenMeta {
	out := make([]lwGoldenMeta, 0, len(cmds))
	for _, c := range cmds {
		ln := make([]int, len(c.Args))
		for i, a := range c.Args {
			ln[i] = len(a)
		}
		out = append(out, lwGoldenMeta{Name: c.Name, ArgLens: ln})
	}
	return out
}

func testGoldenJSON(t *testing.T, fileName string, got []gsc.Command) {
	t.Helper()
	dir := "testdata/golden"
	path := filepath.Join(dir, fileName)
	meta := commandsToGoldenMeta(got)
	gotB, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		t.Fatalf("marshal meta: %v", err)
	}
	if *golden {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(path, append(gotB, '\n'), 0o644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
		t.Logf("wrote %s", path)
		return
	}
	wantB, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v (re-run with: go test ./internal/server/commands -golden)", path, err)
	}
	want := string(wantB)
	gotS := string(gotB)
	for want != "" && want[len(want)-1] == '\n' {
		want = want[:len(want)-1]
	}
	if gotS != want {
		t.Fatalf("golden %s mismatch.\ngot:\n%s\nwant:\n%s", fileName, gotB, wantB)
	}
}

func makeStableRoom(t *testing.T, c *Controller, id, hostID uint32, title string) *state.Room {
	t.Helper()
	c.Store.Players[hostID] = &state.Player{ID: hostID, Nick: "host", ConnectedAt: time.Unix(1_700_000_000, 0).UTC()}
	r := &state.Room{
		ID:           id,
		Title:        title,
		HostID:       hostID,
		HostAddr:     "1.2.3.4",
		HostAddrInt:  0,
		Ver:          2,
		Password:     "",
		MaxPlayers:   8,
		PlayersCount: 1,
		Players:      map[uint32]*state.Player{hostID: c.Store.Players[hostID]},
		PlayersTime:  map[uint32]time.Time{hostID: time.Unix(1_700_000_000, 0).UTC()},
		Row:          []string{"1", "", title, "host", "For all", "1/8", "2"},
		Ctime:        time.Unix(1_700_000_000, 0).UTC(),
	}
	r.CtlSum = state.RoomControlSum(r.Row)
	c.Store.RoomsByID[id] = r
	c.Store.RoomsByPID[hostID] = r
	c.Store.RoomsBySum[r.CtlSum] = r
	return r
}

func TestGoldenOpenEnter(t *testing.T) {
	c := newControllerForJoinTests()
	ctx := context.Background()
	req2 := &gsc.Stream{Ver: 2}
	req8 := &gsc.Stream{Ver: 8}
	conn := &model.Connection{Data: map[string]any{}}
	testGoldenJSON(t, "open_enter_v2.json", c.HandleWithMeta(ctx, conn, req2, "open", []string{"enter.dcml"}, "w", "k").Commands)
	testGoldenJSON(t, "open_enter_v8.json", c.HandleWithMeta(ctx, conn, req8, "open", []string{"enter.dcml"}, "w", "k").Commands)
}

func TestGoldenOpenStartup(t *testing.T) {
	c := newControllerForJoinTests()
	ctx := context.Background()
	req2 := &gsc.Stream{Ver: 2}
	req8 := &gsc.Stream{Ver: 8}
	conn := &model.Connection{Data: map[string]any{}}
	testGoldenJSON(t, "open_startup_v2.json", c.HandleWithMeta(ctx, conn, req2, "go", []string{"startup"}, "w", "k").Commands)
	testGoldenJSON(t, "open_startup_v8.json", c.HandleWithMeta(ctx, conn, req8, "go", []string{"startup"}, "w", "k").Commands)
}

func TestGoldenOpenNewRoomDialog(t *testing.T) {
	c := cForGolden()
	conn := &model.Connection{Data: map[string]any{}}
	testGoldenJSON(t, "open_new_room_dgl_v2.json", c.dispatchOpen(context.Background(), conn, &gsc.Stream{Ver: 2}, "new_room_dgl", map[string]string{"ASTATE": "1"}))
	testGoldenJSON(t, "open_new_room_dgl_v8.json", c.dispatchOpen(context.Background(), conn, &gsc.Stream{Ver: 8}, "new_room_dgl", map[string]string{"ASTATE": "1"}))
}

func TestGoldenGETTBL(t *testing.T) {
	unknown := uint32(0xFFFF_FFFE)
	unknownB := make([]byte, 4)
	binary.LittleEndian.PutUint32(unknownB, unknown)
	name2 := "ROOMS_V2\x00"
	t.Run("v2", func(t *testing.T) {
		c := cForGolden()
		makeStableRoom(t, c, 3, 1, "G")
		out := c.handleGETTBL(&model.Connection{Data: map[string]any{}}, &gsc.Stream{Ver: 2},
			[]string{name2, "0", string(unknownB)})
		testGoldenJSON(t, "gettbl_one_row_v2.json", out)
	})
	t.Run("v8", func(t *testing.T) {
		c := cForGolden()
		r := makeStableRoom(t, c, 3, 1, "G")
		r.Ver = 8
		// re-run checksum if Ver affects row; RoomControlSum is from Row only, ok.
		name8 := "ROOMS_V8\x00"
		out := c.handleGETTBL(&model.Connection{Data: map[string]any{}}, &gsc.Stream{Ver: 8},
			[]string{name8, "0", string(unknownB)})
		testGoldenJSON(t, "gettbl_one_row_v8.json", out)
	})
}

func cForGolden() *Controller {
	c := newControllerForJoinTests()
	c.Config.ShowStartedRooms = true
	return c
}

// TestGoldenRegNewRoomThenGettbl covers create-room → GETTBL (Phase 0 golden checklist)
// as command metadata across ver=2 (cs) and ver=8 (ac).
func TestGoldenRegNewRoomThenGettbl(t *testing.T) {
	unknown := uint32(0xEEEE_EEEE)
	unknownB := make([]byte, 4)
	binary.LittleEndian.PutUint32(unknownB, unknown)

	t.Run("v2", func(t *testing.T) {
		c := cForGolden()
		c.Config.HolePort = 34000
		c.Config.HoleInterval = 3000
		c.Store.LastRoomID = 0
		c.Store.Players[1] = &state.Player{ID: 1, Nick: "host"}
		conn := &model.Connection{IP: "192.0.2.1", IntIP: 111, Data: map[string]any{"id": uint32(1), "nick": "host"}}
		req := &gsc.Stream{Ver: 2}
		reg := c.regNewRoom(conn, req, map[string]string{
			"ASTATE": "1", "VE_TITLE": "GoldenRoom", "VE_MAX_PL": "6", "VE_LEVEL": "0",
		})
		get := c.handleGETTBL(&model.Connection{Data: map[string]any{}}, req,
			[]string{"ROOMS_V2\x00", "0", string(unknownB)})
		testGoldenJSON(t, "reg_new_room_then_gettbl_v2.json", append(reg, get...))
	})
	t.Run("v8", func(t *testing.T) {
		c := cForGolden()
		c.Config.HolePort = 34000
		c.Config.HoleInterval = 3000
		c.Store.LastRoomID = 0
		c.Store.Players[1] = &state.Player{ID: 1, Nick: "host"}
		conn := &model.Connection{IP: "192.0.2.2", IntIP: 222, Data: map[string]any{"id": uint32(1), "nick": "host"}}
		req := &gsc.Stream{Ver: 8}
		reg := c.regNewRoom(conn, req, map[string]string{
			"ASTATE": "1", "VE_TITLE": "GoldenAC", "VE_MAX_PL": "6", "VE_LEVEL": "0",
			"VE_TYPE": "AC",
		})
		get := c.handleGETTBL(&model.Connection{Data: map[string]any{}}, req,
			[]string{"ROOMS_V8\x00", "0", string(unknownB)})
		testGoldenJSON(t, "reg_new_room_then_gettbl_v8.json", append(reg, get...))
	})
}

func TestGoldenJoinRoom(t *testing.T) {
	c := cForGolden()
	c.Store.Players[1] = &state.Player{ID: 1, Nick: "h"}
	c.Store.Players[2] = &state.Player{ID: 2, Nick: "g"}
	makeStableRoom(t, c, 1, 1, "room")
	req := &gsc.Stream{Ver: 2}
	conn := &model.Connection{Data: map[string]any{"id": uint32(2), "nick": "g"}}
	got := c.joinGame(context.Background(), conn, req, map[string]string{"VE_RID": "1", "ASTATE": "1"})
	testGoldenJSON(t, "join_game_success_v2.json", got)
}

// TestGoldenJoinPlCmdRoomInfo covers join_pl_cmd → room_info_dgl (Perl delegation).
func TestGoldenJoinPlCmdRoomInfo(t *testing.T) {
	c := cForGolden()
	c.Config.Raw["show_started_room_info"] = "0"
	c.Store.Players[1] = &state.Player{ID: 1, Nick: "h", ConnectedAt: time.Unix(1_700_000_000, 0).UTC()}
	makeStableRoom(t, c, 42, 1, "PlJoinRoom")
	conn := &model.Connection{Data: map[string]any{"id": uint32(9), "nick": "obs"}}
	out2 := c.dispatchOpen(nil, conn, &gsc.Stream{Ver: 2}, "join_pl_cmd", map[string]string{"VE_PLAYER": "1"})
	testGoldenJSON(t, "join_pl_cmd_room_info_v2.json", out2)

	c8 := cForGolden()
	c8.Config.Raw["show_started_room_info"] = "0"
	c8.Store.Players[1] = &state.Player{ID: 1, Nick: "h", ConnectedAt: time.Unix(1_700_000_000, 0).UTC()}
	r := makeStableRoom(t, c8, 43, 1, "PlJoinAC")
	r.Ver = 8
	conn8 := &model.Connection{Data: map[string]any{"id": uint32(9), "nick": "obs"}}
	out8 := c.dispatchOpen(nil, conn8, &gsc.Stream{Ver: 8}, "join_pl_cmd", map[string]string{"VE_PLAYER": "1"})
	testGoldenJSON(t, "join_pl_cmd_room_info_v8.json", out8)
}

func TestGoldenStartedRoomInfoAndStatcols(t *testing.T) {
	c := cForGolden()
	c.Config.Raw["show_started_room_info"] = "1"
	host := &state.Player{ID: 1, Nick: "host", ConnectedAt: time.Unix(1_700_000_000, 0).UTC()}
	c.Store.Players[1] = host
	r := makeStableRoom(t, c, 70, 1, "StartedRoom")
	r.Started = true
	r.StartedAt = time.Now().UTC()
	r.Ver = 2
	r.StartedUsers = []*state.Player{host}
	conn := &model.Connection{Data: map[string]any{"id": uint32(2), "nick": "obs"}}
	main2 := c.roomInfo(conn, &gsc.Stream{Ver: 2}, map[string]string{"VE_RID": "70"})
	stat2 := c.roomInfo(conn, &gsc.Stream{Ver: 2}, map[string]string{"VE_RID": "70", "part": "statcols", "page": "1", "res": "0"})
	testGoldenJSON(t, "room_info_started_v2.json", main2)
	testGoldenJSON(t, "room_info_started_statcols_v2.json", stat2)

	c8 := cForGolden()
	c8.Config.Raw["show_started_room_info"] = "1"
	host8 := &state.Player{ID: 1, Nick: "host", ConnectedAt: time.Unix(1_700_000_000, 0).UTC()}
	c8.Store.Players[1] = host8
	r8 := makeStableRoom(t, c8, 71, 1, "StartedRoomAC")
	r8.Started = true
	r8.StartedAt = time.Now().UTC()
	r8.Ver = 8
	r8.StartedUsers = []*state.Player{host8}
	conn8 := &model.Connection{Data: map[string]any{"id": uint32(2), "nick": "obs"}}
	main8 := c8.roomInfo(conn8, &gsc.Stream{Ver: 8}, map[string]string{"VE_RID": "71"})
	stat8 := c8.roomInfo(conn8, &gsc.Stream{Ver: 8}, map[string]string{"VE_RID": "71", "part": "statcols", "page": "1", "res": "0"})
	testGoldenJSON(t, "room_info_started_v8.json", main8)
	testGoldenJSON(t, "room_info_started_statcols_v8.json", stat8)
}

func TestGoldenUserDetails(t *testing.T) {
	c := cForGolden()
	c.Store.Players[10] = &state.Player{
		ID:          10,
		Nick:        "User10",
		ConnectedAt: time.Unix(1_700_000_000, 0).UTC(),
		Account: map[string]any{
			"type":    "LCN",
			"profile": "http://example/profile/10",
		},
	}
	conn := &model.Connection{Data: map[string]any{"id": uint32(99), "nick": "viewer"}}
	v2 := c.dispatchOpen(context.Background(), conn, &gsc.Stream{Ver: 2}, "user_details", map[string]string{"ID": "10"})
	v8 := c.dispatchOpen(context.Background(), conn, &gsc.Stream{Ver: 8}, "user_details", map[string]string{"ID": "10"})
	testGoldenJSON(t, "open_user_details_v2.json", v2)
	testGoldenJSON(t, "open_user_details_v8.json", v8)
}

func TestGoldenNoResponseAlive(t *testing.T) {
	c := cForGolden()
	c.ensureRuntimeMaps()
	c.Store.Players[1] = &state.Player{ID: 1, Nick: "p", ConnectedAt: time.Unix(1_700_000_000, 0).UTC()}
	conn := &model.Connection{Data: map[string]any{"id": uint32(1), "nick": "p"}}
	out := c.Handle(context.Background(), conn, &gsc.Stream{Ver: 2}, "alive", []string{}, "w", "k")
	if out != nil {
		t.Fatalf("alive: expected nil from Handle, got %v", out)
	}
	testGoldenJSON(t, "noresponse_alive.json", nil)
}

func TestGoldenNoResponseStartLeave(t *testing.T) {
	// "leave" and "start" (host) from Handle return nil: HasResponse false.
	c := cForGolden()
	c.Store.Players[1] = &state.Player{ID: 1, Nick: "h"}
	room := makeStableRoom(t, c, 9, 1, "R")
	_ = room
	conn := &model.Connection{Data: map[string]any{"id": uint32(1), "nick": "h"}}
	leaveOut := c.Handle(context.Background(), conn, &gsc.Stream{Ver: 2}, "leave", nil, "w", "k")
	if leaveOut != nil {
		t.Fatalf("leave: expected nil from Handle, got %v", leaveOut)
	}
	// re-setup room: leave removed host in non-started path — make room again
	c2 := cForGolden()
	c2.Store.Players[1] = &state.Player{ID: 1, Nick: "h"}
	_ = makeStableRoom(t, c2, 9, 1, "R")
	conn2 := &model.Connection{Data: map[string]any{"id": uint32(1), "nick": "h"}}
	stOut := c2.Handle(context.Background(), conn2, &gsc.Stream{Ver: 2}, "start", []string{
		"sav:[1]", "m.m3d", "0",
	}, "w", "k")
	if stOut != nil {
		t.Fatalf("start: expected nil from Handle, got %v", stOut)
	}
	// empty golden files for "documented" contract of no-outbound
	testGoldenJSON(t, "noresponse_leave.json", nil)
	testGoldenJSON(t, "noresponse_start.json", nil)
}

// TestGoldenMetaRoundTrip is a small sanity check for the snapshot shape.
func TestGoldenMetaRoundTrip(t *testing.T) {
	t.Parallel()
	b, err := json.Marshal(commandsToGoldenMeta([]gsc.Command{
		{Name: "LW_dtbl", Args: []string{"N\x00", string([]byte{1, 2, 3, 4})}},
	}))
	if err != nil {
		t.Fatal(err)
	}
	if string(b) == "" {
		t.Fatal("empty")
	}
}

func TestGoldenFileNames(t *testing.T) {
	// If this test is missing files, gettbl or enter will fail with read error first.
	dir := "testdata/golden"
	want := []string{
		"open_enter_v2.json",
		"open_enter_v8.json",
		"open_startup_v2.json",
		"open_startup_v8.json",
		"open_new_room_dgl_v2.json",
		"open_new_room_dgl_v8.json",
		"open_user_details_v2.json",
		"open_user_details_v8.json",
		"gettbl_one_row_v2.json",
		"gettbl_one_row_v8.json",
		"reg_new_room_then_gettbl_v2.json",
		"reg_new_room_then_gettbl_v8.json",
		"join_pl_cmd_room_info_v2.json",
		"join_pl_cmd_room_info_v8.json",
		"room_info_started_v2.json",
		"room_info_started_v8.json",
		"room_info_started_statcols_v2.json",
		"room_info_started_statcols_v8.json",
		"join_game_success_v2.json",
		"noresponse_alive.json",
		"noresponse_leave.json",
		"noresponse_start.json",
	}
	seen := map[string]bool{}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("list %s: %v", dir, err)
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		seen[e.Name()] = true
	}
	for _, w := range want {
		if !seen[w] {
			t.Errorf("expected golden file %s under %s", w, dir)
		}
	}
}
