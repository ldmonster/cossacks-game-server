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

import (
	"bytes"
	"testing"
)

func TestStreamBinaryRoundTrip(t *testing.T) {
	in := Stream{
		Num:  15,
		Lang: 1,
		Ver:  2,
		CmdSet: CommandSet{
			Commands: []Command{
				{Name: "login", Args: []string{"abc", "WIN", "KEY"}},
				{Name: "echo", Args: []string{"1", "2"}},
			},
		},
	}
	bin, err := in.MarshalBinary()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var out Stream
	if err := out.UnmarshalBinary(bin); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.Num != in.Num || out.Lang != in.Lang || out.Ver != in.Ver {
		t.Fatalf("header mismatch: %#v != %#v", out, in)
	}
	if got, want := out.CmdSet.String(), in.CmdSet.String(); got != want {
		t.Fatalf("cmdset mismatch:\n got=%q\nwant=%q", got, want)
	}
}

func TestReadFrom(t *testing.T) {
	in := Stream{
		Num:    1,
		Lang:   1,
		Ver:    2,
		CmdSet: CommandSet{Commands: []Command{{Name: "ping", Args: []string{"WIN", "KEY"}}}},
	}
	bin, _ := in.MarshalBinary()
	out, err := ReadFrom(bytes.NewReader(bin), 1024*1024)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if out.CmdSet.Commands[0].Name != "ping" {
		t.Fatalf("wrong command: %q", out.CmdSet.Commands[0].Name)
	}
}
