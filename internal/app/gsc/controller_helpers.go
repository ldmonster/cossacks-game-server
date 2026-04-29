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

// Pure helpers used across the controller (no Controller methods).

package gsc

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/ldmonster/cossacks-game-server/internal/domain/session"
	"github.com/ldmonster/cossacks-game-server/internal/render"
	tconn "github.com/ldmonster/cossacks-game-server/internal/transport/tconn"
)

// ensureSession lazily initialises a Session on conn (defensive helper
// for tests that construct Connection literals without a Session).
func ensureSession(conn *tconn.Connection) *session.Session {
	if conn.Session == nil {
		conn.Session = session.New()
	}

	return conn.Session
}

func normalizePage(v string) string {
	s := strings.TrimSpace(v)
	if s == "" {
		return "1"
	}

	if _, err := strconv.ParseUint(s, 10, 64); err != nil {
		return "1"
	}

	if s != "1" && s != "2" && s != "3" {
		return "1"
	}

	return s
}

func normalizeRes(v string) string {
	s := strings.TrimSpace(v)
	if s == "" {
		return "0"
	}

	if _, err := strconv.ParseUint(s, 10, 64); err != nil {
		return "0"
	}

	return s
}

func setRoomPlayersColumn(row []string, playersCount, maxPlayers int) []string {
	if len(row) == 0 {
		return row
	}

	out := append([]string(nil), row...)

	idx := len(out) - 4
	if idx < 0 {
		idx = len(out) - 1
	}

	out[idx] = fmt.Sprintf("%d/%d", playersCount, maxPlayers)

	return out
}

// timeIntervalFromElapsedSec is a thin alias around render.TimeIntervalFromElapsedSec
// so existing callers stay unchanged.
func timeIntervalFromElapsedSec(secs int) string {
	return render.TimeIntervalFromElapsedSec(secs)
}
