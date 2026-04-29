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
	"context"

	"github.com/ldmonster/cossacks-game-server/internal/transport/gsc"
	tconn "github.com/ldmonster/cossacks-game-server/internal/transport/tconn"
)

func (c *Controller) tryEnter(
	ctx context.Context, //nolint:unparam // tests pass nil; runtime dispatch supplies a real context
	conn *tconn.Connection,
	req *gsc.Stream,
	p map[string]string,
) ([]gsc.Command, error) {
	return c.routes.TryEnterImpl(ctx, conn, req, p)
}

func (c *Controller) postAccountAction(
	conn *tconn.Connection,
	action string,
	payload map[string]any,
) {
	if conn.Session == nil || conn.Session.Account == nil {
		return
	}

	acc := conn.Session.Account
	c.auth.PostAccountAction(
		context.Background(),
		string(acc.Type),
		acc.ID,
		conn.IP,
		action,
		payload,
	)
}

// PostAccountAction is the exported variant used by command handlers in
// internal/app/gsc/commands. It delegates to the private
// postAccountAction so that Controller satisfies the
// StartAccountPoster port without leaking lower-case methods.
func (c *Controller) PostAccountAction(
	conn *tconn.Connection,
	action string,
	payload map[string]any,
) {
	c.postAccountAction(conn, action, payload)
}
