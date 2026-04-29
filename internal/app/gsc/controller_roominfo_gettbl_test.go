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
	"encoding/binary"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/ldmonster/cossacks-game-server/internal/adapter/rooms"
	"github.com/ldmonster/cossacks-game-server/internal/domain/player"
	"github.com/ldmonster/cossacks-game-server/internal/domain/session"
	tconn "github.com/ldmonster/cossacks-game-server/internal/transport/tconn"
	"github.com/ldmonster/cossacks-game-server/internal/transport/gsc"
)

func TestRoomInfoStartedBranchSelection(t *testing.T) {
	c := newControllerForJoinTests()
	c.Game.ShowStartedRoomInfo = true
	c.Store.SetPlayer(&player.Player{ID: 1, Nick: "host", ConnectedAt: time.Now()})
	room := makeRoom(c, 100, 1, "started-room", "")
	room.Started = true
	room.StartedAt = time.Now().Add(-3 * time.Minute)
	room.Map = "testmap.m3d"
	room.TimeTick = 4321

	conn := &tconn.Connection{Session: &session.Session{PlayerID: 2, Nick: "viewer"}}
	got, _ := c.roomInfo(conn, nil, map[string]string{"VE_RID": "100"})
	if len(got) != 1 {
		t.Fatalf("expected single response, got %#v", got)
	}
	body := got[0].Args[0]
	if !strings.Contains(body, "time:") {
		t.Fatalf("expected started room template payload, got: %q", body)
	}
	if !strings.Contains(body, "testmap.m3d") {
		t.Fatalf("expected started room map in payload, got: %q", body)
	}
}

func TestRoomInfoNonStartedBranchSelection(t *testing.T) {
	c := newControllerForJoinTests()
	c.Game.ShowStartedRoomInfo = false
	c.Store.SetPlayer(&player.Player{ID: 1, Nick: "host", ConnectedAt: time.Now()})
	_ = makeRoom(c, 101, 1, "normal-room", "")

	conn := &tconn.Connection{Session: &session.Session{PlayerID: 2, Nick: "viewer"}}
	got, _ := c.roomInfo(conn, nil, map[string]string{"VE_RID": "101"})
	if len(got) != 1 {
		t.Fatalf("expected single response, got %#v", got)
	}
	body := got[0].Args[0]
	if !strings.Contains(body, "Game info") {
		t.Fatalf("expected normal room info dialog payload, got: %q", body)
	}
}

// the reference passes the room object; Go also supplies room.id for simple <? … ?> (Join line).
// Full `room_info_dgl.cml` still has TT-only blocks; we lock in what the engine resolves.
func TestRoomInfoDglResolvesJoinRoomIdDotted(t *testing.T) {
	c := newControllerForJoinTests()
	c.Game.ShowStartedRoomInfo = false
	c.Store.SetPlayer(&player.Player{ID: 1, Nick: "HostNick", ConnectedAt: time.Now()})
	_ = makeRoom(c, 201, 1, "DottedTitleOK", "pw")
	conn := &tconn.Connection{Session: &session.Session{PlayerID: 2, Nick: "v"}}
	got, _ := c.roomInfo(conn, nil, map[string]string{"VE_RID": "201"})
	if len(got) != 1 {
		t.Fatalf("expected single response, got %#v", got)
	}
	body := got[0].Args[0]
	if !strings.Contains(body, "VE_RID=201") {
		t.Fatalf("expected join line with room.id, got: %q", body)
	}
}

func TestRoomInfoVE_RIDValidation(t *testing.T) {
	c := newControllerForJoinTests()
	conn := &tconn.Connection{Session: &session.Session{PlayerID: 2, Nick: "viewer"}}
	req := &gsc.Stream{Ver: 2}
	cases := []struct {
		name      string
		veRID     string
		wantNGDLG bool
	}{
		{name: "empty", veRID: "", wantNGDLG: true},
		{name: "spaces", veRID: "   ", wantNGDLG: true},
		{name: "mixed", veRID: "1a", wantNGDLG: true},
		{name: "spaced zero", veRID: " 0 ", wantNGDLG: true},
		{name: "spaced number", veRID: " 1 ", wantNGDLG: true},
		{name: "zero numeric", veRID: "0", wantNGDLG: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out, _ := c.roomInfo(conn, req, map[string]string{"VE_RID": tc.veRID})
			if len(out) != 1 || out[0].Name != "LW_show" {
				t.Fatalf("unexpected response: %#v", out)
			}
			body := out[0].Args[0]
			if tc.wantNGDLG {
				if body != "<NGDLG>\n<NGDLG>" {
					t.Fatalf("expected NGDLG pair for VE_RID=%q, got %q", tc.veRID, body)
				}
				return
			}
			if !strings.Contains(body, "The room is closed") {
				t.Fatalf("expected room closed error for VE_RID=%q, got %q", tc.veRID, body)
			}
		})
	}
}

