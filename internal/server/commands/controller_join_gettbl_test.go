package commands

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"cossacksgameserver/golang/internal/config"
	"cossacksgameserver/golang/internal/protocol/gsc"
	"cossacksgameserver/golang/internal/server/model"
	"cossacksgameserver/golang/internal/server/state"
)

func newControllerForJoinTests() *Controller {
	// Ensure templates are discoverable when tests run from this package cwd.
	if wd, err := os.Getwd(); err == nil {
		templateRoots = []string{
			filepath.Clean(filepath.Join(wd, "../../../templates")),
			filepath.Clean(filepath.Join(wd, "../../../../golang/templates")),
		}
	}
	return &Controller{
		Config: &config.Config{
			ShowStartedRooms: true,
			Raw:              map[string]string{},
		},
		Store: state.NewStore(),
	}
}

func makeRoom(c *Controller, id, hostID uint32, title, password string) *state.Room {
	host := c.Store.Players[hostID]
	r := &state.Room{
		ID:           id,
		Title:        title,
		HostID:       hostID,
		HostAddr:     "1.2.3.4",
		HostAddrInt:  0,
		Ver:          2,
		Password:     password,
		MaxPlayers:   8,
		PlayersCount: 1,
		Players:      map[uint32]*state.Player{hostID: host},
		PlayersTime:  map[uint32]time.Time{hostID: time.Now()},
		Row:          []string{"1", "", title, host.Nick, "For all", "1/8", "2"},
		Ctime:        time.Now(),
	}
	r.CtlSum = state.RoomControlSum(r.Row)
	c.Store.RoomsByID[id] = r
	c.Store.RoomsByPID[hostID] = r
	c.Store.RoomsBySum[r.CtlSum] = r
	return r
}

func TestJoinGameInvalidRidReturnsEmptyDialog(t *testing.T) {
	c := newControllerForJoinTests()
	conn := &model.Connection{Data: map[string]any{"id": uint32(10), "nick": "p1"}}
	got := c.joinGame(nil, conn, nil, map[string]string{"VE_RID": "abc", "ASTATE": "1"})
	if len(got) != 1 || got[0].Name != "LW_show" || got[0].Args[0] != "<NGDLG>\n<NGDLG>" {
		t.Fatalf("expected empty dialog, got %#v", got)
	}
}

func TestJoinGameAstateGuard(t *testing.T) {
	c := newControllerForJoinTests()
	c.Store.Players[1] = &state.Player{ID: 1, Nick: "h"}
	c.Store.Players[2] = &state.Player{ID: 2, Nick: "g"}
	makeRoom(c, 1, 1, "room", "")
	conn := &model.Connection{Data: map[string]any{"id": uint32(2), "nick": "g"}}
	got := c.joinGame(nil, conn, nil, map[string]string{"VE_RID": "1"})
	if len(got) != 1 || !strings.Contains(got[0].Args[0], "You can not create or join room!") {
		t.Fatalf("expected ASTATE guard error, got %#v", got)
	}
}

func TestJoinGamePasswordMismatchShowsConfirm(t *testing.T) {
	c := newControllerForJoinTests()
	c.Store.Players[1] = &state.Player{ID: 1, Nick: "h"}
	c.Store.Players[2] = &state.Player{ID: 2, Nick: "g"}
	makeRoom(c, 1, 1, "room", "secret")
	conn := &model.Connection{Data: map[string]any{"id": uint32(2), "nick": "g"}}
	got := c.joinGame(nil, conn, nil, map[string]string{"VE_RID": "1", "ASTATE": "1", "VE_PASSWD": "bad"})
	if len(got) != 1 || !strings.Contains(got[0].Args[0], "Password is required to join this game!") {
		t.Fatalf("expected password confirm dialog, got %#v", got)
	}
}

func TestJoinGameSuccessReturnsJoinRoomPayload(t *testing.T) {
	c := newControllerForJoinTests()
	c.Store.Players[1] = &state.Player{ID: 1, Nick: "h"}
	c.Store.Players[2] = &state.Player{ID: 2, Nick: "g"}
	makeRoom(c, 1, 1, "room", "")
	conn := &model.Connection{Data: map[string]any{"id": uint32(2), "nick": "g"}}
	got := c.joinGame(nil, conn, nil, map[string]string{"VE_RID": "1", "ASTATE": "1"})
	if len(got) != 1 || !strings.Contains(got[0].Args[0], "%COMMAND&JGAME") {
		t.Fatalf("expected join_room payload, got %#v", got)
	}
}

func TestGetTblUnknownChecksumProducesDtbl(t *testing.T) {
	c := newControllerForJoinTests()
	req := &gsc.Stream{Ver: 2}
	conn := &model.Connection{Data: map[string]any{}}
	unknown := uint32(0x11223344)
	pack := make([]byte, 4)
	binary.LittleEndian.PutUint32(pack, unknown)
	out := c.handleGETTBL(conn, req, []string{"ROOMS_V2\x00", "0", string(pack)})
	if len(out) != 2 || out[0].Name != "LW_dtbl" {
		t.Fatalf("unexpected GETTBL output: %#v", out)
	}
	b := []byte(out[0].Args[1])
	if len(b) != 4 || binary.LittleEndian.Uint32(b) != unknown {
		t.Fatalf("expected dtbl with unknown checksum, got %v", b)
	}
}
