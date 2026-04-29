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
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/ldmonster/cossacks-game-server/internal/adapter/rooms"
	"github.com/ldmonster/cossacks-game-server/internal/app/connectivity"
	"github.com/ldmonster/cossacks-game-server/internal/domain/lobby"
	"github.com/ldmonster/cossacks-game-server/internal/domain/player"
	"github.com/ldmonster/cossacks-game-server/internal/domain/session"
	"github.com/ldmonster/cossacks-game-server/internal/platform/config"
	tconn "github.com/ldmonster/cossacks-game-server/internal/transport/tconn"
	lobbyapp "github.com/ldmonster/cossacks-game-server/internal/app/lobby"
)

func newTestController() *Controller {
	store := rooms.NewStore()
	return &Controller{
		Game:    config.GameConfig{ShowStartedRooms: true},
		Store:   store,
		Log:     zap.NewNop(),
		session: connectivity.NewManager(150 * time.Second),
		rooms:   lobbyapp.NewService(store.AsRoomRepo()),
	}
}

func TestLeaveRoomDeletesNonStartedWhenHostLeaves(t *testing.T) {
	c := newTestController()
	host := &player.Player{ID: 1, Nick: "host", ConnectedAt: time.Now()}
	guest := &player.Player{ID: 2, Nick: "guest", ConnectedAt: time.Now()}
	c.Store.SetPlayer(host)
	c.Store.SetPlayer(guest)

	room := &lobby.Room{
		ID:           10,
		Title:        "r",
		HostID:       1,
		MaxPlayers:   8,
		PlayersCount: 2,
		Players:      map[uint32]*player.Player{1: host, 2: guest},
		PlayersTime:  map[uint32]time.Time{1: time.Now(), 2: time.Now()},
		Row:          []string{"10", "", "r", "host", "For all", "2/8", "2"},
		Ctime:        time.Now(),
	}
	room.CtlSum = rooms.RoomControlSum(room.Row)
	c.Store.IndexRoomByID(room)
	c.Store.IndexRoomByHost(1, room)
	c.Store.IndexRoomByHost(2, room)
	c.Store.IndexRoomBySum(room)

	conn := &tconn.Connection{Session: &session.Session{PlayerID: 1}}
	c.leaveRoom(conn)

	if _, ok := c.Store.FindRoom(room.ID); ok {
		t.Fatalf("expected non-started room to be deleted when host leaves")
	}
}

func TestLeaveRoomKeepsStartedRoomWhenHostLeaves(t *testing.T) {
	c := newTestController()
	host := &player.Player{ID: 1, Nick: "host", ConnectedAt: time.Now()}
	guest := &player.Player{ID: 2, Nick: "guest", ConnectedAt: time.Now()}
	c.Store.SetPlayer(host)
	c.Store.SetPlayer(guest)

	room := &lobby.Room{
		ID:           11,
		Title:        "r2",
		HostID:       1,
		MaxPlayers:   8,
		PlayersCount: 2,
		Players:      map[uint32]*player.Player{1: host, 2: guest},
		PlayersTime:  map[uint32]time.Time{1: time.Now(), 2: time.Now()},
		Started:      true,
		StartedAt:    time.Now(),
		Row:          []string{"11", "\x7f0018", "r2", "host", "For all", "2/8", "1ABCDEF"},
		Ctime:        time.Now(),
	}
	room.CtlSum = rooms.RoomControlSum(room.Row)
	c.Store.IndexRoomByID(room)
	c.Store.IndexRoomByHost(1, room)
	c.Store.IndexRoomByHost(2, room)
	c.Store.IndexRoomBySum(room)

	conn := &tconn.Connection{Session: &session.Session{PlayerID: 1}}
	c.leaveRoom(conn)

	got := c.Store.GetRoom(room.ID)
	if got == nil {
		t.Fatalf("expected started room to remain after host leaves")
	}
	if got.PlayersCount != 1 {
		t.Fatalf("expected players_count to decrement to 1, got %d", got.PlayersCount)
	}
	if got.Players[1] == nil || got.Players[1].ExitedAt.IsZero() {
		t.Fatalf("expected host to be marked exited in started room")
	}
}

func TestLeaveRoomDeletesStartedRoomWhenLastPlayerLeaves(t *testing.T) {
	c := newTestController()
	host := &player.Player{ID: 1, Nick: "host", ConnectedAt: time.Now()}
	c.Store.SetPlayer(host)

	room := &lobby.Room{
		ID:           12,
		Title:        "r3",
		HostID:       1,
		MaxPlayers:   8,
		PlayersCount: 1,
		Players:      map[uint32]*player.Player{1: host},
		PlayersTime:  map[uint32]time.Time{1: time.Now()},
		Started:      true,
		StartedAt:    time.Now(),
		Row:          []string{"12", "\x7f0018", "r3", "host", "For all", "1/8", "1ABCDEF"},
		Ctime:        time.Now(),
	}
	room.CtlSum = rooms.RoomControlSum(room.Row)
	c.Store.IndexRoomByID(room)
	c.Store.IndexRoomByHost(1, room)
	c.Store.IndexRoomBySum(room)

	conn := &tconn.Connection{Session: &session.Session{PlayerID: 1}}
	c.leaveRoom(conn)

	if _, ok := c.Store.FindRoom(room.ID); ok {
		t.Fatalf("expected started room to be deleted when last player leaves")
	}
}
