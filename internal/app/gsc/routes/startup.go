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
	"strconv"
	"strings"

	"go.uber.org/zap"

	rankingapp "github.com/ldmonster/cossacks-game-server/internal/app/ranking"
	"github.com/ldmonster/cossacks-game-server/internal/render"
	"github.com/ldmonster/cossacks-game-server/internal/transport/gsc"
	tconn "github.com/ldmonster/cossacks-game-server/internal/transport/tconn"
)

// Startup renders the lobby/startup show body for the `startup`,
// `games` and `rooms_table_dgl` open routes.
func (r *Routes) Startup(
	_ context.Context, conn *tconn.Connection, req *gsc.Stream, _ map[string]string,
) ([]gsc.Command, error) {
	vars := map[string]string{
		"window_size":   WindowSize(conn),
		"chat_server":   r.deps.Game.ChatServer,
		"table_timeout": strconv.Itoa(r.deps.Game.TableTimeout),
		"ver":           strconv.Itoa(int(req.Ver)),
		// cs/startup.tmpl defines this via TT SET; provide explicit
		// value so the generated show body keeps valid y/h coordinates
		// for the bottom bar.
		"bottom_height": "32",
	}

	if r.deps.Ranking != nil {
		rankingapp.MergeGGCupIntoStartupVars(r.deps.Ranking.LoadGGCup(), vars)
	}

	body := r.render(req.Ver, "startup.tmpl", vars)

	if r.deps.Log != nil {
		btnLines := make([]string, 0, 8)

		for _, ln := range strings.Split(body, "\n") {
			trim := strings.TrimSpace(ln)
			if strings.HasPrefix(trim, "#btn(") || strings.HasPrefix(trim, "#btn[") {
				btnLines = append(btnLines, trim)
			}
		}

		r.deps.Log.Debug("startup payload",
			zap.Uint64("conn_id", conn.ID),
			zap.Uint8("ver", req.Ver),
			zap.Bool("has_join", strings.Contains(body, "join_game.dcml")),
			zap.Bool("has_new", strings.Contains(body, "new_room_dgl.dcml")),
			zap.Int("body_len", len(body)),
			zap.Strings("btn_lines", btnLines),
		)
	}

	return render.Show(body), nil
}
