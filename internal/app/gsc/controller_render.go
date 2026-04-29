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

package gsc

import (
	"github.com/ldmonster/cossacks-game-server/internal/render"
	"github.com/ldmonster/cossacks-game-server/internal/transport/gsc"
)

// render is the Controller-scoped template lookup used by all
// in-package handler files. It resolves through the controller's
// `Renderer` (an instance-scoped `*render.TemplateRenderer`). When the
// renderer is missing (e.g. struct-literal test construction), the
// immutable package defaults are used so the call still resolves
// without depending on any global mutable state.
func (c *Controller) render(ver uint8, name string, vars map[string]string) string {
	if c != nil && c.Renderer != nil {
		return c.Renderer.Render(ver, name, vars)
	}

	return render.LoadShowBodyFromRoots(render.DefaultTemplateRoots, ver, name, vars)
}

// renderAlert is a DRY helper for the most common LW response shape
// (alert_dgl.tmpl with header+text). It exists to consolidate the
// repeated 4-line literal at 16 production call sites.
//
//nolint:unparam // remaining handler call sites all pass "Error"; non-Error headers now live in routes.
func (c *Controller) renderAlert(ver uint8, header, text string) []gsc.Command {
	return render.Show(c.render(ver, "alert_dgl.tmpl", map[string]string{
		"header": header,
		"text":   text,
	}))
}
