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
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/ldmonster/cossacks-game-server/internal/adapter/rooms"
	"github.com/ldmonster/cossacks-game-server/internal/app/connectivity"
	gscroutes "github.com/ldmonster/cossacks-game-server/internal/app/gsc/routes"
	"github.com/ldmonster/cossacks-game-server/internal/app/identity"
	lobbyapp "github.com/ldmonster/cossacks-game-server/internal/app/lobby"
	sgame "github.com/ldmonster/cossacks-game-server/internal/app/match"
	"github.com/ldmonster/cossacks-game-server/internal/app/ranking"
	"github.com/ldmonster/cossacks-game-server/internal/domain/lobby"
	"github.com/ldmonster/cossacks-game-server/internal/domain/player"
	"github.com/ldmonster/cossacks-game-server/internal/domain/session"
	"github.com/ldmonster/cossacks-game-server/internal/platform/config"
	"github.com/ldmonster/cossacks-game-server/internal/render"
	tconn "github.com/ldmonster/cossacks-game-server/internal/transport/tconn"
	"github.com/ldmonster/cossacks-game-server/internal/transport/gsc"
)

func newControllerForJoinTests() *Controller {
	// Templates are discovered relative to this package's cwd; instantiate
	// an instance-scoped renderer rather than mutating any package-global
	// rooms.
	var renderer *render.TemplateRenderer
	if wd, err := os.Getwd(); err == nil {
		renderer = render.NewTemplateRenderer(filepath.Clean(filepath.Join(wd, "../../../templates")))
	}
	store := rooms.NewStore()
	cfg := &config.Config{
		Game: config.GameConfig{ShowStartedRooms: true},
	}
	c := &Controller{
		Game:     cfg.Game,
		Auth:     cfg.AuthConfig(),
		Store:    store,
		Log:      zap.NewNop(),
		Renderer: renderer,
		session:  connectivity.NewManager(150 * time.Second),
		rooms:    lobbyapp.NewService(store.AsRoomRepo()),
		auth:     identity.NewService(cfg.AuthConfig(), nil, zap.NewNop()),
		game:     sgame.NewService(),
		ranking:  ranking.NewCache(),
	}
	c.routes = gscroutes.New(gscroutes.Deps{
		Renderer: c.Renderer,
		Auth:     cfg.AuthConfig(),
		Game:     &c.Game,
		Server:   &c.Server,
		Log:      c.Log,
		Ranking:  controllerRankingAdapter{c: c},
		Players:  c.Store,
		Rooms:    controllerRoomLifecycleAdapter{c: c},
		Lobby:    c.rooms,
		Storage:  nil,
		Identity: c.auth,
		Sessions: c.session,
	})
	c.cmds = defaultCommandRegistry(commandDeps{
		Alive:        c.session,
		Lobby:        c.rooms,
		AliveDriver:  c,
		StartAlive:   c,
		Match:        c.game,
		StartMatch:   c.game,
		Players:      c.Store,
		StartPlayers: c.Store,
		StartRooms:   c.Store,
		StartLobby:   c.rooms,
		StartAccount: c,
		RoomsLookup:  c.Store,
		GETTBLRooms:  c.Store,
		Routes:       c,
		Log:          c.Log,
	})
	return c
}

func makeRoom(c *Controller, id, hostID uint32, title, password string) *lobby.Room {
	host := c.Store.GetPlayer(hostID)
	r := &lobby.Room{
		ID:           id,
		Title:        title,
		HostID:       hostID,
		HostAddr:     "1.2.3.4",
		HostAddrInt:  0,
		Ver:          2,
		Password:     password,
		MaxPlayers:   8,
		PlayersCount: 1,
		Players:      map[uint32]*player.Player{hostID: host},
		PlayersTime:  map[uint32]time.Time{hostID: time.Now()},
		Row:          []string{"1", "", title, host.Nick, "For all", "1/8", "2"},
		Ctime:        time.Now(),
	}
	r.CtlSum = rooms.RoomControlSum(r.Row)
	c.Store.IndexRoomByID(r)
	c.Store.IndexRoomByHost(hostID, r)
	c.Store.IndexRoomBySum(r)
	return r
}

