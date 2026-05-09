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

// Template lookup + Go-template fragment renderer. Migrated here from
// the handler package.
// The handler package now keeps thin aliases for older call
// sites so the move is non-breaking.

package render

import (
	"os"
	"path/filepath"
	"strings"
)

// DefaultTemplateRoots is the built-in lookup search path used when no
// custom root is provided. Order: cwd-local `templates/`, then the older
// share paths.
var DefaultTemplateRoots = []string{
	"/app/templates",
	"/cossacks/templates",
	"templates",
	"../templates",
	"../../templates",
	"/cossacks/SimpleCossacksServer/share",
}

// TemplateRenderer is the typed, instance-scoped owner of the search
// path used by template lookup. It satisfies port.TemplateRenderer.
type TemplateRenderer struct {
	roots []string
}

// NewTemplateRenderer returns a renderer whose search path puts
// customRoot (when non-empty) ahead of the built-in defaults.
// Duplicates and empty entries are ignored.
func NewTemplateRenderer(customRoot string) *TemplateRenderer {
	return &TemplateRenderer{roots: BuildTemplateRoots(customRoot, DefaultTemplateRoots)}
}

// Roots returns a copy of the current template search path.
func (r *TemplateRenderer) Roots() []string {
	if r == nil || len(r.roots) == 0 {
		return append([]string(nil), DefaultTemplateRoots...)
	}

	return append([]string(nil), r.roots...)
}

// Render satisfies port.TemplateRenderer.
func (r *TemplateRenderer) Render(ver uint8, name string, data any) string {
	return LoadShowBodyFromRoots(r.Roots(), ver, name, data)
}

// BuildTemplateRoots prepends customRoot (if non-empty) to defaults
// and removes duplicates and empty entries.
func BuildTemplateRoots(customRoot string, defaults []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(defaults)+1)
	add := func(v string) {
		v = strings.TrimSpace(v)
		if v == "" || seen[v] {
			return
		}

		seen[v] = true
		out = append(out, v)
	}
	add(customRoot)

	for _, root := range defaults {
		add(root)
	}

	return out
}

// IsAC reports whether ver is one of the AC client protocol versions.
func IsAC(ver uint8) bool {
	return ver == 3 || ver == 8 || ver == 10
}

// NormalizeShowTemplateName rewrites .cml suffixes to .tmpl and adds
// the suffix when missing. Returns "" for empty input.
func NormalizeShowTemplateName(name string) string {
	s := strings.TrimSpace(name)
	if s == "" {
		return ""
	}

	s = filepath.ToSlash(s)

	lower := strings.ToLower(s)
	if strings.HasSuffix(lower, ".cml") {
		return s[:len(s)-4] + ".tmpl"
	}

	if strings.HasSuffix(lower, ".tmpl") {
		return s
	}

	return s + ".tmpl"
}

// FallbackShowBody returns the canonical default `LW_show` body used
// when a template lookup misses every search root.
func FallbackShowBody() string {
	return "#font(WF,WF,WF)\n#txt(%BOX[x:10,y:10,w:100%,h:24],{},\"server response\")"
}

// LoadShowBodyFromRoots resolves and renders {root}/cs|ac/{name}.tmpl
// against `roots`. Falls back to FallbackShowBody when no root has it.
func LoadShowBodyFromRoots(
	roots []string,
	ver uint8,
	templateName string,
	data any,
) string {
	name := NormalizeShowTemplateName(templateName)
	if name == "" {
		return FallbackShowBody()
	}

	dir := "cs"
	if IsAC(ver) {
		dir = "ac"
	}

	for _, root := range roots {
		path := filepath.Join(root, dir, filepath.FromSlash(name))

		b, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		extras := loadSubtemplates(path)

		return RenderShowTemplate(string(b), data, extras...)
	}

	return FallbackShowBody()
}

// loadSubtemplates reads sibling `.tmpl` files in a directory named
// after the main template (without extension) so that
// `{{template "name" .}}` references in the main file resolve.
//
// E.g. for `templates/cs/started_room_info.tmpl`, this returns the
// contents of every `.tmpl` under `templates/cs/started_room_info/`.
func loadSubtemplates(mainPath string) []string {
	ext := filepath.Ext(mainPath)
	dir := strings.TrimSuffix(mainPath, ext)

	st, err := os.Stat(dir)
	if err != nil || !st.IsDir() {
		return nil
	}

	var out []string

	_ = filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}

		if !strings.HasSuffix(strings.ToLower(p), ".tmpl") {
			return nil
		}

		b, rerr := os.ReadFile(p)
		if rerr != nil {
			return nil
		}

		out = append(out, string(b))

		return nil
	})

	return out
}

// RenderShowTemplate executes the Go text/template engine over `src`.
// Optional `extras` are additional template sources (e.g. sibling
// files containing `{{define}}` blocks) parsed alongside the main
// src so cross-file `{{template "name" .}}` references resolve.
//
// Any `<%...%>` and `<? P.* ?>` tokens are passed through verbatim;
// they are interpreted by the game client.
func RenderShowTemplate(src string, data any, extras ...string) string {
	return strings.TrimSpace(renderGoTemplate(src, data, extras...))
}
