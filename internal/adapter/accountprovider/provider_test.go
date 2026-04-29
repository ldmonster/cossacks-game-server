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

package httpauth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ldmonster/cossacks-game-server/internal/domain/identity"
)

func TestAuthenticateSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		if r.PostForm.Get("login") != "alice" || r.PostForm.Get("key") != "secret" {
			http.Error(w, "bad", http.StatusBadRequest)
			return
		}
		_, _ = w.Write([]byte(`{"success":true,"id":42,"profile":"http://x/y"}`))
	}))
	defer srv.Close()

	p := New(HostMap{
		identity.AccountTypeLCN: {Host: stripScheme(srv.URL), Key: "secret"},
	}, srv.Client())

	got, err := p.Authenticate(context.Background(), identity.AccountTypeLCN, "alice", "pw")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}

	if got.Login != "alice" || got.ID != "42" || got.Profile != "http://x/y" || got.Type != identity.AccountTypeLCN {
		t.Fatalf("unexpected info: %+v", got)
	}
}

func TestAuthenticateBadCredentials(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"success":false}`))
	}))
	defer srv.Close()

	p := New(HostMap{
		identity.AccountTypeLCN: {Host: stripScheme(srv.URL), Key: "secret"},
	}, srv.Client())

	_, err := p.Authenticate(context.Background(), identity.AccountTypeLCN, "alice", "wrong")
	if !errors.Is(err, ErrBadCredentials) {
		t.Fatalf("want ErrBadCredentials, got %v", err)
	}
}

func TestAuthenticateUnknownType(t *testing.T) {
	p := New(HostMap{}, http.DefaultClient)
	_, err := p.Authenticate(context.Background(), identity.AccountTypeLCN, "x", "y")
	if !errors.Is(err, ErrServiceUnavailable) {
		t.Fatalf("want ErrServiceUnavailable, got %v", err)
	}
}

func stripScheme(u string) string {
	for _, prefix := range []string{"http://", "https://"} {
		if len(u) > len(prefix) && u[:len(prefix)] == prefix {
			return u[len(prefix):]
		}
	}
	return u
}
