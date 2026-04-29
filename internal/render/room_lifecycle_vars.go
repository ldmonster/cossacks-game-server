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

// Template-variable builders for room registration / join responses.

package render

import "strconv"

// BuildRegNewRoomVars assembles the variable map rendered into
// reg_new_room.tmpl after a successful host-side room creation.
func BuildRegNewRoomVars(
	playerID uint32,
	holePort int,
	holeHost string,
	holeInt int,
	gameID, title string,
	maxPlayers int,
) map[string]string {
	return map[string]string{
		"player_id": strconv.FormatUint(uint64(playerID), 10),
		"hole_port": strconv.Itoa(holePort),
		"hole_host": holeHost,
		"hole_int":  strconv.Itoa(holeInt),
		"id":        gameID,
		"name":      title,
		"max_pl":    strconv.Itoa(maxPlayers),
	}
}

// BuildJoinRoomVars assembles the variable map rendered into
// join_room.tmpl after a successful client-side room join.
func BuildJoinRoomVars(
	roomID uint32,
	maxPlayers int,
	title, ip string,
	port int,
) map[string]string {
	return map[string]string{
		"id":     strconv.FormatUint(uint64(roomID), 10),
		"max_pl": strconv.Itoa(maxPlayers),
		"name":   title,
		"ip":     ip,
		"port":   strconv.Itoa(port),
	}
}
