package commands

import (
	"strings"
	"testing"
	"time"

	"cossacksgameserver/golang/internal/protocol/gsc"
	"cossacksgameserver/golang/internal/server/model"
	"cossacksgameserver/golang/internal/server/state"
)

func TestJoinPlayerAlreadyInRoomReturnsEmpty(t *testing.T) {
	c := newControllerForJoinTests()
	c.Store.Players[1] = &state.Player{ID: 1, Nick: "host"}
	c.Store.Players[2] = &state.Player{ID: 2, Nick: "guest"}
	_ = makeRoom(c, 700, 1, "room700", "")
	_ = makeRoom(c, 701, 2, "room701", "")

	// Caller already participates in a room: Perl push_empty behavior.
	conn := &model.Connection{Data: map[string]any{"id": uint32(2), "nick": "guest"}}
	out := c.joinPlayer(conn, map[string]string{"VE_PLAYER": "1"})
	if out == nil || len(out) != 0 {
		t.Fatalf("expected empty command set for already-in-room player, got %#v", out)
	}
}

func TestJoinPlayerNoRoomReturnsNil(t *testing.T) {
	c := newControllerForJoinTests()
	conn := &model.Connection{Data: map[string]any{"id": uint32(10), "nick": "p10"}}
	out := c.joinPlayer(conn, map[string]string{"VE_PLAYER": "999"})
	if out != nil {
		t.Fatalf("expected nil when target player room not found, got %#v", out)
	}
}

func TestJoinPlayerStartedRoomReturnsError(t *testing.T) {
	c := newControllerForJoinTests()
	c.Store.Players[1] = &state.Player{ID: 1, Nick: "host"}
	c.Store.Players[2] = &state.Player{ID: 2, Nick: "viewer"}
	room := makeRoom(c, 702, 1, "started-room", "")
	room.Started = true
	c.Store.RoomsByPID[1] = room

	conn := &model.Connection{Data: map[string]any{"id": uint32(2), "nick": "viewer"}}
	out := c.joinPlayer(conn, map[string]string{"VE_PLAYER": "1"})
	if len(out) != 1 || !strings.Contains(out[0].Args[0], "Game alredy started") {
		t.Fatalf("expected started-room error, got %#v", out)
	}
}

func TestJoinPlCmdInvalidVEPlayerYieldsNoCommands(t *testing.T) {
	c := newControllerForJoinTests()
	c.Store.Players[1] = &state.Player{ID: 1, Nick: "host"}
	_ = makeRoom(c, 703, 1, "r", "")
	conn := &model.Connection{Data: map[string]any{"id": uint32(9), "nick": "v"}}

	cases := []struct {
		ve, label string
	}{
		{"", "empty"},
		{"  ", "spaces"},
		{"x", "non-digits"},
		{"1a", "mixed"},
	}
	for _, tc := range cases {
		t.Run(tc.label, func(t *testing.T) {
			if out := c.joinPlayer(conn, map[string]string{"VE_PLAYER": tc.ve}); out != nil {
				t.Fatalf("expected nil (Perl bare return if no room lookup / invalid key), got %#v", out)
			}
		})
	}
}

func TestJoinPlCmdUnstartedRoomDelegatesToRoomInfoLikePerl(t *testing.T) {
	c := newControllerForJoinTests()
	c.Config.Raw["show_started_room_info"] = "0"
	c.Store.Players[1] = &state.Player{ID: 1, Nick: "host", ConnectedAt: time.Now().Add(-time.Minute)}
	_ = makeRoom(c, 704, 1, "JoinPlRoomTitle", "")

	// Observer not in any room; VE_PLAYER=host id → same as room_info_dgl VE_RID=room id
	conn := &model.Connection{Data: map[string]any{"id": uint32(99), "nick": "observer"}}
	out := c.joinPlayer(conn, map[string]string{"VE_PLAYER": "1"})
	if len(out) != 1 || out[0].Name != "LW_show" {
		t.Fatalf("expected Perl join_pl_cmd → room_info_dgl single LW_show, got %#v", out)
	}
	body := out[0].Args[0]
	// room_info_dgl.cml: title and some vars can remain as TT fragments; header and join affordance are stable.
	if !strings.Contains(body, "Game info") || !strings.Contains(body, "join_game.dcml") {
		t.Fatalf("expected non-started room_info_dgl.cml content, got: %q", body)
	}
}

