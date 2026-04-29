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

// GETTBL room-table delta handler — thin wrapper that delegates to the
// internal/app/gsc/commands.GETTBL implementation. Kept around for
// backwards compatibility with existing handler tests; will be removed
// in a future cleanup.

package gsc

import (
	"context"

	gsccmds "github.com/ldmonster/cossacks-game-server/internal/app/gsc/commands"
	"github.com/ldmonster/cossacks-game-server/internal/transport/gsc"
	tconn "github.com/ldmonster/cossacks-game-server/internal/transport/tconn"
)

func (c *Controller) handleGETTBL(
	conn *tconn.Connection,
	req *gsc.Stream,
	args []string,
) ([]gsc.Command, error) {
	cmd := gsccmds.GETTBL{Rooms: c.Store, ShowStartedRooms: c.Game.ShowStartedRooms}
	res := cmd.Handle(context.Background(), conn, req, args)

	return res.Commands, res.Err
}
