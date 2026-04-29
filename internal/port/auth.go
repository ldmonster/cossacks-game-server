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
	"context"

	"github.com/ldmonster/cossacks-game-server/internal/domain/identity"
)

// AuthProvider is the contract for verifying credentials against an
// external account system (LCN, etc.). Implementations may be HTTP-backed
// or a test stub.
type AuthProvider interface {
	// Authenticate verifies login/password for the given account type. On
	// success it returns the populated AccountInfo; on failure (bad
	// credentials, network error) it returns an error and a zero-value
	// AccountInfo.
	Authenticate(
		ctx context.Context,
		accountType identity.AccountType,
		login, password string,
	) (identity.AccountInfo, error)
}
