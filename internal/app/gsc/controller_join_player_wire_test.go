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
	"time"

	"github.com/ldmonster/cossacks-game-server/internal/domain/player"
	"github.com/ldmonster/cossacks-game-server/internal/domain/session"
	tconn "github.com/ldmonster/cossacks-game-server/internal/transport/tconn"
	"github.com/ldmonster/cossacks-game-server/internal/transport/gsc"
)

func TestJoinPlayerAlreadyInRoomReturnsEmpty(t *testing.T) {
	c := newControllerForJoinTests()
	c.Store.SetPlayer(&player.Player{ID: 1, Nick: "host"})
	c.Store.SetPlayer(&player.Player{ID: 2, Nick: "guest"})
	_ = makeRoom(c, 700, 1, "room700", "")
	_ = makeRoom(c, 701, 2, "room701", "")

	// Caller already participates in a room.
	conn := &tconn.Connection{Session: &session.Session{PlayerID: 2, Nick: "guest"}}
	out, _ := c.joinPlayer(conn, map[string]string{"VE_PLAYER": "1"})
	if out == nil || len(out) != 0 {
		t.Fatalf("expected empty command set for already-in-room player, got %#v", out)
	}
}

func TestJoinPlayerNoRoomReturnsNil(t *testing.T) {
	c := newControllerForJoinTests()
	conn := &tconn.Connection{Session: &session.Session{PlayerID: 10, Nick: "p10"}}
	out, _ := c.joinPlayer(conn, map[string]string{"VE_PLAYER": "999"})
	if out != nil {
		t.Fatalf("expected nil when target player room not found, got %#v", out)
	}
}

func TestJoinPlayerStartedRoomReturnsError(t *testing.T) {
	c := newControllerForJoinTests()
	c.Store.SetPlayer(&player.Player{ID: 1, Nick: "host"})
	c.Store.SetPlayer(&player.Player{ID: 2, Nick: "viewer"})
	room := makeRoom(c, 702, 1, "started-room", "")
	room.Started = true
	c.Store.IndexRoomByHost(1, room)

	conn := &tconn.Connection{Session: &session.Session{PlayerID: 2, Nick: "viewer"}}
	out, _ := c.joinPlayer(conn, map[string]string{"VE_PLAYER": "1"})
	if len(out) != 1 || !strings.Contains(out[0].Args[0], "Game alredy started") {
		t.Fatalf("expected started-room error, got %#v", out)
	}
}

func TestJoinPlCmdInvalidVEPlayerYieldsNoCommands(t *testing.T) {
	c := newControllerForJoinTests()
	c.Store.SetPlayer(&player.Player{ID: 1, Nick: "host"})
	_ = makeRoom(c, 703, 1, "r", "")
	conn := &tconn.Connection{Session: &session.Session{PlayerID: 9, Nick: "v"}}

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
			if out, _ := c.joinPlayer(conn, map[string]string{"VE_PLAYER": tc.ve}); out != nil {
				t.Fatalf("expected nil (bare return if no room lookup / invalid key), got %#v", out)
			}
		})
	}
}

func TestJoinPlCmdUnstartedRoomDelegatesToRoomInfo(t *testing.T) {
	c := newControllerForJoinTests()
	c.Game.ShowStartedRoomInfo = false
	c.Store.SetPlayer(&player.Player{ID: 1, Nick: "host", ConnectedAt: time.Now().Add(-time.Minute)})
	_ = makeRoom(c, 704, 1, "JoinPlRoomTitle", "")

	// Observer not in any room; VE_PLAYER=host id → same as room_info_dgl VE_RID=room id
	conn := &tconn.Connection{Session: &session.Session{PlayerID: 99, Nick: "observer"}}
	out, _ := c.joinPlayer(conn, map[string]string{"VE_PLAYER": "1"})
	if len(out) != 1 || out[0].Name != "LW_show" {
		t.Fatalf("expected reference join_pl_cmd → room_info_dgl single LW_show, got %#v", out)
	}
	body := out[0].Args[0]
	// room_info_dgl.cml: title and some vars can remain as TT fragments; header and join affordance are stable.
	if !strings.Contains(body, "Game info") || !strings.Contains(body, "join_game.dcml") {
		t.Fatalf("expected non-started room_info_dgl.cml content, got: %q", body)
	}
}

