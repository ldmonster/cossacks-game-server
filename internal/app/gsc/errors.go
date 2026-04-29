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
	"fmt"

	gsccmds "github.com/ldmonster/cossacks-game-server/internal/app/gsc/commands"
)

// errUnknownCommand signals that the GSC dispatcher received a command
// name it does not implement. Returns an empty response;
// Err allows callers to distinguish this from an intentional empty.
type errUnknownCommand struct{ name string }

func (e errUnknownCommand) Error() string {
	return fmt.Sprintf("handler: unknown command %q", e.name)
}

// errUnknownOpenRoute signals that an `open`/`go` route name is not
// implemented. The user-visible response is a "Page Not Found" alert
// (see dispatchOpen); Err is metadata for observability.
type errUnknownOpenRoute struct{ method string }

func (e errUnknownOpenRoute) Error() string {
	return fmt.Sprintf("handler: unknown open route %q", e.method)
}

// errURLNoArg signals that the GSC `url` command was issued without
// the required URL argument. It aliases the canonical error type
// defined in internal/app/gsc/commands so handler-level tests using
// errors.As keep passing while the command lives in the application
// layer.
type errURLNoArg = gsccmds.ErrURLNoArg