func TestJoinPlCmdViaDispatchOpenWiring(t *testing.T) {
	c := newControllerForJoinTests()
	c.Config.Raw["show_started_room_info"] = "0"
	c.Store.Players[1] = &state.Player{ID: 1, Nick: "h", ConnectedAt: time.Now()}
	_ = makeRoom(c, 705, 1, "R705", "")

	req := &gsc.Stream{Ver: 2}
	conn := &model.Connection{Data: map[string]any{"id": uint32(2), "nick": "g"}}
	out := c.dispatchOpen(nil, conn, req, "join_pl_cmd", map[string]string{"VE_PLAYER": "1"})
	if len(out) != 1 || out[0].Name != "LW_show" {
		t.Fatalf("expected open route join_pl_cmd, got %#v", out)
	}
	body := out[0].Args[0]
	if !strings.Contains(body, "Game info") || !strings.Contains(body, "join_game.dcml") {
		t.Fatalf("expected room_info_dgl via dispatchOpen, got: %q", body)
	}
}

// TestJoinPlayerParityEdgeMatrix locks Open.pm join_pl_cmd branching: early empty,
// invalid VE_PLAYER (nil), no room (nil), started (error before room_info_dgl),
// and room_info delegation when the caller is not in a room (including non-uint32 id).
func TestJoinPlayerParityEdgeMatrix(t *testing.T) {
	c := newControllerForJoinTests()
	c.Config.Raw["show_started_room_info"] = "1"
	c.Store.Players[1] = &state.Player{ID: 1, Nick: "host", ConnectedAt: time.Now()}
	room := makeRoom(c, 800, 1, "matrix-room", "")
	room.Started = true
	c.Store.RoomsByPID[1] = room

	cases := []struct {
		name    string
		conn    *model.Connection
		params  map[string]string
		wantNil bool
		wantLen int
		substr  string
	}{
		{
			name:    "missing_ve_player",
			conn:    &model.Connection{Data: map[string]any{"id": uint32(2), "nick": "g"}},
			params:  map[string]string{},
			wantNil: true,
		},
		{
			name:    "ve_player_trimmed_spaces",
			conn:    &model.Connection{Data: map[string]any{"id": uint32(2), "nick": "g"}},
			params:  map[string]string{"VE_PLAYER": "  1  "},
			wantLen: 1,
			substr:  "Game alredy started",
		},
		{
			name:    "id_wrong_type_skips_early_empty",
			conn:    &model.Connection{Data: map[string]any{"id": int(2), "nick": "g"}},
			params:  map[string]string{"VE_PLAYER": "1"},
			wantLen: 1,
			substr:  "Game alredy started",
		},
		{
			name:    "uint32_id_not_in_room_still_routes",
			conn:    &model.Connection{Data: map[string]any{"id": uint32(50), "nick": "g50"}},
			params:  map[string]string{"VE_PLAYER": "1"},
			wantLen: 1,
			substr:  "Game alredy started",
		},
		{
			name:    "overflow_ve_player_parse_fails_to_zero",
			conn:    &model.Connection{Data: map[string]any{"id": uint32(2), "nick": "g"}},
			params:  map[string]string{"VE_PLAYER": "4294967296"},
			wantNil: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out := c.joinPlayer(tc.conn, tc.params)
			if tc.wantNil {
				if out != nil {
					t.Fatalf("expected nil, got %#v", out)
				}
				return
			}
			if out == nil {
				t.Fatalf("expected non-nil slice")
			}
			if len(out) != tc.wantLen {
				t.Fatalf("want len %d, got %#v", tc.wantLen, out)
			}
			if tc.substr != "" && !strings.Contains(out[0].Args[0], tc.substr) {
				t.Fatalf("expected body to contain %q, got %q", tc.substr, out[0].Args[0])
			}
		})
	}
}

func TestJoinPlayerStartedRoomUsesErrorNotStartedRoomInfoCML(t *testing.T) {
	// Even with show_started_room_info, join_pl_cmd checks started before delegating (Perl).
	c := newControllerForJoinTests()
	c.Config.Raw["show_started_room_info"] = "1"
	c.Store.Players[1] = &state.Player{ID: 1, Nick: "host", ConnectedAt: time.Now()}
	room := makeRoom(c, 801, 1, "started801", "")
	room.Started = true
	c.Store.RoomsByPID[1] = room

	conn := &model.Connection{Data: map[string]any{"id": uint32(9), "nick": "v"}}
	out := c.joinPlayer(conn, map[string]string{"VE_PLAYER": "1"})
	if len(out) != 1 || !strings.Contains(out[0].Args[0], "Game alredy started") {
		t.Fatalf("expected alert_dgl only, got %#v", out)
	}
	if strings.Contains(out[0].Args[0], "started_room_info") {
		t.Fatalf("did not expect started_room_info template in join_pl_cmd started branch")
	}
}
