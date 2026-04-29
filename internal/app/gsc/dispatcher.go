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
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/ldmonster/cossacks-game-server/internal/adapter/rooms"
	"github.com/ldmonster/cossacks-game-server/internal/app/connectivity"
	gsccmds "github.com/ldmonster/cossacks-game-server/internal/app/gsc/commands"
	gscroutes "github.com/ldmonster/cossacks-game-server/internal/app/gsc/routes"
	"github.com/ldmonster/cossacks-game-server/internal/app/identity"
	"github.com/ldmonster/cossacks-game-server/internal/app/lobby"
	sgame "github.com/ldmonster/cossacks-game-server/internal/app/match"
	"github.com/ldmonster/cossacks-game-server/internal/app/ranking"
	"github.com/ldmonster/cossacks-game-server/internal/platform/config"
	"github.com/ldmonster/cossacks-game-server/internal/port"
	"github.com/ldmonster/cossacks-game-server/internal/render"
	"github.com/ldmonster/cossacks-game-server/internal/transport/gsc"
	tconn "github.com/ldmonster/cossacks-game-server/internal/transport/tconn"
)

// Controller dispatches incoming GSC commands and owns the top-level
// server rooms. All mutable state is guarded by stateMu.
type Controller struct {
	// Narrow config slices — the Controller only reads what it needs.
	Game   config.GameConfig
	Server config.ServerConfig
	Auth   config.AuthConfig

	Store   *rooms.Store
	Storage port.KVStore
	Log     *zap.Logger

	Renderer *render.TemplateRenderer

	ranking port.RankingService
	auth    *identity.Service
	rooms   *lobby.Service
	session *connectivity.Manager
	game    port.GameService
	routes  *gscroutes.Routes

	cmds *Registry
}

// HandleResult is a type alias for port.HandleResult so existing
// call sites that reference handler.HandleResult keep compiling.
type HandleResult = port.HandleResult

// NewController constructs a Controller with all runtime services and
// maps already initialised.
func NewController(
	cfg *config.Config,
	store *rooms.Store,
	storage port.KVStore,
	log *zap.Logger,
) *Controller {
	c := &Controller{
		Game:     cfg.Game,
		Server:   cfg.Server,
		Auth:     cfg.AuthConfig(),
		Store:    store,
		Storage:  storage,
		Log:      log,
		Renderer: render.NewTemplateRenderer(cfg.Game.Templates),
		ranking:  ranking.NewCache(),
		auth:     identity.NewService(cfg.AuthConfig(), nil, log),
		rooms:    lobby.NewService(store.AsRoomRepo()),
		session:  connectivity.NewManager(150 * time.Second),
		game:     sgame.NewService(),
	}
	c.routes = gscroutes.New(gscroutes.Deps{
		Renderer: c.Renderer,
		Auth:     cfg.AuthConfig(),
		Game:     &c.Game,
		Server:   &c.Server,
		Log:      log,
		Ranking:  controllerRankingAdapter{c: c},
		Players:  store,
		Rooms:    controllerRoomLifecycleAdapter{c: c},
		Lobby:    c.rooms,
		Storage:  storage,
		Identity: c.auth,
		Sessions: c.session,
	})

	c.cmds = defaultCommandRegistry(commandDeps{
		ProxyKey:         cfg.Game.ProxyKey,
		Alive:            c.session,
		Lobby:            c.rooms,
		AliveDriver:      c,
		StartAlive:       c,
		Match:            c.game,
		StartMatch:       c.game,
		Players:          store,
		StartPlayers:     store,
		StartRooms:       store,
		StartLobby:       c.rooms,
		StartAccount:     c,
		RoomsLookup:      store,
		GETTBLRooms:      store,
		Routes:           c,
		ShowStartedRooms: cfg.Game.ShowStartedRooms,
		Log:              log,
	})

	return c
}

// commandDeps bundles the dependencies needed to wire the migrated
// commands into a Registry. Adding a new command should add a field
// here and a Register call in defaultCommandRegistry, never edit the
// signature.
type commandDeps struct {
	ProxyKey         string
	Alive            *connectivity.Manager
	Lobby            *lobby.Service
	AliveDriver      gsccmds.AliveDriver
	StartAlive       gsccmds.StartAliveDriver
	Match            gsccmds.EndgameParser
	StartMatch       gsccmds.StartMatch
	Players          gsccmds.EndgamePlayerLookup
	StartPlayers     gsccmds.StartPlayerLookup
	StartRooms       gsccmds.StartRoomLookup
	StartLobby       gsccmds.StartLobby
	StartAccount     gsccmds.StartAccountPoster
	RoomsLookup      gsccmds.EndgameRoomLookup
	GETTBLRooms      gsccmds.GETTBLRoomLookup
	Routes           gsccmds.RouteDispatcher
	ShowStartedRooms bool
	Log              *zap.Logger
}

