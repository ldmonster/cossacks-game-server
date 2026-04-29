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

package port

import (
	"github.com/ldmonster/cossacks-game-server/internal/domain/player"
	"github.com/ldmonster/cossacks-game-server/internal/domain/session"
)

// SessionStore is the contract for per-connection session state. The
// in-memory implementation is owned by the dispatcher; tests can supply
// a mock without spinning up TCP connections.
//
// SessionStore intentionally does not expose the underlying map: callers
// reach session fields via Get and write back via Set. This keeps the
// untyped map[string]any (Connection.Data) from leaking outward.
type SessionStore interface {
	Get(id player.ConnectionID) (*session.Session, bool)
	Set(id player.ConnectionID, s *session.Session)
	Delete(id player.ConnectionID)
}
