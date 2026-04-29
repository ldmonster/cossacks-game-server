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
