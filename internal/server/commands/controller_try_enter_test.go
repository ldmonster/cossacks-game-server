package commands

import (
	"strings"
	"testing"

	"cossacksgameserver/golang/internal/protocol/gsc"
	"cossacksgameserver/golang/internal/server/model"
)

func TestEnterUsesAccountLoggedInView(t *testing.T) {
	c := newControllerForJoinTests()
	req := &gsc.Stream{Ver: 2}
	conn := &model.Connection{
		Data: map[string]any{
			"account": map[string]string{
				"type":  "LCN",
				"login": "AccountNick",
				"id":    "123",
			},
		},
	}
	out := c.dispatchOpen(nil, conn, req, "enter", map[string]string{})
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
	conn := &model.Connection{
		Data: map[string]any{
			"account": map[string]string{
				"type":  "LCN",
				"login": "Nick",
				"id":    "5",
			},
		},
	}
	_ = c.tryEnter(nil, conn, req, map[string]string{"RESET": "1"})
	if _, ok := conn.Data["account"]; ok {
		t.Fatalf("expected account to be cleared on RESET")
	}
}

func TestTryEnterLoggedInWithoutAccountFallsBackToEnter(t *testing.T) {
	c := newControllerForJoinTests()
	req := &gsc.Stream{Ver: 2}
	conn := &model.Connection{Data: map[string]any{}}
	out := c.tryEnter(nil, conn, req, map[string]string{"LOGGED_IN": "1"})
	if len(out) != 1 {
		t.Fatalf("expected one command, got %#v", out)
	}
	if !strings.Contains(out[0].Args[0], "Your nick:") {
		t.Fatalf("expected enter view fallback, got: %q", out[0].Args[0])
	}
}

func TestTryEnterLcnValidationMessages(t *testing.T) {
	c := newControllerForJoinTests()
	req := &gsc.Stream{Ver: 2}
	conn := &model.Connection{Data: map[string]any{}}

	out1 := c.tryEnter(nil, conn, req, map[string]string{"TYPE": "LCN"})
	if len(out1) != 1 || !strings.Contains(out1[0].Args[0], "enter nick") {
		t.Fatalf("expected 'enter nick' message, got %#v", out1)
	}

	out2 := c.tryEnter(nil, conn, req, map[string]string{"TYPE": "LCN", "NICK": "abc"})
	if len(out2) != 1 || !strings.Contains(out2[0].Args[0], "enter password") {
		t.Fatalf("expected 'enter password' message, got %#v", out2)
	}
}
