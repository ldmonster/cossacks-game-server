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

	"github.com/ldmonster/cossacks-game-server/internal/adapter/rooms"
	"github.com/ldmonster/cossacks-game-server/internal/domain/player"
	"github.com/ldmonster/cossacks-game-server/internal/domain/session"
	tconn "github.com/ldmonster/cossacks-game-server/internal/transport/tconn"
	"github.com/ldmonster/cossacks-game-server/internal/transport/gsc"
)

func TestJoinGameLeavesPreviousRoomBeforeJoiningAnother(t *testing.T) {
	c := newControllerForJoinTests()
	c.Store.SetPlayer(&player.Player{ID: 1, Nick: "host1"})
	c.Store.SetPlayer(&player.Player{ID: 2, Nick: "guest"})
	c.Store.SetPlayer(&player.Player{ID: 3, Nick: "host2"})
	roomA := makeRoom(c, 1, 1, "room-a", "")
	roomB := makeRoom(c, 2, 3, "room-b", "")
	roomA.Players[2] = c.Store.GetPlayer(2)
	roomA.PlayersTime[2] = roomA.PlayersTime[1]
	roomA.PlayersCount = 2
	roomA.Row = setRoomPlayersColumn(roomA.Row, 2, roomA.MaxPlayers)
	roomA.CtlSum = rooms.RoomControlSum(roomA.Row)
	c.Store.IndexRoomBySum(roomA)
	c.Store.IndexRoomByHost(2, roomA)

	conn := &tconn.Connection{Session: &session.Session{PlayerID: 2, Nick: "guest"}}
	req := &gsc.Stream{Ver: 2}
	out, _ := c.joinGame(nil, conn, req, map[string]string{"VE_RID": "2", "ASTATE": "1"})
	if len(out) != 1 {
		t.Fatalf("expected join response, got %#v", out)
	}
	if c.Store.GetRoomByHost(2) != roomB {
		t.Fatalf("expected guest in room B, got %#v", c.Store.GetRoomByHost(2))
	}
	if c.Store.GetRoomByHost(1) != roomA {
		t.Fatalf("expected host still mapped to room A")
	}
	if roomA.PlayersCount != 1 {
		t.Fatalf("expected room A count back to 1 after guest left, got %d", roomA.PlayersCount)
	}
	if _, ok := roomA.Players[2]; ok {
		t.Fatalf("expected guest removed from room A players map")
	}
}
