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

import (
	"fmt"
	"strconv"
	"strings"
	"text/template"
	"text/template/parse"
)

// renderGoTemplate runs the Go text/template engine over src against
// the caller-supplied `data`. Templates address Go-side fields with
// PascalCase syntax (`{{.Room.ID}}`) directly: no key remapping, no
// dotted-string projection.
//
// On parse or execution error src is returned unchanged.
func renderGoTemplate(src string, data any, extras ...string) string {
	hasGoToken := strings.Contains(src, "{{")
	hasExtras := len(extras) > 0
	if !hasGoToken && !hasExtras {
		return src
	}

	tpl, err := template.New("show").
		Funcs(showTemplateFuncs).
		Option("missingkey=zero").
		Parse(src)
	if err != nil {
		return src
	}

	for _, ex := range extras {
		if strings.TrimSpace(ex) == "" {
			continue
		}
		if _, err := tpl.Parse(ex); err != nil {
			return src
		}
	}

	// Pick which named template to execute: the root "show" template,
	// or the first non-empty defined sibling when the root body is
	// empty (e.g. files that wrap their entire body in
	// `{{define "name"}}...{{end}}`).
	target := tpl
	if treeIsEmpty(tpl.Tree) {
		for _, t := range tpl.Templates() {
			if t == nil || t == tpl || treeIsEmpty(t.Tree) {
				continue
			}
			target = t
			break
		}
	}

	var buf strings.Builder
	if err := target.Execute(&buf, data); err != nil {
		return src
	}

	return buf.String()
}

// treeIsEmpty reports whether the parsed tree's root list is empty
// (or contains only whitespace text). Used to detect files that only
// contain `{{define ...}}{{end}}` blocks.
func treeIsEmpty(tree *parse.Tree) bool {
	if tree == nil || tree.Root == nil {
		return true
	}
	for _, n := range tree.Root.Nodes {
		if tn, ok := n.(*parse.TextNode); ok {
			if strings.TrimSpace(string(tn.Text)) == "" {
				continue
			}
		}
		return false
	}
	return true
}

// showTemplateFuncs is the FuncMap exposed to on-disk .tmpl bodies.
var showTemplateFuncs = template.FuncMap{
	"cmdFilter":    cmdFilter,
	"argFilter":    argFilter,
	"levelLabel":   tplLevelLabel,
	"levelLabelLC": tplLevelLabelLC,
	"theamLabel":   tplTheamLabel,
	"truthy":       tplTruthy,
	"add":          tplAdd,
	"sub":          tplSub,
	"mul":          tplMul,
	"concat":       tplConcat,
}

// tplTheamLabel renders a team identifier for display.
func tplTheamLabel(v any) string {
	s := strings.TrimSpace(anyToString(v))
	if s == "" || s == "0" {
		return s
	}
	return s
}

// tplTruthy applies Go-bool-like semantics for use inside templates.
// Strings "", "0", "false" are falsy; numeric zero is falsy; nil is
// falsy; everything else is truthy.
func tplTruthy(v any) bool {
	switch t := v.(type) {
	case nil:
		return false
	case bool:
		return t
	case string:
		s := strings.TrimSpace(t)
		if s == "" || s == "0" || strings.EqualFold(s, "false") {
			return false
		}

		return true
	case int:
		return t != 0
	case int64:
		return t != 0
	case uint32:
		return t != 0
	case float64:
		return t != 0
	default:
		return true
	}
}

// cmdFilter escapes characters that would otherwise terminate a
// command segment in the LW command syntax (`&`, `^`, `|`, `%`, NUL).
func cmdFilter(v any) string {
	s := anyToString(v)

	r := strings.NewReplacer(
		"&", `\26`,
		"^", `\5e`,
		"|", `\7c`,
		"%", `\25`,
		"\x00", `\00`,
	)

	return r.Replace(s)
}

// argFilter renders v as an escape-safe argument suitable for
// drawing directives. Quoting (when needed) is the responsibility of
// the surrounding template.
func argFilter(v any) string {
	s := anyToString(v)

	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	s = strings.ReplaceAll(s, "\n", `\n`)

	return s
}

func tplLevelLabel(v any) string {
	switch coerceInt(v) {
	case 1:
		return "Easy"
	case 2:
		return "Normal"
	case 3:
		return "Hard"
	default:
		return "For all"
	}
}

func tplLevelLabelLC(v any) string {
	return strings.ToLower(tplLevelLabel(v))
}

func tplAdd(args ...any) int {
	sum := 0
	for _, a := range args {
		sum += coerceInt(a)
	}

	return sum
}

func tplSub(a, b any, rest ...any) int {
	res := coerceInt(a) - coerceInt(b)
	for _, r := range rest {
		res -= coerceInt(r)
	}

	return res
}

func tplMul(args ...any) int {
	if len(args) == 0 {
		return 0
	}

	prod := 1
	for _, a := range args {
		prod *= coerceInt(a)
	}

	return prod
}

func tplConcat(args ...any) string {
	var b strings.Builder
	for _, a := range args {
		b.WriteString(anyToString(a))
	}

	return b.String()
}

func anyToString(v any) string {
	if v == nil {
		return ""
	}

	switch t := v.(type) {
	case string:
		return t
	case fmt.Stringer:
		return t.String()
	default:
		return fmt.Sprint(v)
	}
}

func coerceInt(v any) int {
	switch t := v.(type) {
	case nil:
		return 0
	case int:
		return t
	case int64:
		return int(t)
	case uint32:
		return int(t)
	case float64:
		return int(t)
	case string:
		s := strings.TrimSpace(t)
		if s == "" {
			return 0
		}

		if n, err := strconv.Atoi(s); err == nil {
			return n
		}

		if f, err := strconv.ParseFloat(s, 64); err == nil {
			return int(f)
		}

		return 0
	default:
		return 0
	}
}
