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

// Package game owns the in-game stats / endgame logic
package match

import (
	"encoding/binary"
	"fmt"
	"math"

	"github.com/ldmonster/cossacks-game-server/internal/domain/match"
)

// Service is the stateless façade for game-stat helpers.
type Service struct{}

// NewService returns a zero-valued Service. The constructor is provided
// for symmetry with other services so wiring code can be uniform.
func NewService() *Service { return &Service{} }

// DecodeStat decodes a 42-byte little-endian STAT payload. Returns
// (nil, false) on short input.
func (Service) DecodeStat(raw []byte) (*match.PlayerStat, bool) {
	if len(raw) < 42 {
		return nil, false
	}

	s := &match.PlayerStat{}
	s.Time = binary.LittleEndian.Uint32(raw[0:4])
	s.PC = raw[4]
	s.PlayerID = binary.LittleEndian.Uint32(raw[5:9])
	s.Status = raw[9]
	s.Scores = uint32(binary.LittleEndian.Uint16(raw[10:12]))
	s.Population = uint32(binary.LittleEndian.Uint16(raw[12:14]))
	s.Wood = binary.LittleEndian.Uint32(raw[14:18])
	s.Gold = binary.LittleEndian.Uint32(raw[18:22])
	s.Stone = binary.LittleEndian.Uint32(raw[22:26])
	s.Food = binary.LittleEndian.Uint32(raw[26:30])
	s.Iron = binary.LittleEndian.Uint32(raw[30:34])
	s.Coal = binary.LittleEndian.Uint32(raw[34:38])
	s.Peasants = uint32(binary.LittleEndian.Uint16(raw[38:40]))
	s.Units = uint32(binary.LittleEndian.Uint16(raw[40:42]))

	return s, true
}

// OldStat returns the previous STAT for delta computation. If the player
// has no recorded stat yet, a synthetic stat is built from cur with Time
// zeroed.
func (Service) OldStat(prev, cur *match.PlayerStat) *match.PlayerStat {
	if prev != nil {
		p := *prev
		return &p
	}

	return &match.PlayerStat{
		Time:        0,
		PC:          cur.PC,
		PlayerID:    cur.PlayerID,
		Status:      cur.Status,
		Scores:      cur.Scores,
		Population:  cur.Population,
		Wood:        cur.Wood,
		Gold:        cur.Gold,
		Stone:       cur.Stone,
		Food:        cur.Food,
		Iron:        cur.Iron,
		Coal:        cur.Coal,
		Peasants:    cur.Peasants,
		Units:       cur.Units,
		Population2: cur.Units + cur.Peasants,
		Casuality:   0,
	}
}

// AbsI64 returns |v|. Float-roundtrip.
func (Service) AbsI64(v int64) int64 { return int64(math.Abs(float64(v))) }

// DiffU32 returns cur-prev as a signed delta.
func (Service) DiffU32(cur, prev uint32) int64 { return int64(cur) - int64(prev) }

// EndgameResult maps the numeric "result" code to a string.
func (Service) EndgameResult(code int) string {
	switch code {
	case 1:
		return "loose"
	case 2:
		return "win"
	case 5:
		return "disconnect"
	}

	return fmt.Sprintf("?%d?", code)
}
