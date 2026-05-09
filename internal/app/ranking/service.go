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

// Package ranking owns the LCN-ranking and GG-Cup external data caches.
package ranking

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"

	"github.com/ldmonster/cossacks-game-server/internal/port"
	"github.com/ldmonster/cossacks-game-server/internal/render"
)

// Cache holds the most recently parsed LCN ranking and GG Cup payloads,
// keyed by mtime. Construct via NewCache; the zero value is also valid.
type Cache struct {
	lcnMTime     int64
	lcnData      map[string]any
	lcnPlaceByID map[string]int

	ggCupMTime int64
	ggCupData  map[string]any
}

// NewCache returns an empty Cache.
func NewCache() *Cache { return &Cache{} }

// LCNPlaceByID returns the (id -> place) snapshot from the most recent
// LCN payload, or nil if no payload has been loaded.
func (c *Cache) LCNPlaceByID() map[string]int { return c.lcnPlaceByID }

// LoadLCN reads and caches the LCN ranking JSON from path.
func (c *Cache) LoadLCN(rawPath string) map[string]any {
	path := strings.TrimSpace(rawPath)
	if path == "" {
		return nil
	}

	st, err := os.Stat(path)
	if err != nil {
		return nil
	}

	mtime := st.ModTime().Unix()
	if c.lcnData != nil && c.lcnMTime == mtime {
		return c.lcnData
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil
	}

	if rankingAny, ok := payload["ranking"].(map[string]any); ok {
		if total, ok := rankingAny["total"].([]any); ok {
			placeByID := map[string]int{}

			for _, rowAny := range total {
				row, _ := rowAny.(map[string]any)
				id := fmt.Sprintf("%v", row["id"])
				place := 0

				switch v := row["place"].(type) {
				case float64:
					place = int(v)
				case int:
					place = v
				}

				if id != "" {
					placeByID[id] = place
				}
			}

			c.lcnPlaceByID = placeByID
		} else {
			c.lcnPlaceByID = nil
		}
	} else {
		c.lcnPlaceByID = nil
	}

	c.lcnMTime = mtime
	c.lcnData = payload

	return payload
}

// LoadGGCup reads and caches the GG Cup JSON from path. When the file is
// missing or unreadable the cached payload is reset to a sentinel
// {"wo_info": true} map.
func (c *Cache) LoadGGCup(rawPath string) map[string]any {
	path := strings.TrimSpace(rawPath)
	if path == "" {
		return nil
	}

	st, err := os.Stat(path)
	if err != nil {
		c.ggCupData = map[string]any{"wo_info": true}
		return c.ggCupData
	}

	mtime := st.ModTime().Unix()
	if c.ggCupData != nil && c.ggCupMTime == mtime {
		return c.ggCupData
	}

	raw, err := os.ReadFile(path)
	if err != nil || len(raw) == 0 {
		c.ggCupMTime = mtime
		c.ggCupData = map[string]any{"wo_info": true}

		return c.ggCupData
	}

	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		c.ggCupMTime = mtime
		c.ggCupData = map[string]any{"wo_info": true}

		return c.ggCupData
	}

	c.ggCupMTime = mtime
	c.ggCupData = payload

	return payload
}

// BuildGGCupView projects the gg_cup hash into a typed view used by
// startup.tmpl and gg_cup_thanks_dgl.tmpl. Returns nil when gg is nil
// so templates can detect absence with `{{if .GGCup}}`.
func BuildGGCupView(gg map[string]any) *render.GGCupView {
	if gg == nil {
		return nil
	}

	v := &render.GGCupView{}

	if s, ok := gg["id"]; ok {
		v.ID = anyToStringVar(s)
	}

	v.WoInfo = boolFromAny(gg["wo_info"])
	v.Started = boolFromAny(gg["started"])

	if s, ok := gg["players_count"]; ok {
		v.PlayersCount = anyToStringVar(s)
	}

	if s, ok := gg["prize_fund"]; ok {
		v.PrizeFund = anyToStringVar(s)
	}

	v.PlayersCountLen = len(v.PlayersCount)
	v.PrizeFundLen = len(v.PrizeFund)

	return v
}

func boolFromAny(v any) bool {
	switch t := v.(type) {
	case bool:
		return t
	case string:
		return t != "" && t != "0" && t != "false"
	case float64:
		return t != 0
	case int:
		return t != 0
	case nil:
		return false
	default:
		s := strings.TrimSpace(fmt.Sprint(v))
		return s != "" && s != "0" && s != "false"
	}
}

func anyToStringVar(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case bool:
		if t {
			return "1"
		}

		return "0"
	case float64:
		if t == math.Trunc(t) {
			return strconv.FormatInt(int64(t), 10)
		}

		return strconv.FormatFloat(t, 'f', -1, 64)
	default:
		return strings.TrimSpace(fmt.Sprint(v))
	}
}

// AnyToStringVar is the exported form of anyToStringVar used by callers
// outside this package.
func AnyToStringVar(v any) string { return anyToStringVar(v) }

// Compile-time interface check.
var _ port.RankingService = (*Cache)(nil)
