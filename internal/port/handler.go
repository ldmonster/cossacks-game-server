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

package port

import (
	"context"

	"github.com/ldmonster/cossacks-game-server/internal/transport/gsc"
	tconn "github.com/ldmonster/cossacks-game-server/internal/transport/tconn"
)

// RequestHandler is the minimal contract the TCP server loop requires
// from the application-layer handler.
type RequestHandler interface {
	// HandleWithMeta dispatches a single GSC command and returns a result
	// indicating whether a response should be written back.
	HandleWithMeta(
		ctx context.Context,
		conn *tconn.Connection,
		req *gsc.Stream,
		cmdName string,
		args []string,
		win, key string,
	) HandleResult

	// OnDisconnect is called after the read loop terminates for conn.
	OnDisconnect(conn *tconn.Connection)
}

// HandleResult carries the output of a single command dispatch.
// Mirrors handler.HandleResult so the server loop stays decoupled from
// the handler package.
type HandleResult struct {
	Commands    []gsc.Command
	HasResponse bool
	Err         error
}
