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
	"fmt"
	"strings"

	"github.com/ldmonster/cossacks-game-server/internal/transport/gsc"
	tconn "github.com/ldmonster/cossacks-game-server/internal/transport/tconn"
)

// ErrLCNRankingUnavailable signals that the LCN ranking provider
// returned no data when the tournaments route required it. The
// user-visible response is the "Internal server error" alert; the
// error value is observability metadata.
type ErrLCNRankingUnavailable struct{}

// Error implements the error interface.
func (ErrLCNRankingUnavailable) Error() string {
	return "routes: LCN ranking unavailable"
}

// Tournaments renders the LCN tournaments alert dialog using cached
// ranking data from RankingProvider.
func (r *Routes) Tournaments(
	_ context.Context, _ *tconn.Connection, req *gsc.Stream, p map[string]string,
) ([]gsc.Command, error) {
	option := strings.TrimSpace(p["option"])
	if option == "" {
		option = "total"
	}

	var rating map[string]any
	if r != nil && r.deps.Ranking != nil {
		rating = r.deps.Ranking.LoadLCN()
	}

	if rating == nil {
		return r.renderAlert(req.Ver, "Error", "Internal server error"), ErrLCNRankingUnavailable{}
	}

	rankingByOption, _ := rating["ranking"].(map[string]any)

	rowsAny, _ := rankingByOption[option].([]any)
	if len(rowsAny) == 0 && option != "total" {
		rowsAny, _ = rankingByOption["total"].([]any)
	}

	lines := make([]string, 0, 12)
	lines = append(lines, "LCN Rating: "+option)

	for i, rowAny := range rowsAny {
		if i >= 10 {
			break
		}

		row, _ := rowAny.(map[string]any)
		place := fmt.Sprintf("%v", row["place"])
		nick := fmt.Sprintf("%v", row["nick"])
		score := fmt.Sprintf("%v", row["score"])
		lines = append(lines, fmt.Sprintf("%s. %s (%s)", place, nick, score))
	}

	return r.renderAlert(req.Ver, "Tournaments", strings.Join(lines, "\n")), nil
}
