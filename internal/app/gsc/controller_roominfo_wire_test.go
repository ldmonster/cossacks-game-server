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

import "testing"

func TestNormalizePage(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"", "1"},
		{"1", "1"},
		{"2", "2"},
		{"3", "3"},
		{" 2 ", "2"},
		{"0", "1"},
		{"4", "1"},
		{"abc", "1"},
	}
	for _, tc := range cases {
		if got := normalizePage(tc.in); got != tc.want {
			t.Fatalf("normalizePage(%q)=%q want %q", tc.in, got, tc.want)
		}
	}
}

func TestNormalizeRes(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"", "0"},
		{"0", "0"},
		{"1", "1"},
		{" 5 ", "5"},
		{"abc", "0"},
		{"-1", "0"},
	}
	for _, tc := range cases {
		if got := normalizeRes(tc.in); got != tc.want {
			t.Fatalf("normalizeRes(%q)=%q want %q", tc.in, got, tc.want)
		}
	}
}
