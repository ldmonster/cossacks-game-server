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

// Routes table for `open` / `go` dispatch. Replaces the long inline
// switch in `dispatchOpen` so adding a new
// route does not require editing the dispatcher core.

package gsc

import (
	"context"

	"github.com/ldmonster/cossacks-game-server/internal/transport/gsc"
	tconn "github.com/ldmonster/cossacks-game-server/internal/transport/tconn"
)

// openRouteHandler is the uniform signature the routes table uses. All
// per-route methods on Controller adopt this signature (unused
// arguments are simply ignored). The signature returns
// `([]gsc.Command, error)`.
type openRouteHandler func(
	ctx context.Context,
	conn *tconn.Connection,
	req *gsc.Stream,
	p map[string]string,
) ([]gsc.Command, error)

// openRoutes returns the dispatch table keyed by route method name.
// Constructed lazily per-call so each Controller's bound methods are
// captured. Aliases (`startup`/`games`/`rooms_table_dgl`) share a
// single handler entry.
func (c *Controller) openRoutes() map[string]openRouteHandler {
	return map[string]openRouteHandler{
		"enter":     c.routesEnter,
		"try_enter": c.routesTryEnter,
		"startup":   c.routesStartup,
		// `games` and `rooms_table_dgl` share the startup payload.
		"games":                c.routesStartup,
		"rooms_table_dgl":      c.routesStartup,
		"resize":               c.routesResize,
		"new_room_dgl":         c.routesNewRoomDialog,
		"reg_new_room":         c.routesRegNewRoom,
		"join_game":            c.routesJoinGame,
		"room_info_dgl":        c.routesRoomInfo,
		"join_pl_cmd":          c.routesJoinPlayer,
		"user_details":         c.routesUserDetails,
		"users_list":           c.routesUsersList,
		"tournaments":          c.routesTournaments,
		"lcn_registration_dgl": c.routesLCNRegistrationDialog,
		"gg_cup_thanks_dgl":    c.routesGGCupThanks,
	}
}

// --- Per-route adapters: keep individual handler signatures intact while
// presenting the uniform openRouteHandler shape to the routes table.

func (c *Controller) routesEnter(
	ctx context.Context, conn *tconn.Connection, req *gsc.Stream, p map[string]string,
) ([]gsc.Command, error) {
	return c.routes.Enter(ctx, conn, req, p)
}

func (c *Controller) routesTryEnter(
	ctx context.Context, conn *tconn.Connection, req *gsc.Stream, p map[string]string,
) ([]gsc.Command, error) {
	return c.routes.TryEnter(ctx, conn, req, p)
}

func (c *Controller) routesStartup(
	ctx context.Context, conn *tconn.Connection, req *gsc.Stream, p map[string]string,
) ([]gsc.Command, error) {
	return c.routes.Startup(ctx, conn, req, p)
}

func (c *Controller) routesResize(
	ctx context.Context, conn *tconn.Connection, req *gsc.Stream, p map[string]string,
) ([]gsc.Command, error) {
	return c.routes.Resize(ctx, conn, req, p)
}

func (c *Controller) routesNewRoomDialog(
	ctx context.Context, conn *tconn.Connection, req *gsc.Stream, p map[string]string,
) ([]gsc.Command, error) {
	return c.routes.NewRoomDialog(ctx, conn, req, p)
}

func (c *Controller) routesRegNewRoom(
	ctx context.Context, conn *tconn.Connection, req *gsc.Stream, p map[string]string,
) ([]gsc.Command, error) {
	return c.routes.RegNewRoom(ctx, conn, req, p)
}

func (c *Controller) routesJoinGame(
	ctx context.Context, conn *tconn.Connection, req *gsc.Stream, p map[string]string,
) ([]gsc.Command, error) {
	return c.routes.JoinGame(ctx, conn, req, p)
}

func (c *Controller) routesRoomInfo(
	ctx context.Context, conn *tconn.Connection, req *gsc.Stream, p map[string]string,
) ([]gsc.Command, error) {
	return c.routes.RoomInfo(ctx, conn, req, p)
}

func (c *Controller) routesJoinPlayer(
	ctx context.Context, conn *tconn.Connection, req *gsc.Stream, p map[string]string,
) ([]gsc.Command, error) {
	return c.routes.JoinPlayer(ctx, conn, req, p)
}

func (c *Controller) routesUserDetails(
	ctx context.Context, conn *tconn.Connection, req *gsc.Stream, p map[string]string,
) ([]gsc.Command, error) {
	return c.routes.UserDetails(ctx, conn, req, p)
}

func (c *Controller) routesUsersList(
	ctx context.Context, conn *tconn.Connection, req *gsc.Stream, p map[string]string,
) ([]gsc.Command, error) {
	return c.routes.UsersList(ctx, conn, req, p)
}

func (c *Controller) routesTournaments(
	ctx context.Context, conn *tconn.Connection, req *gsc.Stream, p map[string]string,
) ([]gsc.Command, error) {
	return c.routes.Tournaments(ctx, conn, req, p)
}

func (c *Controller) routesLCNRegistrationDialog(
	ctx context.Context, conn *tconn.Connection, req *gsc.Stream, p map[string]string,
) ([]gsc.Command, error) {
	return c.routes.LCNRegistrationDialog(ctx, conn, req, p)
}

func (c *Controller) routesGGCupThanks(
	ctx context.Context, conn *tconn.Connection, req *gsc.Stream, p map[string]string,
) ([]gsc.Command, error) {
	return c.routes.GGCupThanks(ctx, conn, req, p)
}
