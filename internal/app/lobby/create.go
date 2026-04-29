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

package lobby

import (
	"fmt"
	"time"

	"github.com/ldmonster/cossacks-game-server/internal/domain/lobby"
	"github.com/ldmonster/cossacks-game-server/internal/domain/player"
)

// CreateParams carries the fields needed to register a brand-new room
// when the caller has already prepared the GSC `Row` payload.
type CreateParams struct {
	Host        *player.Player
	HostAddr    string
	HostAddrInt uint32
	Ver         uint8
	Title       string
	Password    string
	MaxPlayers  int
	Level       int
	Row         []string
}

// Create allocates a new RoomID, builds a lobby.Room from the
// supplied parameters, and indexes it across the repository.
//
// The caller is responsible for ensuring the host is no longer present
// in any other room (typically by calling Leave first).
func (s *Service) Create(p CreateParams) *lobby.Room {
	if s == nil || s.repo == nil {
		return nil
	}

	id := s.repo.NextRoomID()

	hostID := uint32(0)
	if p.Host != nil {
		hostID = p.Host.ID
	}

	r := &lobby.Room{
		ID:           id,
		Title:        p.Title,
		HostID:       hostID,
		HostAddr:     p.HostAddr,
		HostAddrInt:  p.HostAddrInt,
		Ver:          p.Ver,
		Level:        p.Level,
		Password:     p.Password,
		MaxPlayers:   p.MaxPlayers,
		PlayersCount: 1,
		Players:      map[uint32]*player.Player{hostID: p.Host},
		PlayersTime:  map[uint32]time.Time{hostID: time.Now()},
		Row:          p.Row,
		CtlSum:       controlSum(p.Row),
		Ctime:        time.Now(),
	}
	s.repo.IndexByID(r)
	s.repo.IndexByHost(player.PlayerID(hostID), r)
	s.repo.IndexBySum(r)

	return r
}

// RegisterNewParams carries everything RegisterNew needs to allocate,
// populate and index a fresh room from a GSC `regNewRoom` request
type RegisterNewParams struct {
	Host        *player.Player
	HostAddr    string
	HostAddrInt uint32
	Ver         uint8
	Title       string
	Password    string
	MaxPlayers  int
	Level       int
	LevelLabel  string
	LockMark    string
	NickStr     string
	VEType      string
	IsAC        bool
}

// RegisterNew allocates a fresh RoomID, builds the GSC `Row` payload,
// constructs and indexes the resulting lobby.Room. The returned
// room is fully wired into the repository (by-id, by-host, by-sum).
func (s *Service) RegisterNew(p RegisterNewParams) *lobby.Room {
	if s == nil || s.repo == nil {
		return nil
	}

	id := s.repo.NextRoomID()

	hostID := uint32(0)
	if p.Host != nil {
		hostID = p.Host.ID
	}

	row := []string{fmt.Sprintf("%d", id), p.LockMark, p.Title, p.NickStr}
	if p.IsAC {
		row = append(row, p.VEType)
	}

	row = append(row,
		p.LevelLabel,
		fmt.Sprintf("%d/%d", 1, p.MaxPlayers),
		fmt.Sprintf("%d", p.Ver),
		fmt.Sprintf("%d", p.HostAddrInt),
		fmt.Sprintf("0%X", 0xFFFFFFFF-id),
	)

	r := &lobby.Room{
		ID:           id,
		Title:        p.Title,
		HostID:       hostID,
		HostAddr:     p.HostAddr,
		HostAddrInt:  p.HostAddrInt,
		Ver:          p.Ver,
		Level:        p.Level,
		Password:     p.Password,
		MaxPlayers:   p.MaxPlayers,
		PlayersCount: 1,
		Players:      map[uint32]*player.Player{hostID: p.Host},
		PlayersTime:  map[uint32]time.Time{hostID: time.Now()},
		Row:          row,
		CtlSum:       controlSum(row),
		Ctime:        time.Now(),
	}
	s.repo.IndexByID(r)
	s.repo.IndexByHost(player.PlayerID(hostID), r)
	s.repo.IndexBySum(r)

	return r
}
