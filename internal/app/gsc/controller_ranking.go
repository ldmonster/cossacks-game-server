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

// LCN ranking + GG Cup external data loaders.
// The cache implementation lives in internal/server/ranking; this file
// keeps thin Controller-scoped wrappers and the GG-Cup → startup-var
// merge helper for in-package call sites.

package gsc

import "github.com/ldmonster/cossacks-game-server/internal/app/ranking"

// loadLCNRanking / loadGGCup: thin wrappers preserved for existing call sites.
func (c *Controller) loadLCNRanking() map[string]any {
	if c.ranking == nil {
		return nil
	}

	return c.ranking.LoadLCN(c.Game.LCNRanking)
}

func (c *Controller) loadGGCup() map[string]any {
	if c.ranking == nil {
		return nil
	}

	return c.ranking.LoadGGCup(c.Game.GGCupFile)
}

// anyToStringVar is an in-package alias kept for the existing unit test
// (controller_supporter_amount_test.go). The canonical implementation
// lives in the ranking package.
func anyToStringVar(v any) string { return ranking.AnyToStringVar(v) }
