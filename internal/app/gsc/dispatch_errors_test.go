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
	"context"
	"errors"
	"testing"

	"github.com/ldmonster/cossacks-game-server/internal/domain/session"
	tconn "github.com/ldmonster/cossacks-game-server/internal/transport/tconn"
	"github.com/ldmonster/cossacks-game-server/internal/transport/gsc"
)

func TestHandleWithMetaUnknownCommandPopulatesErr(t *testing.T) {
	c := newControllerForJoinTests()
	conn := &tconn.Connection{Session: &session.Session{}}
	req := &gsc.Stream{Ver: 2}

	r := c.HandleWithMeta(
		context.Background(), conn, req, "no_such_command", nil, "", "",
	)

	var typed errUnknownCommand
	if !errors.As(r.Err, &typed) {
		t.Fatalf("expected errUnknownCommand, got %T (%v)", r.Err, r.Err)
	}

	if !r.HasResponse {
		t.Fatalf("unknown command must still emit empty response")
	}

	if len(r.Commands) != 0 {
		t.Fatalf("unknown command should emit empty command set, got %#v", r.Commands)
	}
}

func TestHandleWithMetaUnknownOpenRoutePopulatesErr(t *testing.T) {
	c := newControllerForJoinTests()
	conn := &tconn.Connection{Session: &session.Session{}}
	req := &gsc.Stream{Ver: 2}

	r := c.HandleWithMeta(
		context.Background(), conn, req, "open",
		[]string{"definitely_not_a_route.dcml", ""}, "", "",
	)

	var typed errUnknownOpenRoute
	if !errors.As(r.Err, &typed) {
		t.Fatalf("expected errUnknownOpenRoute, got %T (%v)", r.Err, r.Err)
	}

	if !r.HasResponse || len(r.Commands) == 0 {
		t.Fatalf("unknown open route must emit Page Not Found alert")
	}
}

func TestHandleWithMetaURLNoArgPopulatesErr(t *testing.T) {
	c := newControllerForJoinTests()
	conn := &tconn.Connection{Session: &session.Session{}}
	req := &gsc.Stream{Ver: 2}

	r := c.HandleWithMeta(
		context.Background(), conn, req, "url", nil, "", "",
	)

	var typed errURLNoArg
	if !errors.As(r.Err, &typed) {
		t.Fatalf("expected errURLNoArg, got %T (%v)", r.Err, r.Err)
	}

	if r.HasResponse {
		t.Fatalf("url without arg must emit no response")
	}
}
