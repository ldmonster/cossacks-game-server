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

// resize open-route — records the client's reported window height on
// the session and emits the canonical <RESIZE> response.

package routes

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/ldmonster/cossacks-game-server/internal/domain/session"
	"github.com/ldmonster/cossacks-game-server/internal/render"
	"github.com/ldmonster/cossacks-game-server/internal/transport/gsc"
	tconn "github.com/ldmonster/cossacks-game-server/internal/transport/tconn"
)

// Resize records the client's window height and returns the
// "<RESIZE>" preamble. When the height parameter is non-empty but
// invalid, the response is still sent (reference behaviour) and an
// ErrResizeBadHeight is returned for observability.
func (r *Routes) Resize(
	_ context.Context,
	conn *tconn.Connection,
	_ *gsc.Stream,
	p map[string]string,
) ([]gsc.Command, error) {
	raw := strings.TrimSpace(p["height"])

	height, parseErr := strconv.Atoi(raw)
	if parseErr == nil {
		EnsureSession(conn).WindowH = height
	}

	var retErr error
	if raw != "" && parseErr != nil {
		retErr = ErrResizeBadHeight{Raw: raw}
	}

	if WindowSize(conn) == "large" {
		return render.Show("<RESIZE>\n#large\n<RESIZE>"), retErr
	}

	return render.Show("<RESIZE>\n<RESIZE>"), retErr
}

// EnsureSession returns conn.Session, creating an empty one when nil.
// Exposed at package level so other migrated routes can reuse it
// without going through Controller.
func EnsureSession(conn *tconn.Connection) *session.Session {
	if conn.Session == nil {
		conn.Session = session.New()
	}

	return conn.Session
}

// WindowSize returns "large" when the session reports a window height
// > 366 and "small" otherwise. Mirrors the wire fidelity rule used
// when picking the right template variant.
func WindowSize(conn *tconn.Connection) string {
	if conn == nil || conn.Session == nil {
		return "small"
	}

	if conn.Session.WindowH > 366 {
		return "large"
	}

	return "small"
}

// ErrResizeBadHeight signals that the resize open-route received a
// non-empty but unparsable height parameter. The user-visible
// response is unchanged (wire fidelity) — Err carries the observability
// metadata only.
type ErrResizeBadHeight struct{ Raw string }

// Error implements the error interface.
func (e ErrResizeBadHeight) Error() string {
	return fmt.Sprintf("handler: resize: bad height %q", e.Raw)
}
