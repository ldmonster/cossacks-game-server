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

// Package gsc holds the application-layer GSC command handlers. The
// handlers replace the earlier controller_*.go files in
// internal/server/handler. Each command/route is a small struct that
// implements CommandHandler/RouteHandler and is registered into the
// dispatcher via the WithX options at composition time.
package gsc

import (
	"context"

	"github.com/ldmonster/cossacks-game-server/internal/port"
	"github.com/ldmonster/cossacks-game-server/internal/transport/gsc"
	tconn "github.com/ldmonster/cossacks-game-server/internal/transport/tconn"
)

// Result is the outcome of a single command dispatch.
type Result = port.HandleResult

// CommandHandler is the contract every top-level GSC command satisfies.
// Implementations live in internal/app/gsc/commands/ and are registered
// into a Registry at startup.
type CommandHandler interface {
	// Name returns the GSC command name (case-sensitive) this handler
	// services. The dispatcher looks up handlers by this key.
	Name() string

	// Handle executes the command and returns a Result. Implementations
	// must be safe for concurrent invocation; any required locking is
	// the responsibility of the underlying services.
	Handle(
		ctx context.Context,
		conn *tconn.Connection,
		req *gsc.Stream,
		args []string,
	) Result
}

// Registry stores CommandHandlers keyed by their Name().
type Registry struct {
	byName map[string]CommandHandler
}

// NewRegistry constructs an empty Registry.
func NewRegistry() *Registry {
	return &Registry{byName: make(map[string]CommandHandler)}
}

// Register adds h to the registry. A subsequent registration with the
// same Name() overwrites the previous entry.
func (r *Registry) Register(h CommandHandler) {
	r.byName[h.Name()] = h
}

// Lookup returns the handler registered for name, or nil and false if
// no such handler is registered. A nil receiver is treated as an empty
// registry so call sites can use Lookup without nil-checking.
func (r *Registry) Lookup(name string) (CommandHandler, bool) {
	if r == nil || r.byName == nil {
		return nil, false
	}

	h, ok := r.byName[name]

	return h, ok
}