// defaultCommandRegistry returns the registry used by Controller for
// commands that have been migrated to internal/app/gsc/commands. It is
// shared between NewController and the test helper to keep the two
// construction paths in sync.
func defaultCommandRegistry(d commandDeps) *Registry {
	r := NewRegistry()
	r.Register(gsccmds.Login{})
	r.Register(gsccmds.Echo{})
	r.Register(gsccmds.URL{})
	r.Register(gsccmds.Proxy{Key: d.ProxyKey})
	r.Register(gsccmds.Leave{Alive: d.Alive, Lobby: d.Lobby})
	r.Register(gsccmds.Alive{Driver: d.AliveDriver})
	r.Register(
		gsccmds.Endgame{Match: d.Match, Players: d.Players, Rooms: d.RoomsLookup, Log: d.Log},
	)
	r.Register(gsccmds.GETTBL{Rooms: d.GETTBLRooms, ShowStartedRooms: d.ShowStartedRooms})
	r.Register(gsccmds.Start{
		Rooms:   d.StartRooms,
		Lobby:   d.StartLobby,
		Alive:   d.StartAlive,
		Match:   d.StartMatch,
		Players: d.StartPlayers,
		Account: d.StartAccount,
	})
	r.Register(gsccmds.Open{Routes: d.Routes, Log: d.Log})
	r.Register(gsccmds.Go{Routes: d.Routes, Log: d.Log})

	return r
}

func (c *Controller) HandleWithMeta(
	ctx context.Context,
	conn *tconn.Connection,
	req *gsc.Stream,
	cmdName string,
	args []string,
	win string,
	key string,
) HandleResult {
	// "stats" historically acquired a per-room lock; every other
	// command used to share a coarse Controller-level mutex
	// (stateMu). Both have been removed: state
	// mutations are now protected by the locks inside
	// adapter/rooms.Store and app/lobby.Repository, so per-command
	// concurrency control is the responsibility of the command
	// implementations themselves.
	if cmdName == "stats" {
		err := c.handleStatsLocked(conn, args)
		return HandleResult{HasResponse: false, Err: err}
	}

	_ = key
	_ = win

	// Registry-backed commands: every command that has been
	// extracted into internal/app/gsc/commands/ is dispatched here. The
	// switch below only lists commands that still depend on Controller
	// internals and are awaiting extraction.
	if h, ok := c.cmds.Lookup(cmdName); ok {
		return h.Handle(ctx, conn, req, args)
	}

	if isUnimplementedCommand(cmdName) {
		// Documented see unimplemented.go.
		return HandleResult{HasResponse: false}
	}

	c.Log.Warn("unknown command", zap.String("command", cmdName))
	// Default behaviour: push an empty response. Surface the
	// condition via HandleResult.Err so future dispatch wrappers can
	// distinguish it from a normal empty response.
	return HandleResult{
		Commands:    []gsc.Command{},
		HasResponse: true,
		Err:         errUnknownCommand{name: cmdName},
	}
}

func (c *Controller) handleOpen(
	ctx context.Context,
	conn *tconn.Connection,
	req *gsc.Stream,
	args []string,
) ([]gsc.Command, error) {
	if len(args) < 1 {
		return []gsc.Command{}, nil
	}

	rawURL := strings.TrimSpace(strings.ReplaceAll(args[0], "\x00", ""))
	url := strings.TrimSuffix(rawURL, ".dcml")

	params := map[string]string{}
	if len(args) > 1 {
		params = parseOpenParams(strings.ReplaceAll(args[1], "\x00", ""))
	}

	c.Log.Debug("open route",
		zap.Uint64("conn_id", conn.ID),
		zap.String("raw_url", rawURL),
		zap.String("parsed_method", url),
		zap.Any("params", params),
	)

	return c.dispatchOpen(ctx, conn, req, url, params)
}

// dispatchOpen routes a normalised open/go method through the Controller's
// open-routes table (see dispatch_routes.go). The previous inline switch
// has been replaced by a map lookup so adding a new route does not require
// editing the dispatcher core.
func (c *Controller) dispatchOpen(
	ctx context.Context,
	conn *tconn.Connection,
	req *gsc.Stream,
	method string,
	p map[string]string,
) ([]gsc.Command, error) {
	if isUnimplementedOpenRoute(method) {
		// Documented see unimplemented.go.
		return emptyOpenResponse(), nil
	}

	if h, ok := c.openRoutes()[method]; ok {
		return h(ctx, conn, req, p)
	}

	c.Log.Warn("open route not found",
		zap.Uint64("conn_id", conn.ID),
		zap.String("method", method),
		zap.Any("params", p),
	)

	return c.renderAlert(req.Ver, "Error", "Page Not Found"),
		errUnknownOpenRoute{method: method}
}

// DispatchOpen is the exported entry point used by command handlers in
// internal/app/gsc/commands. It delegates to dispatchOpen so the
// Controller satisfies the RouteDispatcher port without exposing
// lower-case methods.
func (c *Controller) DispatchOpen(
	ctx context.Context,
	conn *tconn.Connection,
	req *gsc.Stream,
	method string,
	p map[string]string,
) ([]gsc.Command, error) {
	return c.dispatchOpen(ctx, conn, req, method, p)
}
