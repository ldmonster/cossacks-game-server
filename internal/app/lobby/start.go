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
	"regexp"
	"strconv"
	"time"

	"github.com/ldmonster/cossacks-game-server/internal/domain/lobby"
)

var savRegexp = regexp.MustCompile(`^sav:\[(\d+)\]$`)

// MarkStarted promotes the supplied room to "started" state, applying
// row mutations and re-indexing the control sum so the
// new state is observable to GETTBL polls.
//
// sav is the optional `sav:[N]` token from the start payload (parsed
// for SaveFrom); mapName is the map identifier reported by the host.
func (s *Service) MarkStarted(r *lobby.Room, sav, mapName string) {
	if s == nil || s.repo == nil || r == nil {
		return
	}

	s.repo.UnindexBySum(r.CtlSum)
	r.Started = true
	r.StartedAt = time.Now().UTC()
	r.StartPlayers = r.PlayersCount
	r.Map = mapName
	r.SaveFrom = 0

	if m := savRegexp.FindStringSubmatch(sav); len(m) == 2 {
		if v, err := strconv.Atoi(m[1]); err == nil {
			r.SaveFrom = v
		}
	}

	if len(r.Row) > 1 {
		// flips the started marker in the row payload.
		r.Row[1] = "\x7f0018"
	}

	if len(r.Row) > 0 && len(r.Row[len(r.Row)-1]) > 0 {
		last := []byte(r.Row[len(r.Row)-1])
		last[0] = '1'
		r.Row[len(r.Row)-1] = string(last)
	}

	r.CtlSum = controlSum(r.Row)
	s.repo.IndexBySum(r)
}
