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

	"github.com/ldmonster/cossacks-game-server/internal/port"
	"github.com/ldmonster/cossacks-game-server/internal/transport/gsc"
	tconn "github.com/ldmonster/cossacks-game-server/internal/transport/tconn"
)

// AliveDriver is the narrow consumer port the Alive command depends on.
// It is satisfied by *handler.Controller (today) and will be satisfied
// by an alive-timer service.
type AliveDriver interface {
	// RefreshAlive updates the player's last-seen timestamp and (re)arms
	// the keep-alive timer that detects abandoned connections.
	RefreshAlive(conn *tconn.Connection)
}

// Alive implements the GSC "alive" command. The client periodically
// pings the server so it knows the connection is still healthy; if the
// ping fails to arrive within the connectivity timeout, the alive
// timer fires and the player is removed from any room.
type Alive struct {
	Driver AliveDriver
}

// Name implements gsc.CommandHandler.
func (Alive) Name() string { return "alive" }

// Handle implements gsc.CommandHandler.
func (a Alive) Handle(
	_ context.Context,
	conn *tconn.Connection,
	_ *gsc.Stream,
	_ []string,
) port.HandleResult {
	a.Driver.RefreshAlive(conn)

	return port.HandleResult{HasResponse: false}
}
