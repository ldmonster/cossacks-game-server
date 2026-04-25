package state

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
	if s.LastRoomID != 50 {
		t.Fatalf("LastRoomID=%d want 50", s.LastRoomID)
	}
}

func TestRoomControlSumPerlParitySample(t *testing.T) {
	row := []string{"12", "#", "Test Room", "Nick", "For all", "1/8", "2"}
	const expected uint32 = 2057570483
	got := RoomControlSum(row)
	if got != expected {
		t.Fatalf("room checksum mismatch: got=%d want=%d", got, expected)
	}
}
