package stun

import (
	"encoding/binary"
	"testing"
)

func TestParsePacketParity(t *testing.T) {
	buf := make([]byte, 25)
	copy(buf[0:4], []byte("CSHP"))
	buf[4] = 1
	binary.BigEndian.PutUint32(buf[5:9], 12345)
	copy(buf[9:], []byte("abc\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00"))
	p, err := ParsePacket(buf)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if p.Tag != "CSHP" || p.Version != 1 || p.PlayerID != 12345 || p.AccessKey != "abc" {
		t.Fatalf("bad packet: %#v", p)
	}
}
