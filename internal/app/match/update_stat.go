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

package match

import (
	"github.com/ldmonster/cossacks-game-server/internal/domain/lobby"
	"github.com/ldmonster/cossacks-game-server/internal/domain/match"
	"github.com/ldmonster/cossacks-game-server/internal/domain/player"
)

// UpdateStat decodes raw, validates it against the player's owning
// room/state, and applies the per-tick cycle and rolling-change math
//
// The caller must have validated that player belongs to room. The
// function returns match.StatApplied on success.
func (s Service) UpdateStat(
	player *player.Player,
	room *lobby.Room,
	raw []byte,
) match.StatRejection {
	if player == nil || room == nil {
		return match.StatRejectShortBuffer
	}

	stat, ok := s.DecodeStat(raw)
	if !ok {
		return match.StatRejectShortBuffer
	}

	if stat.PlayerID != player.ID {
		return match.StatRejectPlayerMismatch
	}

	if room.TimeTick < stat.Time {
		room.TimeTick = stat.Time
	}

	if player.TimeTick > stat.Time {
		return match.StatRejectTickAhead
	}

	if player.StatCycle.Peasants == 0 && player.StatCycle.Units == 0 &&
		player.StatCycle.Scores == 0 {
		player.StatCycle = match.PlayerStatCycle{}
	}

	old := s.OldStat(player.Stat, stat)

	// Zombie / team-color inheritance: when the player's first STAT comes
	// in with zeroed scores+population, copy color from a same-team
	// started peer.
	if player.Stat == nil && stat.Scores == 0 && stat.Population == 0 {
		for _, started := range room.StartedUsers {
			if started != nil && started.ID != player.ID && started.Theam == player.Theam {
				player.Zombie = true
				player.Color = started.Color

				break
			}
		}
	}

	if stat.Peasants < old.Peasants-player.StatCycle.Peasants*0x10000 {
		player.StatCycle.Peasants++
	}

	stat.Peasants += player.StatCycle.Peasants * 0x10000

	if stat.Units < old.Units-player.StatCycle.Units*0x10000 {
		player.StatCycle.Units++
	}

	stat.Units += player.StatCycle.Units * 0x10000

	scoresChange := int64(stat.Scores) - int64(old.Scores)
	if s.AbsI64(scoresChange) > 0x7FFF {
		if scoresChange > 0 {
			player.StatCycle.Scores--
		} else {
			player.StatCycle.Scores++
		}
	}

	stat.RealScores = player.StatCycle.Scores*0x10000 + int64(stat.Scores)
	stat.Population2 = stat.Units + stat.Peasants

	interval := stat.Time - player.TimeTick
	if interval == 0 {
		interval = 1
	}

	intervalF := float64(interval)
	stat.ChangeGold = (float64(s.DiffU32(stat.Gold, old.Gold)) / intervalF) * 25 / 2
	stat.ChangeIron = (float64(s.DiffU32(stat.Iron, old.Iron)) / intervalF) * 25 / 2
	stat.ChangeCoal = (float64(s.DiffU32(stat.Coal, old.Coal)) / intervalF) * 25 / 2

	intervals := map[string]uint32{
		"wood":        60 * 25,
		"stone":       60 * 25,
		"food":        120 * 25,
		"peasants":    600,
		"units":       1000,
		"population2": 1000,
	}
	coefs := map[string]float64{
		"wood":        25.0 / 2.0,
		"stone":       25.0 / 2.0,
		"food":        25.0 / 2.0,
		"peasants":    200,
		"units":       50,
		"population2": 50,
	}

	if player.StatHistory == nil {
		player.StatHistory = map[string][]match.StatHistoryPoint{}
	}

	if player.StatSum == nil {
		player.StatSum = map[string]float64{}
	}

	updateRolling := func(name string, cur, prev uint32) float64 {
		key := "change_" + name
		change := float64(s.DiffU32(cur, prev))
		player.StatHistory[key] = append(player.StatHistory[key], match.StatHistoryPoint{
			Change:   change,
			Time:     stat.Time,
			Interval: interval,
		})
		player.StatSum["sum_"+name] += change

		cutoff := stat.Time - intervals[name]
		for len(player.StatHistory[key]) > 0 && player.StatHistory[key][0].Time < cutoff {
			player.StatSum["sum_"+name] -= player.StatHistory[key][0].Change
			player.StatHistory[key] = player.StatHistory[key][1:]
		}

		if len(player.StatHistory[key]) == 0 {
			return 0
		}

		first := player.StatHistory[key][0]

		denom := float64(stat.Time - (first.Time - first.Interval))
		if denom <= 0 {
			denom = 1
		}

		return player.StatSum["sum_"+name] / denom * coefs[name]
	}

	stat.ChangeWood = updateRolling("wood", stat.Wood, old.Wood)
	stat.ChangeFood = updateRolling("food", stat.Food, old.Food)
	stat.ChangeStone = updateRolling("stone", stat.Stone, old.Stone)
	stat.ChangePeas = updateRolling("peasants", stat.Peasants, old.Peasants)
	stat.ChangeUnits = updateRolling("units", stat.Units, old.Units)
	stat.ChangePop2 = updateRolling("population2", stat.Population2, old.Population2)

	casualityChange := int64(stat.Population2-old.Population2) -
		int64(stat.Population-old.Population)
	stat.Casuality = old.Casuality + casualityChange

	player.TimeTick = stat.Time
	player.Stat = stat

	return match.StatApplied
}
