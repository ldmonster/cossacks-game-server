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
	"strings"
	"testing"

	"github.com/ldmonster/cossacks-game-server/internal/domain/identity"
	"github.com/ldmonster/cossacks-game-server/internal/domain/session"
	tconn "github.com/ldmonster/cossacks-game-server/internal/transport/tconn"
	"github.com/ldmonster/cossacks-game-server/internal/transport/gsc"
)

func TestEnterUsesAccountLoggedInView(t *testing.T) {
	c := newControllerForJoinTests()
	req := &gsc.Stream{Ver: 2}
	conn := &tconn.Connection{
		Session: &session.Session{
			Account: &identity.AccountInfo{
				Type:  "LCN",
				Login: "AccountNick",
				ID:    "123",
			},
		},
	}
	out, _ := c.dispatchOpen(nil, conn, req, "enter", map[string]string{})
	if len(out) != 1 {
		t.Fatalf("expected one command, got %#v", out)
	}
	body := out[0].Args[0]
	if !strings.Contains(body, "logout") {
		t.Fatalf("expected logged-in enter variant with logout, got: %q", body)
	}
}

func TestTryEnterResetClearsAccount(t *testing.T) {
	c := newControllerForJoinTests()
	req := &gsc.Stream{Ver: 2}
	conn := &tconn.Connection{
		Session: &session.Session{
			Account: &identity.AccountInfo{
				Type:  "LCN",
				Login: "Nick",
				ID:    "5",
			},
		},
	}
	_, _ = c.tryEnter(nil, conn, req, map[string]string{"RESET": "1"})
	if conn.Session != nil && conn.Session.Account != nil {
		t.Fatalf("expected account to be cleared on RESET")
	}
}

func TestTryEnterLoggedInWithoutAccountFallsBackToEnter(t *testing.T) {
	c := newControllerForJoinTests()
	req := &gsc.Stream{Ver: 2}
	conn := &tconn.Connection{Session: &session.Session{}}
	out, _ := c.tryEnter(nil, conn, req, map[string]string{"LOGGED_IN": "1"})
	if len(out) != 1 {
		t.Fatalf("expected one command, got %#v", out)
	}
	if !strings.Contains(out[0].Args[0], "Your nickname:") {
		t.Fatalf("expected enter view fallback, got: %q", out[0].Args[0])
	}
}

func TestTryEnterLcnValidationMessages(t *testing.T) {
	c := newControllerForJoinTests()
	req := &gsc.Stream{Ver: 2}
	conn := &tconn.Connection{Session: &session.Session{}}

	out1, _ := c.tryEnter(nil, conn, req, map[string]string{"TYPE": "LCN"})
	if len(out1) != 1 || !strings.Contains(out1[0].Args[0], "enter nick") {
		t.Fatalf("expected 'enter nick' message, got %#v", out1)
	}

	out2, _ := c.tryEnter(nil, conn, req, map[string]string{"TYPE": "LCN", "NICK": "abc"})
	if len(out2) != 1 || !strings.Contains(out2[0].Args[0], "enter password") {
		t.Fatalf("expected 'enter password' message, got %#v", out2)
	}
}
