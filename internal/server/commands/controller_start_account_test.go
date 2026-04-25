package commands

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"cossacksgameserver/golang/internal/server/model"
	"cossacksgameserver/golang/internal/server/state"
)

func TestStartPostsAccountActionPayload(t *testing.T) {
	var posted url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		posted = r.PostForm
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	host := strings.TrimPrefix(srv.URL, "http://")

	c := newControllerForJoinTests()
	c.HTTP = srv.Client()
	c.Config.Raw["lcn_host"] = host
	c.Config.Raw["lcn_key"] = "secret"

	now := time.Now().UTC()
	hostPlayer := &state.Player{
		ID:          1,
		Nick:        "host",
		ConnectedAt: now.Add(-10 * time.Minute),
		Account: map[string]any{
			"type": "LCN",
			"id":   "42",
		},
	}
	guestPlayer := &state.Player{
		ID:          2,
		Nick:        "guest",
		ConnectedAt: now.Add(-5 * time.Minute),
	}
	c.Store.Players[1] = hostPlayer
	c.Store.Players[2] = guestPlayer
	room := makeRoom(c, 301, 1, "start-room", "")
	room.Level = 2
	room.Players[2] = guestPlayer
	room.PlayersCount = 2
	c.Store.RoomsByPID[1] = room

	conn := &model.Connection{
		IP: "127.0.0.1",
		Data: map[string]any{
			"id": uint32(1),
			"account": map[string]string{
				"type":  "LCN",
				"login": "host",
				"id":    "42",
			},
		},
	}
	_ = c.handleStart(conn, nil, []string{
		"sav:[12]",
		"random.m3d",
		"2",
		"1", "3", "1", "4",
		"2", "5", "2", "6",
	})
	if posted.Get("action") != "start" {
		t.Fatalf("expected action=start, got %#v", posted)
	}
	if posted.Get("account_id") != "42" || posted.Get("key") != "secret" || posted.Get("time") == "" {
		t.Fatalf("expected account_id, key, and time, got %#v", posted)
	}
	payload := posted.Get("data")
	if !strings.Contains(payload, "\"map\":\"random.m3d\"") {
		t.Fatalf("expected map in payload, got %s", payload)
	}
	if !strings.Contains(payload, "\"save_from\":12") {
		t.Fatalf("expected save_from in payload, got %s", payload)
	}
	if !strings.Contains(payload, "\"players\"") || !strings.Contains(payload, "\"nation\":3") {
		t.Fatalf("expected players list in payload, got %s", payload)
	}
}
