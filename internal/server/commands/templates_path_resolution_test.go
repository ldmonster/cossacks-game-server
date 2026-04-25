package commands

import (
	"reflect"
	"testing"
)

func TestBuildTemplateRoots(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		custom   string
		defaults []string
		want     []string
	}{
		{
			name:     "custom prepended",
			custom:   "/custom/share",
			defaults: []string{"/a", "/b"},
			want:     []string{"/custom/share", "/a", "/b"},
		},
		{
			name:     "custom empty keeps defaults",
			custom:   "",
			defaults: []string{"/a", "/b"},
			want:     []string{"/a", "/b"},
		},
		{
			name:     "dedupe custom equals default",
			custom:   "/a",
			defaults: []string{"/a", "/b"},
			want:     []string{"/a", "/b"},
		},
		{
			name:     "trim whitespace",
			custom:   "  /x  ",
			defaults: []string{"/a"},
			want:     []string{"/x", "/a"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := buildTemplateRoots(tc.custom, tc.defaults)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("buildTemplateRoots(%q, %v)=%v want %v", tc.custom, tc.defaults, got, tc.want)
			}
		})
	}
}

