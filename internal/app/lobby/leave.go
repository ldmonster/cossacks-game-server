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

package lobby

import (
	"fmt"
	"strings"
	"time"

	"github.com/ldmonster/cossacks-game-server/internal/domain/lobby"
	"github.com/ldmonster/cossacks-game-server/internal/domain/player"
)

// Leave removes the host playerID from any room they currently occupy,
// applying semantics for pre-start vs started rooms. Returns
// the affected room (or nil if the player was not in any room) and a
// flag indicating whether the room itself was deleted from the registry.
func (s *Service) Leave(playerID player.PlayerID) (*lobby.Room, bool) {
	if s == nil || s.repo == nil {
		return nil, false
	}

	r, ok := s.repo.FindByHost(playerID)
	if !ok || r == nil {
		return nil, false
	}

	// P1: acquire per-room lock so concurrent STAT handlers for this room
	// are excluded while we mutate room fields. The store's own mutex
	// remains separate and is taken/released inside each repo method call.
	// Lock order: rooms.Store.mu (short-lived, already released by
	// FindByHost above) → per-room lock.
	mu := s.repo.RoomMu(lobby.RoomID(r.ID))
	mu.Lock()
	defer mu.Unlock()

	oldCtl := r.CtlSum

	s.repo.UnindexByHost(playerID)

	r.PlayersCount--

	pid := uint32(playerID)
	if r.Started {
		// in started rooms players remain in the list and
		// are merely flagged as exited.
		if pl := r.Players[pid]; pl != nil {
			pl.ExitedAt = time.Now().UTC()
		}
	} else {
		delete(r.Players, pid)
		delete(r.PlayersTime, pid)
		r.Row = setPlayersColumn(r.Row, r.PlayersCount, r.MaxPlayers)
	}

	s.repo.UnindexBySum(oldCtl)

	r.CtlSum = controlSum(r.Row)

	shouldDelete := false
	if r.Started {
		shouldDelete = r.PlayersCount <= 0
	} else {
		shouldDelete = r.HostID == pid || r.PlayersCount <= 0
	}

	if shouldDelete {
		s.repo.UnindexByID(lobby.RoomID(r.ID))
		s.repo.UnindexBySum(r.CtlSum)

		return r, true
	}

	s.repo.IndexBySum(r)

	return r, false
}

// setPlayersColumn replaces the "n/m" players column in row preserving
// the rest of the row. Pure helper.
func setPlayersColumn(row []string, playersCount, maxPlayers int) []string {
	if len(row) == 0 {
		return row
	}

	out := append([]string(nil), row...)

	idx := len(out) - 4
	if idx < 0 {
		idx = len(out) - 1
	}

	out[idx] = fmt.Sprintf("%d/%d", playersCount, maxPlayers)

	return out
}

// controlSum is the Adler-32-like checksum.
// Duplicated locally to avoid pulling in the
// state package from this layer (the canonical implementation in
// internal/server/rooms.RoomControlSum is byte-identical).
func controlSum(row []string) uint32 {
	const (
		mod   = 0xFFF1
		chunk = 5552
	)

	s := strings.Join(row, "")
	v1 := uint32(1)
	v2 := uint32(0)

	for i := 0; i < len(s); i += chunk {
		end := i + chunk
		if end > len(s) {
			end = len(s)
		}

		for j := i; j < end; j++ {
			v1 += uint32(s[j])
			v2 += v1
		}

		v1 %= mod
		v2 %= mod
	}

	return (v2 << 16) | v1
}
