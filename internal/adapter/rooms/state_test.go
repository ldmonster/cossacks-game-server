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

package rooms

import (
	"sync"
	"testing"
)

func TestNextRoomIDIsMonotonicAndRaceSafe(t *testing.T) {
	s := NewStore()
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = s.NextRoomID()
		}()
	}
	wg.Wait()
	if got := s.LastRoomID(); got != 50 {
		t.Fatalf("LastRoomID=%d want 50", got)
	}
}

func TestRoomControlSumSample(t *testing.T) {
	row := []string{"12", "#", "Test Room", "Nick", "For all", "1/8", "2"}
	const expected uint32 = 2057570483
	got := RoomControlSum(row)
	if got != expected {
		t.Fatalf("room checksum mismatch: got=%d want=%d", got, expected)
	}
}
