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

package connectivity

import (
	"sync"
	"testing"

	"github.com/ldmonster/cossacks-game-server/internal/domain/player"
	"github.com/ldmonster/cossacks-game-server/internal/domain/session"
	"github.com/ldmonster/cossacks-game-server/internal/port"
)

// Compile-time guarantee Store satisfies the port contract.
var _ port.SessionStore = (*Store)(nil)

func TestStoreGetSetDelete(t *testing.T) {
	s := NewStore()

	if _, ok := s.Get(player.ConnectionID(1)); ok {
		t.Fatalf("empty store: expected miss")
	}

	sess := &session.Session{WindowW: 1024, WindowH: 768}
	s.Set(player.ConnectionID(1), sess)

	got, ok := s.Get(player.ConnectionID(1))
	if !ok || got != sess {
		t.Fatalf("get after set: ok=%v got=%v want=%v", ok, got, sess)
	}

	s.Delete(player.ConnectionID(1))
	if _, ok := s.Get(player.ConnectionID(1)); ok {
		t.Fatalf("after delete: expected miss")
	}

	// Delete of unknown id must not panic.
	s.Delete(player.ConnectionID(999))
}

func TestStoreConcurrentAccess(t *testing.T) {
	s := NewStore()
	const n = 64

	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func(id int) {
			defer wg.Done()
			cid := player.ConnectionID(id)
			s.Set(cid, &session.Session{})
			if _, ok := s.Get(cid); !ok {
				t.Errorf("missing session %d", id)
			}
			s.Delete(cid)
		}(i)
	}
	wg.Wait()
}
