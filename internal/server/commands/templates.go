package commands

import (
	"math"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

// Default search order: golang/templates (cs|ac, .tmpl), then legacy share paths.
var defaultTemplateRoots = []string{
	"/app/templates",
	"/cossacks/templates",
	"golang/templates",
	"templates",
	"../templates",
	"../../templates",
	"/cossacks/SimpleCossacksServer/share",
}

var templateRoots = append([]string(nil), defaultTemplateRoots...)

// ConfigureTemplateRoots wires config.templates into runtime show-template lookup.
// The configured path is checked first; defaults are kept as fallback.
func ConfigureTemplateRoots(customRoot string) {
	templateRoots = buildTemplateRoots(customRoot, defaultTemplateRoots)
}

func effectiveTemplateRoots() []string {
	if len(templateRoots) > 0 {
		return templateRoots
	}
	return defaultTemplateRoots
}

func buildTemplateRoots(customRoot string, defaults []string) []string {
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

func isAC(ver uint8) bool {
	return ver == 3 || ver == 8 || ver == 10
}

// renderShowTemplate applies the TT-style fragment renderer used by on-disk .tmpl
// bodies (Perl TT subset, not Go text/template).
func renderShowTemplate(src string, vars map[string]string) string {
	if vars == nil {
		vars = map[string]string{}
	}
	src = renderInlineIfBlocks(src, vars)
	lines := strings.Split(src, "\n")
	out := make([]string, 0, len(lines))
	enabled := []bool{true}
	ifConds := []bool{}
	for _, line := range lines {
		trim := strings.TrimSpace(line)
		// Handle TT control blocks used by legacy templates.
		if strings.HasPrefix(trim, "<?") && strings.HasSuffix(trim, "?>") && !strings.Contains(trim, "<%") {
			body := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(trim, "<?"), "?>"))
			body = normalizeTT(body)
			switch {
			case strings.HasPrefix(body, "IF "):
				cond := evalCondition(strings.TrimSpace(strings.TrimPrefix(body, "IF ")), vars)
				parent := enabled[len(enabled)-1]
				enabled = append(enabled, parent && cond)
				ifConds = append(ifConds, cond)
			case body == "ELSE":
				if len(ifConds) > 0 && len(enabled) > 1 {
					parent := enabled[len(enabled)-2]
					enabled[len(enabled)-1] = parent && !ifConds[len(ifConds)-1]
				}
			case strings.HasPrefix(body, "END"):
				if len(enabled) > 1 {
					enabled = enabled[:len(enabled)-1]
				}
				if len(ifConds) > 0 {
					ifConds = ifConds[:len(ifConds)-1]
				}
			}
			continue
		}
		if !enabled[len(enabled)-1] {
			continue
		}
		out = append(out, line)
	}
	res := strings.Join(out, "\n")
	// Replace basic TT expression fragments used by connect flow templates.
	expr := regexp.MustCompile(`(?s)<\?\s*(.*?)\s*\?>`)
	res = expr.ReplaceAllStringFunc(res, func(token string) string {
		m := expr.FindStringSubmatch(token)
		if len(m) < 2 {
			return ""
		}
		return evalExpr(strings.TrimSpace(m[1]), vars)
	})
	return strings.TrimSpace(res)
}

func evalCondition(expr string, vars map[string]string) bool {
	expr = strings.TrimSpace(expr)
	expr = normalizeTT(expr)
	if strings.Contains(expr, "||") {
		parts := strings.Split(expr, "||")
		for _, p := range parts {
			if evalCondition(p, vars) {
				return true
			}
		}
		return false
	}
	if strings.Contains(expr, "&&") {
		parts := strings.Split(expr, "&&")
		for _, p := range parts {
			if !evalCondition(p, vars) {
				return false
			}
		}
		return true
	}
	if strings.HasPrefix(expr, "!") {
		return !evalCondition(strings.TrimSpace(expr[1:]), vars)
	}
	if strings.Contains(expr, "!=") {
		parts := strings.SplitN(expr, "!=", 2)
		if len(parts) == 2 {
			return strings.TrimSpace(evalExpr(parts[0], vars)) != strings.TrimSpace(evalExpr(parts[1], vars))
		}
	}
	if strings.Contains(expr, ">=") {
		parts := strings.SplitN(expr, ">=", 2)
		return compareNum(evalExpr(parts[0], vars), evalExpr(parts[1], vars), func(a, b float64) bool { return a >= b })
	}
	if strings.Contains(expr, "<=") {
		parts := strings.SplitN(expr, "<=", 2)
		return compareNum(evalExpr(parts[0], vars), evalExpr(parts[1], vars), func(a, b float64) bool { return a <= b })
	}
	if strings.Contains(expr, ">") {
		parts := strings.SplitN(expr, ">", 2)
		return compareNum(evalExpr(parts[0], vars), evalExpr(parts[1], vars), func(a, b float64) bool { return a > b })
	}
	if strings.Contains(expr, "<") {
		parts := strings.SplitN(expr, "<", 2)
		return compareNum(evalExpr(parts[0], vars), evalExpr(parts[1], vars), func(a, b float64) bool { return a < b })
	}
	if strings.Contains(expr, "==") {
		parts := strings.SplitN(expr, "==", 2)
		return strings.TrimSpace(evalExpr(parts[0], vars)) == strings.TrimSpace(evalExpr(parts[1], vars))
	}
	v := strings.TrimSpace(evalExpr(expr, vars))
	return v != "" && v != "0" && strings.ToLower(v) != "false"
}

