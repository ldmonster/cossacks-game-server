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

// parseOpenParams is a thin alias for the canonical implementation in
// internal/app/gsc/commands. It is kept so existing handler-package
// callers compile without touching every site; it will be removed in
// alongside the rest of the dispatcher.

package gsc

import gsccmds "github.com/ldmonster/cossacks-game-server/internal/app/gsc/commands"

func parseOpenParams(params string) map[string]string {
	return gsccmds.ParseOpenParams(params)
}
