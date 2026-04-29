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

import "testing"

func TestTimeIntervalFromElapsedSecMatchesPerlLayout(t *testing.T) {
	// Open::_time_interval: same day/hour rules as the earlier Cossacks server.
	cases := []struct {
		secs int
		want string
	}{
		{0, "0s"},
		{3, "3s"},
		{65, "1m 5s"},
		{125, "2m 5s"},
		{600, "10m"},
		{700, "11m"}, // 11m 40s — reference returns early at m>=10, drops seconds
		{3600, "1h"},
		{7500, "2h 5m"},
		{86400, "1d"},
		{90000, "1d 1h"},
		{86400 + 3600 + 300, "1d 1h"}, // 90300: 1d and leftover hours
		{86400 + 5*60, "1d"},          // 86700: 1d + 5m, reference still returns "1d" on first d-return
	}
	for _, tc := range cases {
		if got := timeIntervalFromElapsedSec(tc.secs); got != tc.want {
			t.Errorf("timeIntervalFromElapsedSec(%d) = %q, want %q", tc.secs, got, tc.want)
		}
	}
}
