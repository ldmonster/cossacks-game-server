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

// Package match holds in-match domain types: stats, endgame, rejections.
package match

// StatRejection enumerates the reasons UpdateStat may decline to apply
// a STAT payload. The zero value (StatApplied) means the stat was applied.
type StatRejection int

const (
	StatApplied              StatRejection = 0
	StatRejectShortBuffer    StatRejection = 1
	StatRejectPlayerMismatch StatRejection = 2
	StatRejectTickAhead      StatRejection = 3
)

// EndgameEvent is the structured form of the GSC endgame payload after
// numeric/text mapping.
type EndgameEvent struct {
	GameID   int
	PlayerID uint32
	Result   string
	Nick     string
	Own      string
	Title    string
}

// PlayerStat is the per-tick game stats sample reported by the client.
type PlayerStat struct {
	Time        uint32
	PC          uint8
	PlayerID    uint32
	Status      uint8
	Scores      uint32
	RealScores  int64
	Population  uint32
	Wood        uint32
	Gold        uint32
	Stone       uint32
	Food        uint32
	Iron        uint32
	Coal        uint32
	Peasants    uint32
	Units       uint32
	Population2 uint32
	Casuality   int64
	ChangeWood  float64
	ChangeStone float64
	ChangeFood  float64
	ChangeGold  float64
	ChangeIron  float64
	ChangeCoal  float64
	ChangePeas  float64
	ChangeUnits float64
	ChangePop2  float64
}

// PlayerStatCycle accumulates per-cycle increments between samples.
type PlayerStatCycle struct {
	Peasants uint32
	Units    uint32
	Scores   int64
}

// StatHistoryPoint is one entry of a stat's rolling history series.
type StatHistoryPoint struct {
	Change   float64
	Time     uint32
	Interval uint32
}
