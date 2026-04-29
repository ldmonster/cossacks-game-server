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

// ParseEndgame builds an match.EndgameEvent from raw GSC args. Returns
// (match.EndgameEvent{}, false) when the payload is malformed.
//
// connectedPlayerID is the live connection's owning player id (used
// to derive the "own" tag for the host); pass 0 when unknown.
func (Service) ParseEndgame(
	args []string,
	connectedPlayerID uint32,
	getPlayer func(uint32) *player.Player,
	getRoom func(uint32) *lobby.Room,
) (match.EndgameEvent, bool) {
	if len(args) < 3 {
		return match.EndgameEvent{}, false
	}

	gameID, _ := IntArg(args[0])
	playerIDInt, _ := IntArg(args[1])
	resultCode, _ := IntArg(args[2])

	playerU32 := uint32(int32(playerIDInt))

	nick := "."

	if getPlayer != nil {
		if pl := getPlayer(playerU32); pl != nil {
			nick = pl.Nick
		}
	}

	resultStr := Service{}.EndgameResult(resultCode)

	var room *lobby.Room
	if getRoom != nil {
		room = getRoom(uint32(gameID))
	}

	own := ""
	if room != nil && connectedPlayerID != 0 && room.HostID == connectedPlayerID {
		own = "his "
	}

	title := ""
	if room != nil {
		title = " " + room.Title
	}

	return match.EndgameEvent{
		GameID:   gameID,
		PlayerID: playerU32,
		Result:   resultStr,
		Nick:     nick,
		Own:      own,
		Title:    title,
	}, true
}
