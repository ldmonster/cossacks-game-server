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
	"strings"

	"github.com/ldmonster/cossacks-game-server/internal/render"
	"github.com/ldmonster/cossacks-game-server/internal/transport/gsc"
	tconn "github.com/ldmonster/cossacks-game-server/internal/transport/tconn"
)

// ErrGGCupUnavailable signals that the gg_cup_thanks_dgl route had no
// supporter data to render. The user-visible response is the
// "No info yet" alert; the error value is observability metadata.
type ErrGGCupUnavailable struct{}

// Error implements the error interface.
func (ErrGGCupUnavailable) Error() string {
	return "routes: gg cup info unavailable"
}

// GGCupThanks renders the GG-Cup "thanks" supporter list using cached
// data from RankingProvider.
func (r *Routes) GGCupThanks(
	_ context.Context, _ *tconn.Connection, req *gsc.Stream, _ map[string]string,
) ([]gsc.Command, error) {
	var ggCup map[string]any
	if r != nil && r.deps.Ranking != nil {
		ggCup = r.deps.Ranking.LoadGGCup()
	}

	if ggCup == nil || ggCupBool(ggCup["wo_info"]) {
		return r.renderAlert(req.Ver, "Thanks for", "No info yet"), ErrGGCupUnavailable{}
	}

	supportersAny, _ := ggCup["supporters"].([]any)

	supporters := make([]map[string]any, 0, len(supportersAny))
	for _, sAny := range supportersAny {
		s, ok := sAny.(map[string]any)
		if !ok {
			continue
		}

		supporters = append(supporters, s)
	}

	if len(supporters) == 0 {
		return r.renderAlert(req.Ver, "Thanks for", "No info yet"), ErrGGCupUnavailable{}
	}

	return render.Show(render.GGCupThanksBody(supporters)), nil
}

// ggCupBool mirrors the relaxed boolean-coercion used by the earlier
// handler (string "0"/"false" → false, empty → false, otherwise true).
func ggCupBool(v any) bool {
	switch t := v.(type) {
	case bool:
		return t
	case string:
		return t != "" && t != "0" && strings.ToLower(t) != "false"
	default:
		return false
	}
}
