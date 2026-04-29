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

import "github.com/ldmonster/cossacks-game-server/internal/domain/lobby"

// ApplyStartedUsers parses the host-side `start` payload tail
// (count, then 4-tuples of player_id/nation/theam/color) and assigns
// the parsed values onto the matching room.Players entries while
// rebuilding room.StartedUsers in declaration order.
//
// args is the full GSC arg slice received by the start handler:
// args[0]=sav, args[1]=map, args[2]=count, args[3..]=tuples.
func (s Service) ApplyStartedUsers(room *lobby.Room, args []string) {
	if room == nil || len(args) < 3 {
		return
	}

	count, err := IntArg(args[2])
	if err != nil || count <= 0 {
		return
	}

	room.StartedUsers = nil
	list := args[3:]

	for i := 0; i+3 < len(list) && len(room.StartedUsers) < count; i += 4 {
		pid, err1 := IntArg(list[i])
		nation, err2 := IntArg(list[i+1])
		theam, err3 := IntArg(list[i+2])

		color, err4 := IntArg(list[i+3])
		if err1 != nil || err2 != nil || err3 != nil || err4 != nil {
			continue
		}

		pl := room.Players[uint32(int32(pid))]
		if pl == nil {
			continue
		}

		pl.Nation = uint32(nation)
		pl.Theam = uint32(theam)
		pl.Color = uint32(color)
		room.StartedUsers = append(room.StartedUsers, pl)
	}
}
