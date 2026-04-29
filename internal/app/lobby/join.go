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
	"time"

	"github.com/ldmonster/cossacks-game-server/internal/domain/lobby"
	"github.com/ldmonster/cossacks-game-server/internal/domain/player"
)

// Join admits a player into the supplied room, applying the row /
// control-sum reindex side-effects.
//
// The caller is responsible for validating room state (started flag,
// password, capacity) and for ensuring the player has previously left
// any other room they were in.
func (s *Service) Join(r *lobby.Room, p *player.Player) {
	if s == nil || s.repo == nil || r == nil || p == nil {
		return
	}

	r.Players[p.ID] = p
	if r.PlayersTime == nil {
		r.PlayersTime = map[uint32]time.Time{}
	}

	r.PlayersTime[p.ID] = time.Now()
	r.PlayersCount++

	s.repo.UnindexBySum(r.CtlSum)
	r.Row = setPlayersColumn(r.Row, r.PlayersCount, r.MaxPlayers)
	r.CtlSum = controlSum(r.Row)
	s.repo.IndexByHost(player.PlayerID(p.ID), r)
	s.repo.IndexBySum(r)
}
