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
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"sync"
)

const headerSize = 12

// zlibWriterPool recycles zlib.Writer instances to avoid per-message
// allocations on the hot outbound path.
var zlibWriterPool = sync.Pool{
	New: func() any {
		w, _ := zlib.NewWriterLevel(io.Discard, zlib.DefaultCompression)
		return w
	},
}

// bufPool recycles bytes.Buffer instances for compression and output
// assembly buffers.
var bufPool = sync.Pool{
	New: func() any { return new(bytes.Buffer) },
}

func (s Stream) MarshalBinary() ([]byte, error) {
	cmdsetBin, err := s.CmdSet.MarshalBinary()
	if err != nil {
		return nil, err
	}

	compressed := bufPool.Get().(*bytes.Buffer)
	compressed.Reset()

	zw := zlibWriterPool.Get().(*zlib.Writer)
	zw.Reset(compressed)

	if _, err := zw.Write(cmdsetBin); err != nil {
		zlibWriterPool.Put(zw)
		bufPool.Put(compressed)

		return nil, err
	}

	if err := zw.Close(); err != nil {
		zlibWriterPool.Put(zw)
		bufPool.Put(compressed)

		return nil, err
	}

	zlibWriterPool.Put(zw)

	size := uint32(compressed.Len() + headerSize)
	payloadLen := uint32(len(cmdsetBin))

	buf := bufPool.Get().(*bytes.Buffer)
	buf.Reset()

	if err := binary.Write(buf, binary.LittleEndian, s.Num); err != nil {
		bufPool.Put(compressed)
		bufPool.Put(buf)

		return nil, err
	}

	buf.WriteByte(s.Lang)
	buf.WriteByte(s.Ver)

	if err := binary.Write(buf, binary.LittleEndian, size); err != nil {
		bufPool.Put(compressed)
		bufPool.Put(buf)

		return nil, err
	}

	if err := binary.Write(buf, binary.LittleEndian, payloadLen); err != nil {
		bufPool.Put(compressed)
		bufPool.Put(buf)

		return nil, err
	}

	buf.Write(compressed.Bytes())
	bufPool.Put(compressed)

	out := make([]byte, buf.Len())
	copy(out, buf.Bytes())
	bufPool.Put(buf)

	return out, nil
}

func (s *Stream) UnmarshalBinary(b []byte) error {
	if len(b) < headerSize {
		return io.ErrUnexpectedEOF
	}

	r := bytes.NewReader(b)
	if err := binary.Read(r, binary.LittleEndian, &s.Num); err != nil {
		return err
	}

	if err := binary.Read(r, binary.LittleEndian, &s.Lang); err != nil {
		return err
	}

	if err := binary.Read(r, binary.LittleEndian, &s.Ver); err != nil {
		return err
	}

	var (
		size       uint32
		payloadLen uint32
	)

	if err := binary.Read(r, binary.LittleEndian, &size); err != nil {
		return err
	}

	if err := binary.Read(r, binary.LittleEndian, &payloadLen); err != nil {
		return err
	}

	if int(size) < headerSize || len(b) < int(size) {
		return fmt.Errorf("invalid size: %d", size)
	}

	zr, err := zlib.NewReader(bytes.NewReader(b[headerSize:size]))
	if err != nil {
		return err
	}
	defer zr.Close()

	buf := bufPool.Get().(*bytes.Buffer)
	buf.Reset()

	if _, err := buf.ReadFrom(zr); err != nil {
		bufPool.Put(buf)

		return err
	}

	if uint32(buf.Len()) != payloadLen {
		bufPool.Put(buf)

		return fmt.Errorf("wrong stream len: got %d want %d", buf.Len(), payloadLen)
	}

	cmdsetBin := make([]byte, buf.Len())
	copy(cmdsetBin, buf.Bytes())
	bufPool.Put(buf)

	return s.CmdSet.UnmarshalBinary(cmdsetBin)
}

func ReadFrom(r io.Reader, maxSize uint32) (*Stream, error) {
	head := make([]byte, headerSize)
	if _, err := io.ReadFull(r, head); err != nil {
		return nil, err
	}

	size := binary.LittleEndian.Uint32(head[4:8])
	if maxSize > 0 && size > maxSize {
		return nil, fmt.Errorf("request too large: %d", size)
	}

	full := make([]byte, size)
	copy(full, head)

	if _, err := io.ReadFull(r, full[headerSize:]); err != nil {
		return nil, err
	}

	var s Stream
	if err := s.UnmarshalBinary(full); err != nil {
		return nil, err
	}

	return &s, nil
}
