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

package routes

import (
	"context"

	"github.com/ldmonster/cossacks-game-server/internal/app/identity"
	tconn "github.com/ldmonster/cossacks-game-server/internal/transport/tconn"
)

// IdentityService is the consumer port for routes that need to drive
// the try_enter authentication flow. It mirrors the subset of
// identity.Service used by the route.
type IdentityService interface {
	TryEnterDecide(ctx context.Context, req identity.EnterRequest) identity.EnterDecision
	PostAccountAction(
		ctx context.Context,
		accountType, accountID, clientIP, action string,
		payload map[string]any,
	)
}

// SessionRegistry is the consumer port for routes that need to map a
// player id to the live connection (used after a successful enter).
type SessionRegistry interface {
	Register(playerID uint32, conn *tconn.Connection)
}
