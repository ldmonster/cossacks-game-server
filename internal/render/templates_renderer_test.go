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
	"os"
	"path/filepath"
	"testing"

	"github.com/ldmonster/cossacks-game-server/internal/port"
)

// Compile-time check: *TemplateRenderer satisfies port.TemplateRenderer.
var _ port.TemplateRenderer = (*TemplateRenderer)(nil)

func TestNewTemplateRendererPutsCustomRootFirst(t *testing.T) {
	r := NewTemplateRenderer("/tmp/custom")
	roots := r.Roots()

	if len(roots) == 0 || roots[0] != "/tmp/custom" {
		t.Fatalf("custom root not first: %v", roots)
	}

	for _, want := range DefaultTemplateRoots {
		found := false
		for _, got := range roots {
			if got == want {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("default root %q dropped from %v", want, roots)
		}
	}
}

func TestNewTemplateRendererEmptyCustomFallsBackToDefaults(t *testing.T) {
	r := NewTemplateRenderer("")
	roots := r.Roots()

	if len(roots) != len(DefaultTemplateRoots) {
		t.Fatalf("expected %d defaults, got %d (%v)", len(DefaultTemplateRoots), len(roots), roots)
	}
}

func TestNewTemplateRendererDeduplicates(t *testing.T) {
	r := NewTemplateRenderer(DefaultTemplateRoots[0])
	roots := r.Roots()

	if len(roots) != len(DefaultTemplateRoots) {
		t.Fatalf("duplicate root not deduped: got %v", roots)
	}
}

func TestTemplateRendererRootsIsCopy(t *testing.T) {
	r := NewTemplateRenderer("/tmp/x")
	a := r.Roots()
	a[0] = "MUTATED"
	b := r.Roots()

	if b[0] == "MUTATED" {
		t.Fatalf("Roots returned shared slice: %v", b)
	}
}

func TestTemplateRendererIsolatedFromDefaults(t *testing.T) {
	// Renderer construction must not mutate the package-level immutable
	// `DefaultTemplateRoots` slice.
	before := append([]string(nil), DefaultTemplateRoots...)
	_ = NewTemplateRenderer("/tmp/isolated")

	if len(before) != len(DefaultTemplateRoots) {
		t.Fatalf("defaults mutated: before=%v after=%v", before, DefaultTemplateRoots)
	}
	for i := range before {
		if before[i] != DefaultTemplateRoots[i] {
			t.Fatalf("defaults mutated at %d: before=%q after=%q", i, before[i], DefaultTemplateRoots[i])
		}
	}
}

func TestTemplateRendererRenderFindsTemplateInCustomRoot(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "cs"), 0o755); err != nil {
		t.Fatal(err)
	}
	body := "#font(WF,WF,WF)\nHELLO_TMPL"
	if err := os.WriteFile(filepath.Join(dir, "cs", "renderer_test.tmpl"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	r := NewTemplateRenderer(dir)
	got := r.Render(0, "renderer_test", nil)

	if got != body {
		t.Fatalf("Render mismatch:\n got: %q\nwant: %q", got, body)
	}
}

func TestTemplateRendererRenderFallbackOnMissing(t *testing.T) {
	r := NewTemplateRenderer(t.TempDir())
	got := r.Render(0, "definitely_does_not_exist", nil)

	if got != FallbackShowBody() {
		t.Fatalf("expected fallback body, got %q", got)
	}
}
