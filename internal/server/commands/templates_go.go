package commands

import (
	"os"
	"path/filepath"
	"strings"
)

func normalizeShowTemplateName(name string) string {
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

func fallbackShowBody() string {
	return "#font(WF,WF,WF)\n#txt(%BOX[x:10,y:10,w:100%,h:24],{},\"server response\")"
}

// loadShowBody renders an LW_show payload from {root}/cs|ac/{name}.tmpl using the
// TT-style fragment engine (renderShowTemplate).
func loadShowBody(ver uint8, templateName string, vars map[string]string) string {
	name := normalizeShowTemplateName(templateName)
	if name == "" {
		return fallbackShowBody()
	}
	dir := "cs"
	if isAC(ver) {
		dir = "ac"
	}
	for _, root := range effectiveTemplateRoots() {
		path := filepath.Join(root, dir, filepath.FromSlash(name))
		b, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		return renderShowTemplate(string(b), vars)
	}
	return fallbackShowBody()
}
