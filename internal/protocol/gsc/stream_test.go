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
