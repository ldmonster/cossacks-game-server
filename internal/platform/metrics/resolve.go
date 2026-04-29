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

package metrics

import (
	"strings"

	"github.com/ldmonster/cossacks-game-server/internal/platform/config"
)

// ResolveAddr returns the TCP listen address for the metrics HTTP server.
// Non-empty flag overrides cfg.Metrics.Addr. Empty result means metrics HTTP is disabled.
func ResolveAddr(cfg *config.Config, flag string) string {
	s := strings.TrimSpace(flag)
	if s != "" {
		return s
	}

	if cfg != nil {
		return strings.TrimSpace(cfg.Metrics.Addr)
	}

	return ""
}
