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
			got := BuildTemplateRoots(tc.custom, tc.defaults)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("BuildTemplateRoots(%q, %v)=%v want %v", tc.custom, tc.defaults, got, tc.want)
			}
		})
	}
}
