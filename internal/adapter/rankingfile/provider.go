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

// Package fileranking implements port.RankingProvider on top of two
// JSON files on local disk (LCN ranking + GG Cup), with mtime-based
// caching that mirrors the earlier controller behavior. It is a port
// adapter: callers depend on the interface, not on file paths.
//
// Wiring this adapter into Controller is deferred — Controller currently
// owns equivalent loader methods in `controller_ranking.go`.
package fileranking

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"sync"
)

// Provider is the file-backed RankingProvider. It is safe for concurrent
// use; reloads are mtime-gated so repeated reads are cheap.
type Provider struct {
	mu sync.RWMutex

	lcnPath   string
	ggCupPath string

	lcnMTime int64
	lcnData  map[string]any
	lcnPlace map[string]int

	ggCupMTime int64
	ggCupData  map[string]any
}

// New constructs a Provider. Empty paths disable the corresponding loader
// (returns nil from the matching accessor).
func New(lcnRankingPath, ggCupPath string) *Provider {
	return &Provider{lcnPath: lcnRankingPath, ggCupPath: ggCupPath}
}

// LCNRanking satisfies port.RankingProvider.
func (p *Provider) LCNRanking() map[string]any {
	if p == nil || p.lcnPath == "" {
		return nil
	}

	p.refreshLCN()

	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.lcnData
}

// LCNPlace satisfies port.RankingProvider.
func (p *Provider) LCNPlace(id string) int {
	if p == nil || p.lcnPath == "" || id == "" {
		return 0
	}

	p.refreshLCN()

	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.lcnPlace[id]
}

// GGCup satisfies port.RankingProvider.
func (p *Provider) GGCup() map[string]any {
	if p == nil || p.ggCupPath == "" {
		return nil
	}

	p.refreshGGCup()

	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.ggCupData
}

func (p *Provider) refreshLCN() {
	st, err := os.Stat(p.lcnPath)
	if err != nil {
		p.mu.Lock()
		p.lcnData = nil
		p.lcnPlace = nil
		p.mu.Unlock()

		return
	}

	mtime := st.ModTime().Unix()

	p.mu.RLock()
	cached := p.lcnData != nil && p.lcnMTime == mtime
	p.mu.RUnlock()

	if cached {
		return
	}

	raw, err := os.ReadFile(p.lcnPath)
	if err != nil {
		return
	}

	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return
	}

	placeByID := buildLCNPlaceIndex(payload)

	p.mu.Lock()
	p.lcnMTime = mtime
	p.lcnData = payload
	p.lcnPlace = placeByID
	p.mu.Unlock()
}

func (p *Provider) refreshGGCup() {
	st, err := os.Stat(p.ggCupPath)
	if err != nil {
		p.mu.Lock()
		p.ggCupData = map[string]any{"wo_info": true}
		p.mu.Unlock()

		return
	}

	mtime := st.ModTime().Unix()

	p.mu.RLock()
	cached := p.ggCupData != nil && p.ggCupMTime == mtime
	p.mu.RUnlock()

	if cached {
		return
	}

	raw, err := os.ReadFile(p.ggCupPath)
	if err != nil || len(raw) == 0 {
		p.mu.Lock()
		p.ggCupMTime = mtime
		p.ggCupData = map[string]any{"wo_info": true}
		p.mu.Unlock()

		return
	}

	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		p.mu.Lock()
		p.ggCupMTime = mtime
		p.ggCupData = map[string]any{"wo_info": true}
		p.mu.Unlock()

		return
	}

	p.mu.Lock()
	p.ggCupMTime = mtime
	p.ggCupData = payload
	p.mu.Unlock()
}

// buildLCNPlaceIndex extracts ranking.total[].{id,place} into a map. The
// rules mirror the earlier controller exactly so a future cut-over does not
// change observed place numbers.
func buildLCNPlaceIndex(payload map[string]any) map[string]int {
	rankingAny, ok := payload["ranking"].(map[string]any)
	if !ok {
		return nil
	}

	total, ok := rankingAny["total"].([]any)
	if !ok {
		return nil
	}

	out := make(map[string]int, len(total))
	for _, rowAny := range total {
		row, _ := rowAny.(map[string]any)
		id := fmt.Sprintf("%v", row["id"])
		place := 0

		switch v := row["place"].(type) {
		case float64:
			place = int(v)
		case int:
			place = v
		case string:
			if n, err := strconv.Atoi(v); err == nil {
				place = n
			}
		}

		if id != "" {
			out[id] = place
		}
	}

	return out
}
