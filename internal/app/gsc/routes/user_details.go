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
	"time"

	matchapp "github.com/ldmonster/cossacks-game-server/internal/app/match"
	"github.com/ldmonster/cossacks-game-server/internal/domain/lobby"
	"github.com/ldmonster/cossacks-game-server/internal/domain/player"
	"github.com/ldmonster/cossacks-game-server/internal/render"
	"github.com/ldmonster/cossacks-game-server/internal/transport/gsc"
	tconn "github.com/ldmonster/cossacks-game-server/internal/transport/tconn"
)

// ErrInvalidPlayerArg signals that a route received a player
// identifier that could not be parsed as uint32.
type ErrInvalidPlayerArg struct{ Raw string }

// Error implements the error interface.
func (e ErrInvalidPlayerArg) Error() string {
	return fmt.Sprintf("routes: invalid player id %q", e.Raw)
}

// ErrPlayerNotFound signals that the requested player is not
// registered.
type ErrPlayerNotFound struct{ ID uint32 }

// Error implements the error interface.
func (e ErrPlayerNotFound) Error() string {
	return fmt.Sprintf("routes: player %d not found", e.ID)
}

// UserDetails renders the user_details dialog for the player whose id
// was supplied in the `ID` parameter. Returns the alert command list
// for wire fidelity even on error.
func (r *Routes) UserDetails(
	_ context.Context, conn *tconn.Connection, _ *gsc.Stream, p map[string]string,
) ([]gsc.Command, error) {
	// Mirror the earlier handler: warm the LCN cache so the place
	// lookup below sees fresh data.
	if r != nil && r.deps.Ranking != nil {
		_ = r.deps.Ranking.LoadLCN()
	}

	raw := p["ID"]

	id, err := parsePlayerIDArg(raw)
	if err != nil {
		return []gsc.Command{}, ErrInvalidPlayerArg{Raw: raw}
	}

	if r == nil || r.deps.Players == nil {
		return []gsc.Command{}, ErrPlayerNotFound{ID: id}
	}

	pl := r.deps.Players.GetPlayer(id)
	if pl == nil {
		return []gsc.Command{}, ErrPlayerNotFound{ID: id}
	}

	room := r.deps.Players.GetRoomByHost(pl.ID)

	return render.Show(r.buildUserDetailsBody(conn, pl, room)), nil
}

// buildUserDetailsBody adapts domain types into render.UserDetailsBody
// inputs.
func (r *Routes) buildUserDetailsBody(
	conn *tconn.Connection,
	pl *player.Player,
	room *lobby.Room,
) string {
	dev := conn.Session != nil && conn.Session.Dev

	rp := render.UserDetailsPlayer{
		ID:             pl.ID,
		Nick:           pl.Nick,
		ConnectedAt:    pl.ConnectedAt,
		AccountType:    pl.AccountType,
		AccountLogin:   pl.AccountLogin,
		AccountID:      pl.AccountID,
		AccountProfile: pl.AccountProfile,
	}

	var rr *render.UserDetailsRoom
	if room != nil {
		rr = &render.UserDetailsRoom{ID: room.ID, Title: room.Title}
	}

	connectedAgo := render.TimeIntervalFromElapsedSec(int(time.Since(pl.ConnectedAt).Seconds()))

	var lcnPlaces map[string]int
	if r.deps.Ranking != nil {
		lcnPlaces = r.deps.Ranking.LCNPlaceByID()
	}

	return render.UserDetailsBody(dev, rp, rr, lcnPlaces, connectedAgo)
}

// parsePlayerIDArg converts a route parameter into a uint32 player id.
func parsePlayerIDArg(v string) (uint32, error) {
	i, err := matchapp.IntArg(v)
	if err != nil {
		return 0, err
	}

	return uint32(i), nil
}
