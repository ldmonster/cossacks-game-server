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

// Package commands holds the per-command application handlers. Each
// file implements one GSC command via the gsc.CommandHandler contract
// (see internal/app/gsc/registry.go).
package commands

import (
	"context"

	"github.com/ldmonster/cossacks-game-server/internal/port"
	"github.com/ldmonster/cossacks-game-server/internal/render"
	"github.com/ldmonster/cossacks-game-server/internal/transport/gsc"
	tconn "github.com/ldmonster/cossacks-game-server/internal/transport/tconn"
)

// Login renders the earlier "open&enter.dcml" startup page. It carries
// no dependencies.
type Login struct{}

// Name implements gsc.CommandHandler.
func (Login) Name() string { return "login" }

// Handle implements gsc.CommandHandler.
func (Login) Handle(
	_ context.Context,
	_ *tconn.Connection,
	_ *gsc.Stream,
	_ []string,
) port.HandleResult {
	return port.HandleResult{
		Commands:    render.Show(":GW|open&enter.dcml"),
		HasResponse: true,
	}
}

// Echo replies with the earlier LW_echo command list, used by the
// client to round-trip text args for keep-alive purposes.
type Echo struct{}

// Name implements gsc.CommandHandler.
func (Echo) Name() string { return "echo" }

// Handle implements gsc.CommandHandler.
func (Echo) Handle(
	_ context.Context,
	_ *tconn.Connection,
	_ *gsc.Stream,
	args []string,
) port.HandleResult {
	return port.HandleResult{
		Commands:    render.Echo(args),
		HasResponse: true,
	}
}

// ErrURLNoArg is returned when the "url" command is invoked without
// the required first argument.
type ErrURLNoArg struct{}

// Error implements the error interface.
func (ErrURLNoArg) Error() string { return "gsc: url: no argument" }

// URL renders the earlier "open <url>" Time command sequence used to
// redirect the client to an external page.
type URL struct{}

// Name implements gsc.CommandHandler.
func (URL) Name() string { return "url" }

// Handle implements gsc.CommandHandler.
func (URL) Handle(
	_ context.Context,
	_ *tconn.Connection,
	_ *gsc.Stream,
	args []string,
) port.HandleResult {
	if len(args) == 0 {
		return port.HandleResult{HasResponse: false, Err: ErrURLNoArg{}}
	}

	return port.HandleResult{
		Commands:    render.Time("0", "open:"+args[0]),
		HasResponse: true,
	}
}
