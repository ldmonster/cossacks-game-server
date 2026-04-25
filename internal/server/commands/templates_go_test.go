package commands

import (
	"strings"
	"testing"
)

func TestLoadShowBodyUsesTmplUnderRoots(t *testing.T) {
	ensureShareTemplateRootsForFullbody(t)
	vars := varsForShareTemplateGolden("cs/alert_dgl.tmpl", 2)
	want := strings.TrimSpace(loadShowBody(2, "alert_dgl.tmpl", vars))
	if want == "" || strings.Contains(want, "server response") && strings.Contains(want, "%BOX[x:10,y:10") {
		t.Fatalf("expected rendered alert_dgl.tmpl, got: %q", want)
	}
}

func TestNormalizeShowTemplateName(t *testing.T) {
	if got := normalizeShowTemplateName("alert_dgl.cml"); got != "alert_dgl.tmpl" {
		t.Fatalf("cml suffix: got %q", got)
	}
	if got := normalizeShowTemplateName("x/y.z.tmpl"); got != "x/y.z.tmpl" {
		t.Fatalf("tmpl passthrough: got %q", got)
	}
	if got := normalizeShowTemplateName("enter"); got != "enter.tmpl" {
		t.Fatalf("bare name: got %q", got)
	}
}
