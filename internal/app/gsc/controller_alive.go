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

package gsc

import (
	"time"

	tconn "github.com/ldmonster/cossacks-game-server/internal/transport/tconn"
)

// RefreshAlive marks the connection's session as alive and (re)arms
// the per-player alive timer. Exported so the application-layer Alive
// command (internal/app/gsc/commands.Alive) can drive it without
// reaching into Controller internals.
func (c *Controller) RefreshAlive(conn *tconn.Connection) {
	c.refreshAlive(conn)
}

func (c *Controller) refreshAlive(conn *tconn.Connection) {
	s := ensureSession(conn)
	s.AliveAt = time.Now().UTC()

	id := uint32(s.PlayerID)
	if id == 0 {
		return
	}

	c.armAliveTimer(id)
}

func (c *Controller) armAliveTimer(playerID uint32) {
	c.session.ArmTimer(playerID, func() { c.notAlive(playerID) })
}

// ArmAliveTimer is the exported variant used by command handlers in
// internal/app/gsc/commands. It delegates to the private armAliveTimer.
func (c *Controller) ArmAliveTimer(playerID uint32) {
	c.armAliveTimer(playerID)
}

func (c *Controller) clearAliveTimer(playerID uint32) {
	c.session.ClearTimer(playerID)
}

func (c *Controller) notAlive(playerID uint32) {
	c.leaveRoomByID(playerID)
	c.session.Unregister(playerID)
}
