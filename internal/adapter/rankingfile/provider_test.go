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

package fileranking

import (
	"os"
	"path/filepath"
	"testing"
)

func writeJSON(t *testing.T, dir, name, body string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestLCNRankingParsesPlaces(t *testing.T) {
	dir := t.TempDir()
	path := writeJSON(t, dir, "lcn.json", `{
		"ranking": {
			"total": [
				{"id": "alice", "place": 1},
				{"id": "bob",   "place": 2.0},
				{"id": "carol", "place": "3"}
			]
		}
	}`)

	p := New(path, "")
	got := p.LCNRanking()
	if got == nil {
		t.Fatalf("expected ranking, got nil")
	}

	if p.LCNPlace("alice") != 1 || p.LCNPlace("bob") != 2 || p.LCNPlace("carol") != 3 {
		t.Fatalf("place mismatch: alice=%d bob=%d carol=%d",
			p.LCNPlace("alice"), p.LCNPlace("bob"), p.LCNPlace("carol"))
	}

	if p.LCNPlace("dave") != 0 {
		t.Fatalf("unknown id should be 0, got %d", p.LCNPlace("dave"))
	}
}

func TestLCNRankingMissingFileReturnsNil(t *testing.T) {
	p := New(filepath.Join(t.TempDir(), "missing.json"), "")
	if got := p.LCNRanking(); got != nil {
		t.Fatalf("missing file should yield nil, got %v", got)
	}
	if got := p.LCNPlace("x"); got != 0 {
		t.Fatalf("missing file place should be 0, got %d", got)
	}
}

func TestLCNRankingEmptyPathDisabled(t *testing.T) {
	p := New("", "")
	if p.LCNRanking() != nil || p.LCNPlace("x") != 0 {
		t.Fatalf("empty path should disable LCN")
	}
}

func TestGGCupHappyPath(t *testing.T) {
	dir := t.TempDir()
	path := writeJSON(t, dir, "gg.json", `{"id":"x","started":1,"prize_fund":100}`)

	p := New("", path)
	got := p.GGCup()
	if got == nil || got["id"] != "x" {
		t.Fatalf("unexpected gg cup payload: %v", got)
	}
}

func TestGGCupMissingFileYieldsWoInfo(t *testing.T) {
	p := New("", filepath.Join(t.TempDir(), "missing.json"))
	got := p.GGCup()
	if got == nil || got["wo_info"] != true {
		t.Fatalf("missing file should yield wo_info stub, got %v", got)
	}
}

func TestGGCupEmptyFileYieldsWoInfo(t *testing.T) {
	dir := t.TempDir()
	path := writeJSON(t, dir, "empty.json", "")

	p := New("", path)
	got := p.GGCup()
	if got == nil || got["wo_info"] != true {
		t.Fatalf("empty file should yield wo_info stub, got %v", got)
	}
}

func TestGGCupCorruptJSONYieldsWoInfo(t *testing.T) {
	dir := t.TempDir()
	path := writeJSON(t, dir, "bad.json", "{not-json")

	p := New("", path)
	got := p.GGCup()
	if got == nil || got["wo_info"] != true {
		t.Fatalf("bad json should yield wo_info stub, got %v", got)
	}
}

func TestGGCupEmptyPathDisabled(t *testing.T) {
	p := New("", "")
	if p.GGCup() != nil {
		t.Fatalf("empty path should disable GG cup")
	}
}
