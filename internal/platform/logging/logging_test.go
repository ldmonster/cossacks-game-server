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

package logging

import "testing"

func TestParseFormat(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want Format
	}{
		{"", FormatUser},
		{"user", FormatUser},
		{"USER", FormatUser},
		{"json", FormatJSON},
		{"JSON", FormatJSON},
	} {
		got, err := ParseFormat(tc.in)
		if err != nil {
			t.Fatalf("ParseFormat(%q): %v", tc.in, err)
		}
		if got != tc.want {
			t.Fatalf("ParseFormat(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}

	if _, err := ParseFormat("xml"); err == nil {
		t.Fatal("expected error for invalid format")
	}
}
