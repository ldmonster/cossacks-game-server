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

package identity

import (
	"regexp"
	"strings"
)

// MaxNickLen is the hard cap on guest-nick length.
const MaxNickLen = 25

// nickAllowedPattern restricts nicks to the same character class
// the reference server enforces (`a-z A-Z 0-9 [ ] _ -`).
var nickAllowedPattern = regexp.MustCompile(`^[\[\]_\w-]+$`)

// NickError is the typed rejection reason returned by ValidateNick.
// Each value carries a stable user-facing message via Message() so
// callers can render it directly into the `error_enter.tmpl` template
// without duplicating the strings.
type NickError int

const (
	// NickOK is the zero value used when no problem was detected.
	NickOK NickError = iota
	// NickEmpty is returned when the trimmed nick is empty.
	NickEmpty
	// NickBadCharacter is returned when the nick contains characters
	// outside the allowed `[a-zA-Z0-9\[\]_-]` set.
	NickBadCharacter
	// NickStartsWithDigit is returned when the nick's first character
	// is an ASCII digit.
	NickStartsWithDigit
	// NickStartsWithDash is returned when the nick begins with `-`.
	NickStartsWithDash
)

// Message returns the message associated with `e`.
// Callers feed the value directly into the `error_enter.tmpl`
// `error_text` slot.
func (e NickError) Message() string {
	switch e {
	case NickEmpty:
		return "Enter nick"
	case NickBadCharacter:
		return "Bad character in nick. Nick can contain only a-z,A-Z,0-9,[]_-"
	case NickStartsWithDigit:
		return "Bad character in nick. Nick can't start with numerical digit"
	case NickStartsWithDash:
		return "Bad character in nick. Nick can't start with -"
	default:
		return ""
	}
}

// ValidateNick reports whether `nick` is acceptable as a guest nick.
// Returns NickOK when the nick passes all checks, otherwise the
// specific failure reason. Callers are expected to render
// `Message()` via the enter-error template.
func ValidateNick(nick string) NickError {
	if nick == "" {
		return NickEmpty
	}

	if !nickAllowedPattern.MatchString(nick) {
		return NickBadCharacter
	}

	if strings.HasPrefix(nick, "-") {
		return NickStartsWithDash
	}

	if nick[0] >= '0' && nick[0] <= '9' {
		return NickStartsWithDigit
	}

	return NickOK
}

// TruncateNick caps `nick` at MaxNickLen characters, returning the shortened nick.
func TruncateNick(nick string) string {
	if len(nick) > MaxNickLen {
		return nick[:MaxNickLen]
	}

	return nick
}

// SanitizeAccountNick converts an LCN/WCL account login into a nick
// that satisfies ValidateNick. Unsupported characters are dropped, an
// empty result is replaced with "player", and a leading digit is
// prefixed with `_` to satisfy the start-with-digit rule.
func SanitizeAccountNick(nick string) string {
	var b strings.Builder

	for _, r := range nick {
		if r == '[' || r == ']' || r == '_' || r == '-' ||
			(r >= '0' && r <= '9') ||
			(r >= 'a' && r <= 'z') ||
			(r >= 'A' && r <= 'Z') {
			b.WriteRune(r)
		}
	}

	out := b.String()
	if out == "" {
		return "player"
	}

	if out[0] >= '0' && out[0] <= '9' {
		return "_" + out
	}

	return out
}
