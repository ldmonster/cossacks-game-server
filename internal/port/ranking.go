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

package port

// RankingService exposes the external ranking data (LCN ranking, GG Cup)
// to the dispatcher and any service that needs to render leaderboards or
// place-by-id lookups.
//
// The current implementation (ranking.Cache) reads JSON files from disk
// on demand with mtime caching. Future implementations can swap in an
// HTTP-backed cache without changing callers.
type RankingService interface {
	// LoadLCN reads and caches the LCN ranking JSON from path.
	// Returns nil when path is empty or the file cannot be read.
	LoadLCN(path string) map[string]any
	// LoadGGCup reads and caches the GG Cup JSON from path.
	// Returns nil when path is empty or the file cannot be read.
	LoadGGCup(path string) map[string]any
	// LCNPlaceByID returns the (lcn_id → place) snapshot from the most
	// recently loaded LCN payload, or nil if no payload has been loaded.
	LCNPlaceByID() map[string]int
}
