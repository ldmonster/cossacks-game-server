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

// Package routes contains the open/go route handlers extracted from
// internal/server/handler. Each route is a method on Routes that takes
// the standard (ctx, conn, req, params) tuple and returns the GSC
// command list plus an error for observability.
//
// Routes are added incrementally; the Controller's openRoutes() table
// merges handler-side methods (still on Controller) with methods on
// Routes for the open/go route handlers. The route registry tracks
// the migration.
package routes

import (
	"context"

	"go.uber.org/zap"

	"github.com/ldmonster/cossacks-game-server/internal/platform/config"
	"github.com/ldmonster/cossacks-game-server/internal/port"
	"github.com/ldmonster/cossacks-game-server/internal/render"
	"github.com/ldmonster/cossacks-game-server/internal/transport/gsc"
	tconn "github.com/ldmonster/cossacks-game-server/internal/transport/tconn"
)

// Deps bundles the dependencies shared by every route handler.
type Deps struct {
	Renderer *render.TemplateRenderer
	Auth     config.AuthConfig
	Game     *config.GameConfig
	Server   *config.ServerConfig
	Log      *zap.Logger
	Ranking  RankingProvider
	Players  PlayerLookup
	Rooms    RoomLifecycle
	Lobby    LobbyRegistrar
	Storage  port.KVStore
	Identity IdentityService
	Sessions SessionRegistry
}

// Routes is the receiver type for migrated open/go route handlers. It
// is constructed once per Controller and lives for the controller's
// lifetime; methods are safe for concurrent use as long as the
// underlying Renderer and Auth config are not mutated after
// construction.
type Routes struct {
	deps Deps
}

// New returns a Routes value initialised with the supplied
// dependencies.
func New(deps Deps) *Routes {
	return &Routes{deps: deps}
}

// Handler is the uniform signature exposed to the route registry. It
// matches handler.openRouteHandler so the two implementations can be
// merged into a single dispatch table during the migration.
type Handler func(
	ctx context.Context,
	conn *tconn.Connection,
	req *gsc.Stream,
	p map[string]string,
) ([]gsc.Command, error)

// render is a thin wrapper that mirrors Controller.render. It uses the
// package defaults when the Routes value was constructed without a
// renderer (only relevant in struct-literal tests).
func (r *Routes) render(ver uint8, name string, vars map[string]string) string {
	if r != nil && r.deps.Renderer != nil {
		return r.deps.Renderer.Render(ver, name, vars)
	}

	return render.LoadShowBodyFromRoots(render.DefaultTemplateRoots, ver, name, vars)
}

// renderAlert mirrors Controller.renderAlert for the migrated routes.
func (r *Routes) renderAlert(ver uint8, header, text string) []gsc.Command {
	return render.Show(r.render(ver, "alert_dgl.tmpl", map[string]string{
		"header": header,
		"text":   text,
	}))
}
