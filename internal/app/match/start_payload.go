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
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/ldmonster/cossacks-game-server/internal/domain/lobby"
	"github.com/ldmonster/cossacks-game-server/internal/domain/player"
)

var intArgRe = regexp.MustCompile(`-?\d+`)

// IntArg extracts the first signed integer token from a GSC-style arg
// string (e.g. "game=10", "pid=-1", "0\x00 ").  Mirrors the earlier
// `parseIntArg` previously embedded in handler/controller_helpers.go
// so wire fidelity is preserved.
func IntArg(s string) (int, error) {
	s = strings.TrimRight(strings.TrimSpace(s), "\x00")

	m := intArgRe.FindString(s)
	if m == "" {
		return 0, fmt.Errorf("no int in %q", s)
	}

	return strconv.Atoi(m)
}

var savRegexp = regexp.MustCompile(`^sav:\[(\d+)\]$`)

// BuildStartAccountPayload assembles the JSON-shaped payload posted to
// the configured account endpoint when a host starts a game. Returns nil
// when the inputs are malformed.
//
// `getPlayer` is an optional read-through accessor. When nil, players
// are emitted with `{"id": ..., "lost": true}` placeholders.
func (Service) BuildStartAccountPayload(
	room *lobby.Room,
	args []string,
	sav, mapName string,
	getPlayer func(uint32) *player.Player,
) map[string]any {
	if room == nil || len(args) < 3 {
		return nil
	}

	playersCount, err := IntArg(args[2])
	if err != nil || playersCount <= 0 {
		return nil
	}

	payload := map[string]any{
		"id":            int(room.ID),
		"title":         room.Title,
		"max_players":   room.MaxPlayers,
		"players_count": room.PlayersCount,
		"level":         room.Level,
		"ctime":         room.Ctime.Unix(),
		"map":           mapName,
	}

	if m := savRegexp.FindStringSubmatch(sav); len(m) == 2 {
		if v, err := strconv.Atoi(m[1]); err == nil {
			payload["save_from"] = v
		}
	}

	players := make([]map[string]any, 0, playersCount)

	raw := args[3:]
	for i := 0; i+3 < len(raw) && len(players) < playersCount; i += 4 {
		playerIDInt, err1 := IntArg(raw[i])
		nation, err2 := IntArg(raw[i+1])
		theam, err3 := IntArg(raw[i+2])

		color, err4 := IntArg(raw[i+3])
		if err1 != nil || err2 != nil || err3 != nil || err4 != nil {
			continue
		}

		playerID := uint32(int32(playerIDInt))

		postPlayer := map[string]any{
			"id":     int(playerID),
			"nation": nation,
			"theam":  theam,
			"color":  color,
		}

		var pl *player.Player
		if getPlayer != nil {
			pl = getPlayer(playerID)
		}

		if pl != nil {
			postPlayer["nick"] = pl.Nick

			postPlayer["connected_at"] = pl.ConnectedAt.Unix()
			if pl.AccountType != "" {
				account := map[string]any{
					"type": pl.AccountType,
				}
				if pl.AccountProfile != "" {
					account["profile"] = pl.AccountProfile
				}

				if pl.AccountID != "" {
					account["id"] = pl.AccountID
				}

				postPlayer["account"] = account
			}
		} else {
			postPlayer["lost"] = true
		}

		players = append(players, postPlayer)
	}

	payload["players"] = players

	return payload
}
