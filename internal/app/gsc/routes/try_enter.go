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
	"strconv"

	"github.com/ldmonster/cossacks-game-server/internal/app/identity"
	"github.com/ldmonster/cossacks-game-server/internal/domain/player"
	"github.com/ldmonster/cossacks-game-server/internal/domain/session"
	"github.com/ldmonster/cossacks-game-server/internal/render"
	"github.com/ldmonster/cossacks-game-server/internal/transport/gsc"
	tconn "github.com/ldmonster/cossacks-game-server/internal/transport/tconn"
)

// TryEnterImpl implements the `try_enter` route. It drives the
// identity service decision and renders the resulting dialog.
func (r *Routes) TryEnterImpl(
	ctx context.Context, conn *tconn.Connection, req *gsc.Stream, p map[string]string,
) ([]gsc.Command, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	s := ensureSessionRoutes(conn)

	currentAcc := s.Account

	dec := r.deps.Identity.TryEnterDecide(
		ctx,
		identity.ParseEnterRequest(p, currentAcc, conn.IP),
	)

	if dec.DevFlag != nil {
		s.Dev = *dec.DevFlag
	}

	if dec.SetHeight != 0 {
		s.WindowH = dec.SetHeight
	}

	if dec.ClearAccount {
		s.Account = nil
	}

	switch dec.Kind {
	case identity.EnterRender:
		return r.RenderEnter(req, dec.LoginType, dec.Message, "", "", ""), dec.Err

	case identity.EnterRenderErrorPage:
		return render.Show(r.render(
			req.Ver,
			"error_enter.tmpl",
			map[string]string{"error_text": dec.Message},
		)), dec.Err

	case identity.EnterSuccess:
		if dec.Account != nil {
			s.Account = dec.Account
		}

		if dec.RunPostAccountAction {
			r.postAccountAction(conn, "enter", nil)
		}

		return r.successEnter(conn, req, dec.Nick), dec.Err
	}

	return r.RenderEnter(req, "", "", "", "", ""), dec.Err
}

// postAccountAction fires the post-enter notification to the auth
// provider. Errors are swallowed (wire fidelity).
func (r *Routes) postAccountAction(
	conn *tconn.Connection,
	action string,
	payload map[string]any,
) {
	if conn.Session == nil || conn.Session.Account == nil {
		return
	}

	acc := conn.Session.Account
	r.deps.Identity.PostAccountAction(
		context.Background(),
		string(acc.Type),
		acc.ID,
		conn.IP,
		action,
		payload,
	)
}

// successEnter finalises a successful enter decision: it allocates a
// player id, registers the session, and renders ok_enter.
func (r *Routes) successEnter(
	conn *tconn.Connection,
	req *gsc.Stream,
	nick string,
) []gsc.Command {
	sess := ensureSessionRoutes(conn)

	id := uint32(sess.PlayerID)
	if id > 0 {
		// re-enter keeps id and leaves current room.
		if r.deps.Rooms != nil {
			r.deps.Rooms.LeaveByPlayer(id)
		}
	} else {
		id = r.deps.Players.NextPlayerID()
	}

	pl := &player.Player{
		ID:          id,
		Nick:        nick,
		ConnectedAt: conn.Ctime,
	}
	if conn.Session != nil && conn.Session.Account != nil {
		acc := conn.Session.Account
		pl.AccountType = string(acc.Type)
		pl.AccountLogin = acc.Login
		pl.AccountID = acc.ID
		pl.AccountProfile = acc.Profile
	}

	r.deps.Players.UpsertPlayer(pl)

	sess.PlayerID = player.PlayerID(id)
	sess.Nick = nick

	if r.deps.Sessions != nil {
		r.deps.Sessions.Register(id, conn)
	}

	return render.Show(r.render(req.Ver, "ok_enter.tmpl", map[string]string{
		"nick":        nick,
		"id":          fmt.Sprintf("%d", id),
		"chat_server": r.deps.Game.ChatServer,
		"window_size": WindowSize(conn),
		"ver":         strconv.Itoa(int(req.Ver)),
	}))
}

// ensureSessionRoutes mirrors handler.ensureSession.
func ensureSessionRoutes(conn *tconn.Connection) *session.Session {
	if conn.Session == nil {
		conn.Session = session.New()
	}

	return conn.Session
}
