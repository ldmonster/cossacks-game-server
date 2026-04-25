package commands

import "testing"

func TestTimeIntervalFromElapsedSecMatchesPerlLayout(t *testing.T) {
	// Open::_time_interval: same day/hour rules as the legacy Cossacks server.
	cases := []struct {
		secs int
		want string
	}{
		{0, "0s"},
		{3, "3s"},
		{65, "1m 5s"},
		{125, "2m 5s"},
		{600, "10m"},
		{700, "11m"},      // 11m 40s — Perl returns early at m>=10, drops seconds
		{3600, "1h"},
		{7500, "2h 5m"},
		{86400, "1d"},
		{90000, "1d 1h"},
		{86400 + 3600 + 300, "1d 1h"},   // 90300: 1d and leftover hours, TT matches Perl
		{86400 + 5*60, "1d"},            // 86700: 1d + 5m, Perl still returns "1d" on first d-return
	}
	for _, tc := range cases {
		if got := timeIntervalFromElapsedSec(tc.secs); got != tc.want {
			t.Errorf("timeIntervalFromElapsedSec(%d) = %q, want %q", tc.secs, got, tc.want)
		}
	}
}
