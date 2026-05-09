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

	"github.com/ldmonster/cossacks-game-server/internal/render"
	"github.com/ldmonster/cossacks-game-server/internal/transport/gsc"
	tconn "github.com/ldmonster/cossacks-game-server/internal/transport/tconn"
)

// Enter renders the initial `enter` dialog. When the connection
// already carries an authenticated account it pre-fills the form;
// otherwise it falls back to the requested login type.
func (r *Routes) Enter(
	_ context.Context, conn *tconn.Connection, req *gsc.Stream, p map[string]string,
) ([]gsc.Command, error) {
	if conn.Session != nil && conn.Session.Account != nil &&
		string(conn.Session.Account.Type) != "" {
		acc := conn.Session.Account

		return r.RenderEnter(req, string(acc.Type), "", "1", acc.Login, acc.ID), nil
	}

	return r.RenderEnter(req, p["TYPE"], "", "", "", ""), nil
}

// RenderEnter materialises the `enter.tmpl` show body. Exported so
// other (still handler-side) auth flows can share the same renderer
// during the migration.
func (r *Routes) RenderEnter(
	req *gsc.Stream,
	loginType, errText, loggedIn, nick, id string,
) []gsc.Command {
	tableTimeout := 0
	if r.deps.Game != nil {
		tableTimeout = r.deps.Game.TableTimeout
	}

	chatServer := ""
	serverTitle := ""
	if r.deps.Game != nil {
		chatServer = r.deps.Game.ChatServer
		serverTitle = r.deps.Game.ServerTitle
	}

	serverName := ""
	if r.deps.Server != nil {
		serverName = r.deps.Server.HostName
	}

	view := &render.View{
		Type:         loginType,
		Error:        errText,
		LoggedIn:     loggedIn == "1",
		Nick:         nick,
		Nickname:     nick,
		ID:           id,
		ChatServer:   chatServer,
		ServerTitle:  serverTitle,
		TableTimeout: tableTimeout,
		Ver:          strconv.Itoa(int(req.Ver)),
		ServerName:   serverName,
	}

	return render.Show(r.render(req.Ver, "enter.tmpl", view))
}
