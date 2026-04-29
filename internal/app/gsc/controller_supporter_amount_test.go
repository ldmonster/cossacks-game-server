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

import (
	"encoding/json"
	"testing"
)

func TestSupporterAmountString(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		in   any
		want string
	}{
		{"int", 42, "42"},
		{"int64", int64(7), "7"},
		{"float64", float64(100), "100"},
		{"json int", json.Number("99"), "99"},
		{"json float token int64", json.Number("12.0"), "12"},
		{"json non-integer", json.Number("1.5e2"), "150"},
		{"json invalid falls back to string", json.Number("not-a-number"), "not-a-number"},
		{"string numeric", "33", "33"},
		{"string empty", "", "0"},
		{"nil formats as v default", nil, "<nil>"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := supporterAmountString(tc.in); got != tc.want {
				t.Errorf("supporterAmountString(%v) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestAnyToStringVar(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		in   any
		want string
	}{
		{"string", "x", "x"},
		{"bool true", true, "1"},
		{"bool false", false, "0"},
		{"float int-like", float64(3), "3"},
		{"float fractional", 1.25, "1.25"},
		{"unhandled type", struct{ A int }{1}, "{1}"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := anyToStringVar(tc.in); got != tc.want {
				t.Errorf("anyToStringVar(%v) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}
