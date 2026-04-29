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

package lobby

import "strings"

// titleControlBytes is the set of ASCII control characters
// (0x00-0x1F + 0x7F) that are rejected in room titles
// before any other normalisation. Centralised here so callers do not
// duplicate the byte-list literal.
const titleControlBytes = "\x00\x01\x02\x03\x04\x05\x06\x07\x08\x09\x0A\x0B\x0C\x0D\x0E\x0F" +
	"\x10\x11\x12\x13\x14\x15\x16\x17\x18\x19\x1A\x1B\x1C\x1D\x1E\x1F\x7F"

// MaxTitleLen is the hard cap on the (post-validation)
// room title length. Titles longer than this are truncated to the
// limit and then trimmed of surrounding whitespace.
const MaxTitleLen = 60

// ValidateTitle reports whether `raw` is acceptable as a new room
// title. A title is rejected when it is empty after trimming or
// contains any ASCII control byte.
func ValidateTitle(raw string) bool {
	if strings.TrimSpace(raw) == "" {
		return false
	}

	if strings.ContainsAny(raw, titleControlBytes) {
		return false
	}

	return true
}

// NormalizeTitle applies the normalisation order: cap
// length to MaxTitleLen, then trim leading/trailing whitespace. The
// caller is expected to have already passed `raw` through
// ValidateTitle.
func NormalizeTitle(raw string) string {
	if len(raw) > MaxTitleLen {
		raw = raw[:MaxTitleLen]
	}

	return strings.TrimSpace(raw)
}
