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

	"github.com/ldmonster/cossacks-game-server/internal/transport/gsc"
	tconn "github.com/ldmonster/cossacks-game-server/internal/transport/tconn"
)

// TryEnter delegates to the in-package TryEnterImpl body.
func (r *Routes) TryEnter(
	ctx context.Context, conn *tconn.Connection, req *gsc.Stream, p map[string]string,
) ([]gsc.Command, error) {
	return r.TryEnterImpl(ctx, conn, req, p)
}

// RegNewRoom delegates to the in-package RegNewRoomImpl body.
func (r *Routes) RegNewRoom(
	ctx context.Context, conn *tconn.Connection, req *gsc.Stream, p map[string]string,
) ([]gsc.Command, error) {
	return r.RegNewRoomImpl(ctx, conn, req, p)
}

// JoinGame delegates to the in-package JoinGameImpl body.
func (r *Routes) JoinGame(
	ctx context.Context, conn *tconn.Connection, req *gsc.Stream, p map[string]string,
) ([]gsc.Command, error) {
	return r.JoinGameImpl(ctx, conn, req, p)
}

// RoomInfo delegates to the in-package RoomInfoImpl body.
func (r *Routes) RoomInfo(
	ctx context.Context, conn *tconn.Connection, req *gsc.Stream, p map[string]string,
) ([]gsc.Command, error) {
	return r.RoomInfoImpl(ctx, conn, req, p)
}

// JoinPlayer delegates to the in-package JoinPlayerImpl body.
func (r *Routes) JoinPlayer(
	ctx context.Context, conn *tconn.Connection, req *gsc.Stream, p map[string]string,
) ([]gsc.Command, error) {
	return r.JoinPlayerImpl(ctx, conn, req, p)
}
