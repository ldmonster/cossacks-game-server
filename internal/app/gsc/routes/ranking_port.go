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

package routes

// RankingProvider is the consumer port that bundles the ranking and
// GG-Cup data sources used by routes such as tournaments,
// gg_cup_thanks_dgl and user_details. The handler-side adapter
// captures the configured file paths so this port stays free of
// configuration concerns.
type RankingProvider interface {
	// LoadLCN returns the cached LCN ranking, or nil if unavailable.
	LoadLCN() map[string]any
	// LoadGGCup returns the cached GG-Cup payload, or nil if
	// unavailable.
	LoadGGCup() map[string]any
	// LCNPlaceByID returns the player-id → LCN place mapping. May be
	// nil if no ranking is loaded.
	LCNPlaceByID() map[string]int
}
