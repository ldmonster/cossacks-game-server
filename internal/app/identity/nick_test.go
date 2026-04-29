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

package identity

import "testing"

func TestValidateNick(t *testing.T) {
	cases := []struct {
		name string
		nick string
		want NickError
	}{
		{"empty", "", NickEmpty},
		{"plain ascii", "Player", NickOK},
		{"with brackets and underscore", "[Clan]_Hero-1", NickOK},
		{"space rejected", "Player One", NickBadCharacter},
		{"dot rejected", "player.one", NickBadCharacter},
		{"leading dash", "-vader", NickStartsWithDash},
		{"leading digit", "1up", NickStartsWithDigit},
		{"leading bracket ok", "[clan]", NickOK},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			if got := ValidateNick(tc.nick); got != tc.want {
				t.Errorf("ValidateNick(%q) = %v, want %v", tc.nick, got, tc.want)
			}
		})
	}
}

func TestNickErrorMessageNonEmpty(t *testing.T) {
	for _, e := range []NickError{NickEmpty, NickBadCharacter, NickStartsWithDigit, NickStartsWithDash} {
		if e.Message() == "" {
			t.Errorf("NickError(%d).Message() must be non-empty", e)
		}
	}
	if NickOK.Message() != "" {
		t.Errorf("NickOK.Message() must be empty, got %q", NickOK.Message())
	}
}

func TestTruncateNickRespectsCap(t *testing.T) {
	if got := TruncateNick("abc"); got != "abc" {
		t.Errorf("TruncateNick(short) = %q, want unchanged", got)
	}
	long := "abcdefghijklmnopqrstuvwxyz0123456789"
	got := TruncateNick(long)
	if len(got) != MaxNickLen {
		t.Errorf("TruncateNick(long) len = %d, want %d", len(got), MaxNickLen)
	}
	if got != long[:MaxNickLen] {
		t.Errorf("TruncateNick(long) = %q, want prefix", got)
	}
}

func TestSanitizeAccountNick(t *testing.T) {
	cases := []struct {
		login string
		want  string
	}{
		{"alice", "alice"},
		{"user@example.com", "userexamplecom"},
		{"@@@", "player"},
		{"", "player"},
		{"42player", "_42player"},
		{"[clan]_hero-1", "[clan]_hero-1"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.login, func(t *testing.T) {
			if got := SanitizeAccountNick(tc.login); got != tc.want {
				t.Errorf("SanitizeAccountNick(%q) = %q, want %q", tc.login, got, tc.want)
			}
		})
	}
}
