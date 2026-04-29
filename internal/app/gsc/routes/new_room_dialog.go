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

// new_room_dgl open-route — renders the "create new room" confirm
// dialog when the client is allowed to create a room.

package routes

import (
	"context"
	"fmt"

	"github.com/ldmonster/cossacks-game-server/internal/render"
	"github.com/ldmonster/cossacks-game-server/internal/transport/gsc"
	tconn "github.com/ldmonster/cossacks-game-server/internal/transport/tconn"
)

// NewRoomDialog renders the "create new room" dialog. When the client
// is already in a room (ASTATE empty or "0") it returns the
// "already-in-room" alert and ErrAlreadyInRoom.
func (r *Routes) NewRoomDialog(
	_ context.Context,
	_ *tconn.Connection,
	req *gsc.Stream,
	p map[string]string,
) ([]gsc.Command, error) {
	if p["ASTATE"] == "" || p["ASTATE"] == "0" {
		return r.renderAlert(
			req.Ver,
			"Error",
			"You can not create or join room!\nYou are already participate in some room\nPlease disconnect from that room first to create a new one",
		), ErrAlreadyInRoom{}
	}

	return render.Show(r.render(req.Ver, "new_room_dgl.tmpl", map[string]string{})), nil
}

// ErrAlreadyInRoom signals that a create/join request was rejected
// because the caller is already a member of some room. PlayerID is
// optional metadata captured by some call sites.
type ErrAlreadyInRoom struct{ PlayerID uint32 }

// Error implements the error interface.
func (e ErrAlreadyInRoom) Error() string {
	return fmt.Sprintf("handler: player %d already in a room", e.PlayerID)
}