func TestJoinPlCmdViaDispatchOpenWiring(t *testing.T) {
	c := newControllerForJoinTests()
	c.Game.ShowStartedRoomInfo = false
	c.Store.SetPlayer(&player.Player{ID: 1, Nick: "h", ConnectedAt: time.Now()})
	_ = makeRoom(c, 705, 1, "R705", "")

	req := &gsc.Stream{Ver: 2}
	conn := &tconn.Connection{Session: &session.Session{PlayerID: 2, Nick: "g"}}
	out, _ := c.dispatchOpen(nil, conn, req, "join_pl_cmd", map[string]string{"VE_PLAYER": "1"})
	if len(out) != 1 || out[0].Name != "LW_show" {
		t.Fatalf("expected open route join_pl_cmd, got %#v", out)
	}
	body := out[0].Args[0]
	if !strings.Contains(body, "Game info") || !strings.Contains(body, "join_game.dcml") {
		t.Fatalf("expected room_info_dgl via dispatchOpen, got: %q", body)
	}
}

// TestJoinPlayerEdgeMatrix locks Open.pm join_pl_cmd branching: early empty,
// invalid VE_PLAYER (nil), no room (nil), started (error before room_info_dgl),
// and room_info delegation when the caller is not in a room (including non-uint32 id).
func TestJoinPlayerEdgeMatrix(t *testing.T) {
	c := newControllerForJoinTests()
	c.Game.ShowStartedRoomInfo = true
	c.Store.SetPlayer(&player.Player{ID: 1, Nick: "host", ConnectedAt: time.Now()})
	room := makeRoom(c, 800, 1, "matrix-room", "")
	room.Started = true
	c.Store.IndexRoomByHost(1, room)

	cases := []struct {
		name    string
		conn    *tconn.Connection
		params  map[string]string
		wantNil bool
		wantLen int
		substr  string
	}{
		{
			name:    "missing_ve_player",
			conn:    &tconn.Connection{Session: &session.Session{PlayerID: 2, Nick: "g"}},
			params:  map[string]string{},
			wantNil: true,
		},
		{
			name:    "ve_player_trimmed_spaces",
			conn:    &tconn.Connection{Session: &session.Session{PlayerID: 2, Nick: "g"}},
			params:  map[string]string{"VE_PLAYER": "  1  "},
			wantLen: 1,
			substr:  "Game alredy started",
		},
		{
			name:    "id_wrong_type_skips_early_empty",
			conn:    &tconn.Connection{Session: &session.Session{Nick: "g"} /* PlayerID intentionally 0 for bad-id test */},
			params:  map[string]string{"VE_PLAYER": "1"},
			wantLen: 1,
			substr:  "Game alredy started",
		},
		{
			name:    "uint32_id_not_in_room_still_routes",
			conn:    &tconn.Connection{Session: &session.Session{PlayerID: 50, Nick: "g50"}},
			params:  map[string]string{"VE_PLAYER": "1"},
			wantLen: 1,
			substr:  "Game alredy started",
		},
		{
			name:    "overflow_ve_player_parse_fails_to_zero",
			conn:    &tconn.Connection{Session: &session.Session{PlayerID: 2, Nick: "g"}},
			params:  map[string]string{"VE_PLAYER": "4294967296"},
			wantNil: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out, _ := c.joinPlayer(tc.conn, tc.params)
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
	// Even with show_started_room_info, join_pl_cmd checks started before delegating (reference).
	c := newControllerForJoinTests()
	c.Game.ShowStartedRoomInfo = true
	c.Store.SetPlayer(&player.Player{ID: 1, Nick: "host", ConnectedAt: time.Now()})
	room := makeRoom(c, 801, 1, "started801", "")
	room.Started = true
	c.Store.IndexRoomByHost(1, room)

	conn := &tconn.Connection{Session: &session.Session{PlayerID: 9, Nick: "v"}}
	out, _ := c.joinPlayer(conn, map[string]string{"VE_PLAYER": "1"})
	if len(out) != 1 || !strings.Contains(out[0].Args[0], "Game alredy started") {
		t.Fatalf("expected alert_dgl only, got %#v", out)
	}
	if strings.Contains(out[0].Args[0], "started_room_info") {
		t.Fatalf("did not expect started_room_info template in join_pl_cmd started branch")
	}
}
