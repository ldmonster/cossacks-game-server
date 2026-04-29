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

	"github.com/ldmonster/cossacks-game-server/internal/domain/player"
	"github.com/ldmonster/cossacks-game-server/internal/domain/session"
	tconn "github.com/ldmonster/cossacks-game-server/internal/transport/tconn"
)

func TestAliveTimerRemovesPlayerFromRoom(t *testing.T) {
	c := newControllerForJoinTests()
	c.session.SetTTL(20 * time.Millisecond)

	c.Store.SetPlayer(&player.Player{ID: 1, Nick: "p1", ConnectedAt: time.Now()})
	room := makeRoom(c, 201, 1, "timer-room", "")
	conn := &tconn.Connection{Session: &session.Session{PlayerID: 1, Nick: "p1"}}
	c.session.Register(1, conn)
	c.Store.IndexRoomByHost(1, room)

	c.armAliveTimer(1)
	time.Sleep(80 * time.Millisecond)

	if _, ok := c.Store.FindRoomByHost(1); ok {
		t.Fatalf("expected player to be removed from room by not_alive timer")
	}
	if conn.IsClosed() {
		t.Fatalf("not_alive should leave room without force-closing socket")
	}
}

func TestOnDisconnectRemovesPlayerAndLeavesHostRoom(t *testing.T) {
	c := newControllerForJoinTests()

	c.Store.SetPlayer(&player.Player{ID: 1, Nick: "host", ConnectedAt: time.Now()})
	_ = makeRoom(c, 301, 1, "dc-room", "")
	conn := &tconn.Connection{Session: &session.Session{PlayerID: 1, Nick: "host"}}
	c.session.Register(1, conn)

	c.OnDisconnect(conn)

	if _, ok := c.Store.FindPlayer(1); ok {
		t.Fatalf("expected player removed from store")
	}
	if _, ok := c.Store.FindRoomByHost(1); ok {
		t.Fatalf("expected host removed from room mapping")
	}
	if _, ok := c.Store.FindRoom(301); ok {
		t.Fatalf("expected room deleted when host disconnects (pre-start)")
	}
	if _, ok := c.session.Conn(1); ok {
		t.Fatalf("expected playerConns cleared")
	}
}

func TestLeaveClearsAliveTimer(t *testing.T) {
	c := newControllerForJoinTests()
	c.session.SetTTL(200 * time.Millisecond)

	c.Store.SetPlayer(&player.Player{ID: 1, Nick: "p1", ConnectedAt: time.Now()})
	room := makeRoom(c, 202, 1, "leave-room", "")
	conn := &tconn.Connection{Session: &session.Session{PlayerID: 1, Nick: "p1"}}
	c.session.Register(1, conn)
	c.Store.IndexRoomByHost(1, room)
	c.armAliveTimer(1)

	c.leaveRoom(conn)
	time.Sleep(260 * time.Millisecond)

	if conn.IsClosed() {
		t.Fatalf("expected timer to be cleared on leave; connection should remain open")
	}
}