func evalExpr(expr string, vars map[string]string) string {
	expr = normalizeTT(expr)
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return ""
	}
	// Support TT filters like: <? id | cmd ?>. For now we only need source value.
	if strings.Contains(expr, "|") {
		parts := strings.Split(expr, "|")
		expr = strings.TrimSpace(parts[0])
	}
	if s, ok := tryEvalAddMul(expr, vars); ok {
		return s
	}
	return evalExprLeaf(expr, vars)
}

// evalExprLeaf is template evaluation for one expression, excluding + / * at
// the top level (so startup.tmpl pixel arithmetic does not mangle "gg_cup" etc.).
func evalExprLeaf(expr string, vars map[string]string) string {
	expr = strings.TrimSpace(expr)
	if q := unquote(expr); q != nil {
		return *q
	}
	if i := strings.Index(expr, "?"); i > 0 && strings.Contains(expr[i+1:], ":") {
		cond := strings.TrimSpace(expr[:i])
		rest := strings.TrimSpace(expr[i+1:])
		j := strings.Index(rest, ":")
		if j > 0 {
			left := strings.TrimSpace(rest[:j])
			right := strings.TrimSpace(rest[j+1:])
			if evalCondition(cond, vars) {
				return evalExpr(left, vars)
			}
			return evalExpr(right, vars)
		}
	}
	if strings.Contains(expr, "==") {
		if evalCondition(expr, vars) {
			return "1"
		}
		return ""
	}
	if strings.HasSuffix(expr, ".length") {
		inner := strings.TrimSpace(strings.TrimSuffix(expr, ".length"))
		if inner != "" {
			s := evalExpr(inner, vars)
			if s == "" {
				return "0"
			}
			return strconv.Itoa(utf8.RuneCountInString(s))
		}
	}
	if strings.HasPrefix(expr, "POSIX.floor(") && strings.HasSuffix(expr, ")") {
		inner := strings.TrimSuffix(strings.TrimPrefix(expr, "POSIX.floor("), ")")
		v := evalExpr(inner, vars)
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return strconv.Itoa(int(f))
		}
		return "0"
	}
	if expr == "h.req.ver" {
		return vars["ver"]
	}
	if strings.HasPrefix(expr, "server.config.") {
		return lookupVar(strings.TrimPrefix(expr, "server.config."), vars)
	}
	if strings.HasPrefix(expr, "P.") {
		return lookupVar(strings.TrimPrefix(expr, "P."), vars)
	}
	if _, err := strconv.ParseFloat(strings.TrimSpace(expr), 64); err == nil {
		return strings.TrimSpace(expr)
	}
	if strings.Contains(expr, " _ ") {
		parts := strings.Split(expr, " _ ")
		var b strings.Builder
		for _, p := range parts {
			b.WriteString(evalExpr(p, vars))
		}
		return b.String()
	}
	return lookupVar(expr, vars)
}

func hasTopLevelOp(s string, op rune) bool {
	depth := 0
	for _, r := range s {
		switch r {
		case '(':
			depth++
		case ')':
			if depth > 0 {
				depth--
			}
		}
		if depth == 0 && r == op {
			return true
		}
	}
	return false
}

func splitTopLevelOp(s string, op rune) []string {
	depth := 0
	var parts []string
	start := 0
	for i, r := range s {
		switch r {
		case '(':
			depth++
		case ')':
			if depth > 0 {
				depth--
			}
		}
		if depth == 0 && r == op {
			if i > start {
				parts = append(parts, strings.TrimSpace(s[start:i]))
			}
			start = i + utf8.RuneLen(r)
		}
	}
	if start <= len(s) {
		if tail := strings.TrimSpace(s[start:]); tail != "" {
			parts = append(parts, tail)
		} else if len(parts) == 0 {
			parts = append(parts, "")
		}
	}
	return parts
}

