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
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/ldmonster/cossacks-game-server/internal/domain/identity"
	"github.com/ldmonster/cossacks-game-server/internal/domain/player"
	"github.com/ldmonster/cossacks-game-server/internal/domain/session"
	tconn "github.com/ldmonster/cossacks-game-server/internal/transport/tconn"
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
	c.Auth.Provider("LCN").Host = host
	c.Auth.Provider("LCN").Key = "secret"

	now := time.Now().UTC()
	hostPlayer := &player.Player{
		ID:          1,
		Nick:        "host",
		ConnectedAt: now.Add(-10 * time.Minute),
		AccountType: "LCN",
		AccountID:   "42",
	}
	guestPlayer := &player.Player{
		ID:          2,
		Nick:        "guest",
		ConnectedAt: now.Add(-5 * time.Minute),
	}
	c.Store.SetPlayer(hostPlayer)
	c.Store.SetPlayer(guestPlayer)
	room := makeRoom(c, 301, 1, "start-room", "")
	room.Level = 2
	room.Players[2] = guestPlayer
	room.PlayersCount = 2
	c.Store.IndexRoomByHost(1, room)

	conn := &tconn.Connection{
		IP: "127.0.0.1",
		Session: &session.Session{
			PlayerID: 1,
			Account: &identity.AccountInfo{
				Type:  "LCN",
				Login: "host",
				ID:    "42",
			},
		},
	}
	c.handleStart(conn, []string{
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