func TestJoinGameInvalidRidReturnsEmptyDialog(t *testing.T) {
	c := newControllerForJoinTests()
	conn := &tconn.Connection{Session: &session.Session{PlayerID: 10, Nick: "p1"}}
	got, _ := c.joinGame(nil, conn, nil, map[string]string{"VE_RID": "abc", "ASTATE": "1"})
	if len(got) != 1 || got[0].Name != "LW_show" || got[0].Args[0] != "<NGDLG>\n<NGDLG>" {
		t.Fatalf("expected empty dialog, got %#v", got)
	}
}

func TestJoinGameAstateGuard(t *testing.T) {
	c := newControllerForJoinTests()
	c.Store.SetPlayer(&player.Player{ID: 1, Nick: "h"})
	c.Store.SetPlayer(&player.Player{ID: 2, Nick: "g"})
	makeRoom(c, 1, 1, "room", "")
	conn := &tconn.Connection{Session: &session.Session{PlayerID: 2, Nick: "g"}}
	got, _ := c.joinGame(nil, conn, nil, map[string]string{"VE_RID": "1"})
	if len(got) != 1 || !strings.Contains(got[0].Args[0], "You can not create or join room!") {
		t.Fatalf("expected ASTATE guard error, got %#v", got)
	}
}

func TestJoinGamePasswordMismatchShowsConfirm(t *testing.T) {
	c := newControllerForJoinTests()
	c.Store.SetPlayer(&player.Player{ID: 1, Nick: "h"})
	c.Store.SetPlayer(&player.Player{ID: 2, Nick: "g"})
	makeRoom(c, 1, 1, "room", "secret")
	conn := &tconn.Connection{Session: &session.Session{PlayerID: 2, Nick: "g"}}
	got, _ := c.joinGame(nil, conn, nil, map[string]string{"VE_RID": "1", "ASTATE": "1", "VE_PASSWD": "bad"})
	if len(got) != 1 || !strings.Contains(got[0].Args[0], "Password is required to join this game!") {
		t.Fatalf("expected password confirm dialog, got %#v", got)
	}
}

func TestJoinGameSuccessReturnsJoinRoomPayload(t *testing.T) {
	c := newControllerForJoinTests()
	c.Store.SetPlayer(&player.Player{ID: 1, Nick: "h"})
	c.Store.SetPlayer(&player.Player{ID: 2, Nick: "g"})
	makeRoom(c, 1, 1, "room", "")
	conn := &tconn.Connection{Session: &session.Session{PlayerID: 2, Nick: "g"}}
	got, _ := c.joinGame(nil, conn, nil, map[string]string{"VE_RID": "1", "ASTATE": "1"})
	if len(got) != 1 || !strings.Contains(got[0].Args[0], "%COMMAND&JGAME") {
		t.Fatalf("expected join_room payload, got %#v", got)
	}
}

func TestGetTblUnknownChecksumProducesDtbl(t *testing.T) {
	c := newControllerForJoinTests()
	req := &gsc.Stream{Ver: 2}
	conn := &tconn.Connection{Session: &session.Session{}}
	unknown := uint32(0x11223344)
	pack := make([]byte, 4)
	binary.LittleEndian.PutUint32(pack, unknown)
	out, _ := c.handleGETTBL(conn, req, []string{"ROOMS_V2\x00", "0", string(pack)})
	if len(out) != 2 || out[0].Name != "LW_dtbl" {
		t.Fatalf("unexpected GETTBL output: %#v", out)
	}
	b := []byte(out[0].Args[1])
	if len(b) != 4 || binary.LittleEndian.Uint32(b) != unknown {
		t.Fatalf("expected dtbl with unknown checksum, got %v", b)
	}
}
