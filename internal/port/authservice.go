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

import "context"

// AuthService is the application-level contract for the authentication
// subsystem. It is satisfied by auth.Service in internal/server/auth.
//
// The interface is intentionally narrow: it captures only the methods
// that use domain/basic types so other packages can depend on the
// interface without importing the auth package.
type AuthService interface {
	// PostAccountAction reports a player event to the upstream account
	// server (enter, start, endgame, …). Errors are fire-and-forget;
	// failures are logged by the implementation and never returned.
	PostAccountAction(
		ctx context.Context,
		accountType, accountID, clientIP, action string,
		payload map[string]any,
	)
}
