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

package rooms

import (
	"testing"

	"github.com/ldmonster/cossacks-game-server/internal/domain/lobby"
	"github.com/ldmonster/cossacks-game-server/internal/domain/player"
	"github.com/ldmonster/cossacks-game-server/internal/port"
)

// Compile-time guarantees that *Store satisfies the typed repo and allocate ports.
var (
	_ port.PlayerRepository = (*Store)(nil)
	_ port.RoomRepository   = (*Store)(nil)
)

func TestStorePlayerByID(t *testing.T) {
	s := NewStore()
	p := &Player{ID: 7, Nick: "alice"}
	s.UpsertPlayer(p)

	got, ok := s.PlayerByID(player.PlayerID(7))
	if !ok {
		t.Fatalf("expected player 7, got miss")
	}
	if got != p {
		t.Fatalf("returned wrong player: %v vs %v", got, p)
	}

	if _, ok := s.PlayerByID(player.PlayerID(999)); ok {
		t.Fatalf("expected miss for unknown id")
	}
}

func TestStoreRoomByID(t *testing.T) {
	s := NewStore()
	r := &Room{ID: 4}
	s.IndexRoomByID(r)

	got, ok := s.RoomByID(lobby.RoomID(4))
	if !ok || got != r {
		t.Fatalf("RoomByID hit failed: ok=%v got=%v", ok, got)
	}

	if _, ok := s.RoomByID(lobby.RoomID(99)); ok {
		t.Fatalf("expected miss")
	}
}

func TestStoreRoomByPlayerID(t *testing.T) {
	s := NewStore()
	r := &Room{ID: 9}
	s.IndexRoomByHost(3, r)

	got, ok := s.RoomByPlayerID(player.PlayerID(3))
	if !ok || got != r {
		t.Fatalf("RoomByPlayerID hit failed: ok=%v got=%v", ok, got)
	}

	if _, ok := s.RoomByPlayerID(player.PlayerID(0)); ok {
		t.Fatalf("expected miss for unknown pid")
	}
}
