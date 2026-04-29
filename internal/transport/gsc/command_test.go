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

func TestCommandStringEscaping(t *testing.T) {
	cmd := Command{
		Name: "go",
		Args: []string{`A\B&C|D` + "\x00"},
	}
	encoded := cmd.String()
	if encoded != `go&A\5CB\26C\7CD\00` {
		t.Fatalf("unexpected encoded command: %q", encoded)
	}
	decoded := CommandFromString(encoded)
	if decoded.Name != cmd.Name || len(decoded.Args) != 1 || decoded.Args[0] != cmd.Args[0] {
		t.Fatalf("decode mismatch: %#v", decoded)
	}
}

func TestCommandSetStringRoundTrip(t *testing.T) {
	in := "GW|open&enter.dcml&NICK=foo\\26bar|echo&a&b"
	cs := CommandSetFromString(in)
	out := cs.String()
	if out != in {
		t.Fatalf("roundtrip mismatch:\n in=%q\nout=%q", in, out)
	}
}
