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

package render

import "testing"

func TestTimeIntervalFromElapsedSec(t *testing.T) {
	cases := []struct {
		secs int
		want string
	}{
		{-5, "0s"},
		{0, "0s"},
		{1, "1s"},
		{45, "45s"},
		{60, "1m"},
		{61, "1m 1s"},
		{599, "9m 59s"},
		{600, "10m"},
		{605, "10m"},
		{3600, "1h"},
		{3661, "1h 1m"},
		{86400, "1d"},
		{90061, "1d 1h"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.want, func(t *testing.T) {
			if got := TimeIntervalFromElapsedSec(tc.secs); got != tc.want {
				t.Errorf("TimeIntervalFromElapsedSec(%d) = %q, want %q", tc.secs, got, tc.want)
			}
		})
	}
}
