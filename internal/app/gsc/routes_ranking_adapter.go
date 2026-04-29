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
	gscroutes "github.com/ldmonster/cossacks-game-server/internal/app/gsc/routes"
)

// controllerRankingAdapter satisfies gscroutes.RankingProvider by
// capturing the configured file paths and delegating to the
// Controller's ranking cache. It is registered with the routes
// container during Controller construction.
type controllerRankingAdapter struct{ c *Controller }

var _ gscroutes.RankingProvider = controllerRankingAdapter{}

// LoadLCN delegates to the Controller's loadLCNRanking helper.
func (a controllerRankingAdapter) LoadLCN() map[string]any {
	if a.c == nil {
		return nil
	}

	return a.c.loadLCNRanking()
}

// LoadGGCup delegates to the Controller's loadGGCup helper.
func (a controllerRankingAdapter) LoadGGCup() map[string]any {
	if a.c == nil {
		return nil
	}

	return a.c.loadGGCup()
}

// LCNPlaceByID returns the LCN ranking place lookup, or nil when no
// ranking provider is configured.
func (a controllerRankingAdapter) LCNPlaceByID() map[string]int {
	if a.c == nil || a.c.ranking == nil {
		return nil
	}

	return a.c.ranking.LCNPlaceByID()
}
