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

import (
	"fmt"
	"strings"
)

// TimeIntervalFromElapsedSec formats `secs` as a human-readable
// duration used by room/started-room template renderers.
//
// The rules:
//   - Always show days when present, then hours; if days were shown,
//     stop there.
//   - Otherwise show minutes; if hours were shown or minutes >= 10,
//     stop there.
//   - Otherwise add seconds; never include a "0s" component unless
//     the total elapsed time is zero.
//   - Negative input is normalised to 0.
func TimeIntervalFromElapsedSec(secs int) string {
	if secs < 0 {
		secs = 0
	}

	t := secs
	d := t / 86400
	t %= 86400
	h := t / 3600
	t %= 3600

	var parts []string
	if d > 0 {
		parts = append(parts, fmt.Sprintf("%dd", d))
	}

	if h > 0 {
		parts = append(parts, fmt.Sprintf("%dh", h))
	}

	if d > 0 {
		return strings.Join(parts, " ")
	}

	m := t / 60
	t %= 60

	if m > 0 {
		parts = append(parts, fmt.Sprintf("%dm", m))
	}

	if h > 0 || m >= 10 {
		return strings.Join(parts, " ")
	}

	if t > 0 {
		parts = append(parts, fmt.Sprintf("%ds", t))
	}

	if len(parts) > 0 {
		return strings.Join(parts, " ")
	}

	return "0s"
}
