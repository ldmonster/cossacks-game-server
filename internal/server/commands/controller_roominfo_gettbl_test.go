package commands

import (
	"encoding/binary"
	"fmt"
	"strings"
	"testing"
	"time"

	"cossacksgameserver/golang/internal/protocol/gsc"
	"cossacksgameserver/golang/internal/server/model"
	"cossacksgameserver/golang/internal/server/state"
)

func TestRoomInfoStartedBranchSelection(t *testing.T) {
	c := newControllerForJoinTests()
	c.Config.Raw["show_started_room_info"] = "1"
	c.Store.Players[1] = &state.Player{ID: 1, Nick: "host", ConnectedAt: time.Now()}
	room := makeRoom(c, 100, 1, "started-room", "")
	room.Started = true
	room.StartedAt = time.Now().Add(-3 * time.Minute)
	room.Map = "testmap.m3d"
	room.TimeTick = 4321

	conn := &model.Connection{Data: map[string]any{"id": uint32(2), "nick": "viewer"}}
	got := c.roomInfo(conn, nil, map[string]string{"VE_RID": "100"})
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
	c.Config.Raw["show_started_room_info"] = "0"
	c.Store.Players[1] = &state.Player{ID: 1, Nick: "host", ConnectedAt: time.Now()}
	_ = makeRoom(c, 101, 1, "normal-room", "")

	conn := &model.Connection{Data: map[string]any{"id": uint32(2), "nick": "viewer"}}
	got := c.roomInfo(conn, nil, map[string]string{"VE_RID": "101"})
	if len(got) != 1 {
		t.Fatalf("expected single response, got %#v", got)
	}
	body := got[0].Args[0]
	if !strings.Contains(body, "Game info") {
		t.Fatalf("expected normal room info dialog payload, got: %q", body)
	}
}

// Perl passes the room object; Go also supplies room.id for simple <? … ?> (Join line).
// Full `room_info_dgl.cml` still has TT-only blocks; we lock in what the engine resolves.
func TestRoomInfoDglResolvesJoinRoomIdDotted(t *testing.T) {
	c := newControllerForJoinTests()
	c.Config.Raw["show_started_room_info"] = "0"
	c.Store.Players[1] = &state.Player{ID: 1, Nick: "HostNick", ConnectedAt: time.Now()}
	_ = makeRoom(c, 201, 1, "DottedTitleOK", "pw")
	conn := &model.Connection{Data: map[string]any{"id": uint32(2), "nick": "v"}}
	got := c.roomInfo(conn, nil, map[string]string{"VE_RID": "201"})
	if len(got) != 1 {
		t.Fatalf("expected single response, got %#v", got)
	}
	body := got[0].Args[0]
	if !strings.Contains(body, "VE_RID=201") {
		t.Fatalf("expected join line with room.id, got: %q", body)
	}
}

func TestRoomInfoVE_RIDValidationParity(t *testing.T) {
	c := newControllerForJoinTests()
	conn := &model.Connection{Data: map[string]any{"id": uint32(2), "nick": "viewer"}}
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
			out := c.roomInfo(conn, req, map[string]string{"VE_RID": tc.veRID})
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
	c.Config.ShowStartedRooms = false
	c.Store.Players[1] = &state.Player{ID: 1, Nick: "host", ConnectedAt: time.Now()}
	room := makeRoom(c, 102, 1, "started-hide", "")
	room.Started = true
	room.CtlSum = state.RoomControlSum(room.Row)
	c.Store.RoomsBySum[room.CtlSum] = room

	pack := make([]byte, 4)
	binary.LittleEndian.PutUint32(pack, room.CtlSum)
	conn := &model.Connection{Data: map[string]any{"dev": false}}
	req := &gsc.Stream{Ver: 2}

	out := c.handleGETTBL(conn, req, []string{"ROOMS_V2\x00", "0", string(pack)})
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
	c.Config.ShowStartedRooms = true
	c.Store.Players[1] = &state.Player{ID: 1, Nick: "host", ConnectedAt: time.Now()}
	room := makeRoom(c, 103, 1, "visible-room", "")
	room.Row = []string{"103", "", "visible-room", "host", "For all", "1/8", "2"}
	room.CtlSum = state.RoomControlSum(room.Row)
	c.Store.RoomsBySum[room.CtlSum] = room

	conn := &model.Connection{Data: map[string]any{"dev": false}}
	req := &gsc.Stream{Ver: 2}
	out := c.handleGETTBL(conn, req, []string{"ROOMS_V2\x00", "0", ""})
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
	c.Config.Raw["show_started_room_info"] = "0"
	c.Store.Players[1] = &state.Player{ID: 1, Nick: "host", ConnectedAt: time.Now()}
	_ = makeRoom(c, 301, 1, "BacktoRoom", "")

	conn := &model.Connection{Data: map[string]any{"id": uint32(77), "nick": "viewer"}}
	got := c.roomInfo(conn, &gsc.Stream{Ver: 2}, map[string]string{
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
	c.Config.Raw["show_started_room_info"] = "1"
	c.Store.Players[1] = &state.Player{ID: 1, Nick: "host", ConnectedAt: time.Now()}
	room := makeRoom(c, 302, 1, "StartedStats", "")
	room.Started = true
	room.StartedAt = time.Now().Add(-2 * time.Minute)
	room.StartedUsers = []*state.Player{c.Store.Players[1]}

	conn := &model.Connection{Data: map[string]any{"id": uint32(3), "nick": "observer"}}
	got := c.roomInfo(conn, &gsc.Stream{Ver: 2}, map[string]string{
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
