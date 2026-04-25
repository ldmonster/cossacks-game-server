package gsc

import "testing"

func TestCommandStringEscapingParity(t *testing.T) {
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
