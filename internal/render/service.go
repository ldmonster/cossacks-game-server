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

package render

import "github.com/ldmonster/cossacks-game-server/internal/port"

// Service is the renderer facade.
// It composes a port.TemplateRenderer (for show-template lookup) with
// the package-level builders so callers depend on this single type
// instead of mixing render package functions and a free-floating
// renderer pointer.
type Service struct {
	tpl port.TemplateRenderer
}

// NewService wraps tpl. tpl may be nil; in that case Render returns "".
func NewService(tpl port.TemplateRenderer) *Service { return &Service{tpl: tpl} }

// Renderer returns the wrapped TemplateRenderer (nil-safe).
func (s *Service) Renderer() port.TemplateRenderer {
	if s == nil {
		return nil
	}

	return s.tpl
}

// Render delegates to the wrapped template renderer. Returns "" when
// no renderer has been wired (nil-safe).
func (s *Service) Render(ver uint8, name string, vars map[string]string) string {
	if s == nil || s.tpl == nil {
		return ""
	}

	return s.tpl.Render(ver, name, vars)
}
