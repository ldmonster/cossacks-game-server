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

// Documented stubs for route. Centralising them here keeps the
// dispatcher free from inline empty-response literals and makes the set
// of intentional no-ops easy to audit.

package gsc

import "github.com/ldmonster/cossacks-game-server/internal/transport/gsc"

// unimplementedOpenRoutes lists `open`/`go` route names whose reference
// implementation has no concrete body. The dispatcher routes any
// matching method through `emptyOpenResponse`
//
// TODO: Decide whether each of these will ever gain a real
// implementation or be removed from the public protocol surface.
var unimplementedOpenRoutes = map[string]struct{}{
	"direct":               {},
	"direct_ping":          {},
	"direct_join":          {},
	"started_room_message": {},
}

// isUnimplementedOpenRoute reports whether `method` is a documented
// no-op open route.
func isUnimplementedOpenRoute(method string) bool {
	_, ok := unimplementedOpenRoutes[method]
	return ok
}

// emptyOpenResponse is the canonical empty payload returned by
// unimplemented.
func emptyOpenResponse() []gsc.Command {
	return []gsc.Command{}
}

// unimplementedCommands lists GSC top-level command names that have no
// concrete body in the reference implementation. They are intentional no-ops
// at the dispatcher level. `upfile` and `unsync` are client-only
// flow-control beacons.
var unimplementedCommands = map[string]struct{}{
	"upfile": {},
	"unsync": {},
}

// isUnimplementedCommand reports whether `name` is a documented no-op
// top-level command.
func isUnimplementedCommand(name string) bool {
	_, ok := unimplementedCommands[name]
	return ok
}