func TestGetTblHideStartedBehavior(t *testing.T) {
	c := newControllerForJoinTests()
	c.Game.ShowStartedRooms = false
	c.Store.SetPlayer(&player.Player{ID: 1, Nick: "host", ConnectedAt: time.Now()})
	room := makeRoom(c, 102, 1, "started-hide", "")
	room.Started = true
	room.CtlSum = rooms.RoomControlSum(room.Row)
	c.Store.IndexRoomBySum(room)

	pack := make([]byte, 4)
	binary.LittleEndian.PutUint32(pack, room.CtlSum)
	conn := &tconn.Connection{Session: &session.Session{}}
	req := &gsc.Stream{Ver: 2}

	out, _ := c.handleGETTBL(conn, req, []string{"ROOMS_V2\x00", "0", string(pack)})
	if len(out) != 2 || out[0].Name != "LW_dtbl" || out[1].Name != "LW_tbl" {
		t.Fatalf("unexpected GETTBL responses: %#v", out)
	}
	dtbl := []byte(out[0].Args[1])
	if len(dtbl) != 4 || binary.LittleEndian.Uint32(dtbl) != room.CtlSum {
		t.Fatalf("expected started room checksum in dtbl, got %v", dtbl)
	}
	if out[1].Args[1] != "0" {
		t.Fatalf("expected no tbl rows for hidden started room, got count=%s args=%v", out[1].Args[1], out[1].Args)
	}
}

func TestGetTblIncludesVisibleRoomRows(t *testing.T) {
	c := newControllerForJoinTests()
	c.Game.ShowStartedRooms = true
	c.Store.SetPlayer(&player.Player{ID: 1, Nick: "host", ConnectedAt: time.Now()})
	room := makeRoom(c, 103, 1, "visible-room", "")
	room.Row = []string{"103", "", "visible-room", "host", "For all", "1/8", "2"}
	room.CtlSum = rooms.RoomControlSum(room.Row)
	c.Store.IndexRoomBySum(room)

	conn := &tconn.Connection{Session: &session.Session{}}
	req := &gsc.Stream{Ver: 2}
	out, _ := c.handleGETTBL(conn, req, []string{"ROOMS_V2\x00", "0", ""})
	if len(out) != 2 || out[1].Name != "LW_tbl" {
		t.Fatalf("unexpected GETTBL responses: %#v", out)
	}
	if out[1].Args[1] != "1" {
		t.Fatalf("expected one tbl row, got count=%s", out[1].Args[1])
	}
	all := strings.Join(out[1].Args, "|")
	if !strings.Contains(all, fmt.Sprintf("%d", room.ID)) || !strings.Contains(all, room.Title) {
		t.Fatalf("expected room row content in LW_tbl args, got %v", out[1].Args)
	}
}

func TestRoomInfoBacktoUserDetailsUsesCallerID(t *testing.T) {
	c := newControllerForJoinTests()
	c.Game.ShowStartedRoomInfo = false
	c.Store.SetPlayer(&player.Player{ID: 1, Nick: "host", ConnectedAt: time.Now()})
	_ = makeRoom(c, 301, 1, "BacktoRoom", "")

	conn := &tconn.Connection{Session: &session.Session{PlayerID: 77, Nick: "viewer"}}
	got, _ := c.roomInfo(conn, &gsc.Stream{Ver: 2}, map[string]string{
		"VE_RID": "301",
		"BACKTO": "user_details",
	})
	if len(got) != 1 || got[0].Name != "LW_show" {
		t.Fatalf("expected single LW_show, got %#v", got)
	}
	if !strings.Contains(got[0].Args[0], "open&user_details.dcml&ID=77") {
		t.Fatalf("expected BACKTO user_details command in payload, got: %q", got[0].Args[0])
	}
}

func TestRoomInfoStartedStatcolsRouteRendersStatHeaders(t *testing.T) {
	c := newControllerForJoinTests()
	c.Game.ShowStartedRoomInfo = true
	c.Store.SetPlayer(&player.Player{ID: 1, Nick: "host", ConnectedAt: time.Now()})
	room := makeRoom(c, 302, 1, "StartedStats", "")
	room.Started = true
	room.StartedAt = time.Now().Add(-2 * time.Minute)
	room.StartedUsers = []*player.Player{c.Store.GetPlayer(1)}

	conn := &tconn.Connection{Session: &session.Session{PlayerID: 3, Nick: "observer"}}
	got, _ := c.roomInfo(conn, &gsc.Stream{Ver: 2}, map[string]string{
		"VE_RID": "302",
		"part":   "statcols",
		"page":   "1",
		"res":    "0",
	})
	if len(got) != 1 || got[0].Name != "LW_show" {
		t.Fatalf("expected single LW_show, got %#v", got)
	}
	body := got[0].Args[0]
	if !strings.Contains(body, "+wood") || !strings.Contains(body, "*casuality") {
		t.Fatalf("expected started statcols headers in payload, got: %q", body)
	}
}