func tryEvalAddMul(expr string, vars map[string]string) (string, bool) {
	if !hasTopLevelOp(expr, '+') && !hasTopLevelOp(expr, '*') {
		return "", false
	}
	addends := splitTopLevelOp(expr, '+')
	if len(addends) == 0 {
		return "", false
	}
	var sum float64
	for _, addend := range addends {
		if addend == "" {
			continue
		}
		if !hasTopLevelOp(addend, '*') {
			sum += evalArithAtom(addend, vars)
			continue
		}
		prod := 1.0
		for _, f := range splitTopLevelOp(addend, '*') {
			prod *= evalArithAtom(f, vars)
		}
		sum += prod
	}
	// Coordinates in show bodies are integral; follow TT by truncating toward 0.
	return strconv.FormatInt(int64(math.Trunc(sum)), 10), true
}

// evalArithAtom parses a single factor: literal number or a leaf expression
// (variables, .length, POSIX.floor, no top-level +*).
func evalArithAtom(expr string, vars map[string]string) float64 {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return 0
	}
	if f, err := strconv.ParseFloat(expr, 64); err == nil {
		return f
	}
	s := evalExprLeaf(expr, vars)
	if s == "" {
		return 0
	}
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return f
	}
	// e.g. length-like digit strings
	if f, err := strconv.ParseFloat(strings.TrimSpace(s), 64); err == nil {
		return f
	}
	return 0
}

func normalizeTT(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "~")
	s = strings.TrimSuffix(s, "~")
	return strings.TrimSpace(s)
}

func lookupVar(name string, vars map[string]string) string {
	name = strings.TrimSpace(name)
	switch name {
	case "id":
		return vars["id"]
	case "nick":
		return vars["nick"]
	case "NICK":
		return vars["nick"]
	case "error_text":
		return vars["error_text"]
	case "chat_server":
		return vars["chat_server"]
	case "logged_in":
		return vars["logged_in"]
	case "type":
		return vars["type"]
	case "window_size":
		return vars["window_size"]
	case "table_timeout":
		return vars["table_timeout"]
	case "ver":
		return vars["ver"]
	case "header":
		return vars["header"]
	case "text":
		return vars["text"]
	case "ok_text":
		return vars["ok_text"]
	case "height":
		return vars["height"]
	case "command":
		return vars["command"]
	case "ip":
		return vars["ip"]
	case "port":
		return vars["port"]
	case "max_pl":
		return vars["max_pl"]
	case "name":
		return vars["name"]
	case "active_players":
		return vars["active_players"]
	case "exited_players":
		return vars["exited_players"]
	case "has_exited_players":
		return vars["has_exited_players"]
	case "room_players_start":
		return vars["room_players_start"]
	default:
		return vars[name]
	}
}

func unquote(s string) *string {
	if len(s) >= 2 && ((s[0] == '\'' && s[len(s)-1] == '\'') || (s[0] == '"' && s[len(s)-1] == '"')) {
		v := s[1 : len(s)-1]
		return &v
	}
	return nil
}

func compareNum(aRaw, bRaw string, cmp func(a, b float64) bool) bool {
	a, errA := strconv.ParseFloat(strings.TrimSpace(aRaw), 64)
	b, errB := strconv.ParseFloat(strings.TrimSpace(bRaw), 64)
	if errA != nil || errB != nil {
		return false
	}
	return cmp(a, b)
}

func renderInlineIfBlocks(src string, vars map[string]string) string {
	reElse := regexp.MustCompile(`(?s)<\?\s*IF\s+(.+?)\s*\?>(.*?)<\?\s*ELSE\s*\?>(.*?)<\?\s*END\s*\?>`)
	for reElse.MatchString(src) {
		src = reElse.ReplaceAllStringFunc(src, func(m string) string {
			sub := reElse.FindStringSubmatch(m)
			if len(sub) != 4 {
				return m
			}
			if evalCondition(sub[1], vars) {
				return sub[2]
			}
			return sub[3]
		})
	}
	reNoElse := regexp.MustCompile(`(?s)<\?\s*IF\s+(.+?)\s*\?>(.*?)<\?\s*END\s*\?>`)
	for reNoElse.MatchString(src) {
		src = reNoElse.ReplaceAllStringFunc(src, func(m string) string {
			sub := reNoElse.FindStringSubmatch(m)
			if len(sub) != 3 {
				return m
			}
			if evalCondition(sub[1], vars) {
				return sub[2]
			}
			return ""
		})
	}
	return src
}
