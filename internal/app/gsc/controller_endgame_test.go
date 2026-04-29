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

	"github.com/ldmonster/cossacks-game-server/internal/domain/lobby"
	"github.com/ldmonster/cossacks-game-server/internal/domain/player"
	"github.com/ldmonster/cossacks-game-server/internal/domain/session"
	tconn "github.com/ldmonster/cossacks-game-server/internal/transport/tconn"
)

func TestParseEndgameSignedPlayerIDAndWinLabel(t *testing.T) {
	c := newControllerForJoinTests()
	c.Store.SetPlayer(&player.Player{ID: 42, Nick: "p42"})
	c.Store.IndexRoomByID(&lobby.Room{ID: 10, HostID: 7, Title: "room10"})
	conn := &tconn.Connection{Session: &session.Session{PlayerID: 7}}

	ev, ok := c.parseEndgame(conn, []string{"game=10", "pid=42", "result=2"})
	if !ok {
		t.Fatalf("expected parse success")
	}
	if ev.GameID != 10 || ev.PlayerID != 42 {
		t.Fatalf("unexpected ids: %+v", ev)
	}
	if ev.Result != "win" {
		t.Fatalf("expected win label, got %q", ev.Result)
	}
	if ev.Nick != "p42" {
		t.Fatalf("expected nick p42, got %q", ev.Nick)
	}
	if ev.Own != "his " {
		t.Fatalf("expected host ownership marker, got %q", ev.Own)
	}
	if ev.Title != " room10" {
		t.Fatalf("expected room title suffix, got %q", ev.Title)
	}
}

func TestParseEndgameNegativePlayerIDReinterpretedUint32(t *testing.T) {
	c := newControllerForJoinTests()
	conn := &tconn.Connection{Session: &session.Session{}}

	ev, ok := c.parseEndgame(conn, []string{"1", "-1", "5"})
	if !ok {
		t.Fatalf("expected parse success")
	}
	if ev.PlayerID != ^uint32(0) {
		t.Fatalf("expected unsigned reinterpretation of -1, got %d", ev.PlayerID)
	}
	if ev.Result != "disconnect" {
		t.Fatalf("expected disconnect label, got %q", ev.Result)
	}
	if ev.Nick != "." {
		t.Fatalf("expected missing-player nick '.', got %q", ev.Nick)
	}
}

func TestParseEndgameUnknownResultLabel(t *testing.T) {
	c := newControllerForJoinTests()
	conn := &tconn.Connection{Session: &session.Session{}}

	ev, ok := c.parseEndgame(conn, []string{"1", "2", "9"})
	if !ok {
		t.Fatalf("expected parse success")
	}
	if ev.Result != "?9?" {
		t.Fatalf("expected unknown result token, got %q", ev.Result)
	}
}
