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

package commands

import (
	"context"
	"encoding/binary"
	"fmt"
	"strconv"
	"strings"

	"github.com/ldmonster/cossacks-game-server/internal/domain/lobby"
	"github.com/ldmonster/cossacks-game-server/internal/port"
	"github.com/ldmonster/cossacks-game-server/internal/transport/gsc"
	tconn "github.com/ldmonster/cossacks-game-server/internal/transport/tconn"
)

// GETTBLRoomLookup is the narrow consumer port used by GETTBL to query
// the room store without depending on the entire adapter type.
type GETTBLRoomLookup interface {
	GetRoomBySum(sum uint32) *lobby.Room
	AllRooms() []*lobby.Room
}

// GETTBL implements the room-table delta command. The renderer-side
// behaviour (LW_dtbl + LW_tbl) is identical to controller_gettbl.go;
// this command captures the same logic with explicit dependencies so it
// can be unit-tested in isolation.
type GETTBL struct {
	Rooms            GETTBLRoomLookup
	ShowStartedRooms bool
}

// Name returns the GSC command name handled by this command.
func (GETTBL) Name() string { return "GETTBL" }

// Handle responds with the LW_dtbl + LW_tbl pair describing changes
// since the client's last view of the room list.
func (g GETTBL) Handle(
	_ context.Context,
	conn *tconn.Connection,
	req *gsc.Stream,
	args []string,
) port.HandleResult {
	if len(args) < 3 {
		return port.HandleResult{
			Commands:    []gsc.Command{},
			HasResponse: true,
			Err:         ErrGETTBLBadArgs{},
		}
	}

	name := strings.Trim(args[0], "\x00")
	requestedSums := unpackU32LE([]byte(args[2]))

	requested := map[uint32]bool{}
	for _, sum := range requestedSums {
		requested[sum] = true
	}

	hideStarted := (conn.Session == nil || !conn.Session.Dev) && !g.ShowStartedRooms
	dtbl := make([]uint32, 0)
	tblRows := make([][]string, 0)

	for _, sum := range requestedSums {
		room := g.Rooms.GetRoomBySum(sum)
		if room == nil || (hideStarted && room.Started) {
			dtbl = append(dtbl, sum)
		}
	}

	for _, room := range g.Rooms.AllRooms() {
		if hideStarted && room.Started {
			continue
		}

		if strings.EqualFold(name, "ROOMS_V"+strconv.Itoa(int(req.Ver))) &&
			!requested[room.CtlSum] {
			tblRows = append(tblRows, room.Row)
		}
	}

	dtblBin := packU32LE(dtbl)

	tblArgs := []string{name + "\x00", fmt.Sprintf("%d", len(tblRows))}
	for _, row := range tblRows {
		tblArgs = append(tblArgs, row...)
	}

	cmds := []gsc.Command{
		{Name: "LW_dtbl", Args: []string{name + "\x00", string(dtblBin)}},
		{Name: "LW_tbl", Args: tblArgs},
	}

	return port.HandleResult{Commands: cmds, HasResponse: true}
}

// ErrGETTBLBadArgs is returned when the command is invoked with fewer
// than three arguments.
type ErrGETTBLBadArgs struct{}

// Error implements the error interface.
func (ErrGETTBLBadArgs) Error() string {
	return "handler: GETTBL: insufficient args"
}

func unpackU32LE(b []byte) []uint32 {
	if len(b) < 4 {
		return nil
	}

	n := len(b) / 4

	out := make([]uint32, 0, n)
	for i := 0; i+4 <= len(b); i += 4 {
		out = append(out, binary.LittleEndian.Uint32(b[i:i+4]))
	}

	return out
}

func packU32LE(v []uint32) []byte {
	b := make([]byte, len(v)*4)
	for i := range v {
		binary.LittleEndian.PutUint32(b[i*4:(i+1)*4], v[i])
	}

	return b
}
