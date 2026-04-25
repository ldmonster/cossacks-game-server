package gsc

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
)

const headerSize = 12

func (s Stream) MarshalBinary() ([]byte, error) {
	cmdsetBin, err := s.CmdSet.MarshalBinary()
	if err != nil {
		return nil, err
	}
	var compressed bytes.Buffer
	zw := zlib.NewWriter(&compressed)
	if _, err := zw.Write(cmdsetBin); err != nil {
		return nil, err
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	size := uint32(compressed.Len() + headerSize)
	payloadLen := uint32(len(cmdsetBin))
	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.LittleEndian, s.Num); err != nil {
		return nil, err
	}
	buf.WriteByte(s.Lang)
	buf.WriteByte(s.Ver)
	if err := binary.Write(buf, binary.LittleEndian, size); err != nil {
		return nil, err
	}
	if err := binary.Write(buf, binary.LittleEndian, payloadLen); err != nil {
		return nil, err
	}
	buf.Write(compressed.Bytes())
	return buf.Bytes(), nil
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
	var size uint32
	var payloadLen uint32
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
	cmdsetBin, err := io.ReadAll(zr)
	if err != nil {
		return err
	}
	if uint32(len(cmdsetBin)) != payloadLen {
		return fmt.Errorf("wrong stream len: got %d want %d", len(cmdsetBin), payloadLen)
	}
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
