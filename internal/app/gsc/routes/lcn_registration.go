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

// lcn_registration_dgl open-route — renders the LCN registration
// confirm dialog. The destination URL is sourced from the configured
// LCN auth provider host.

package routes

import (
	"context"

	"github.com/ldmonster/cossacks-game-server/internal/render"
	"github.com/ldmonster/cossacks-game-server/internal/transport/gsc"
	tconn "github.com/ldmonster/cossacks-game-server/internal/transport/tconn"
)

// LCNRegistrationDialog renders the LCN registration confirm dialog
// for the `lcn_registration_dgl` open-route.
func (r *Routes) LCNRegistrationDialog(
	_ context.Context,
	_ *tconn.Connection,
	req *gsc.Stream,
	_ map[string]string,
) ([]gsc.Command, error) {
	return render.Show(r.render(req.Ver, "confirm_dgl.tmpl", map[string]string{
		"header":  "LCN Registration",
		"text":    "Open www.newlcn.com?",
		"ok_text": "Ok",
		"height":  "100",
		"command": "GW|url&http://" + r.deps.Auth.Provider("LCN").Host +
			"/lang_redir.php&from=tournaments",
	})), nil
}
