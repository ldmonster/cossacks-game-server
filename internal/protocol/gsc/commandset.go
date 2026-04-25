package gsc

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strings"
)

func CommandSetFromString(s string) CommandSet {
	s = strings.TrimPrefix(s, "GW|")
	parts := strings.Split(s, "|")
	out := CommandSet{Commands: make([]Command, 0, len(parts))}
	for _, part := range parts {
		out.Commands = append(out.Commands, CommandFromString(part))
	}
	return out
}

func (cs CommandSet) String() string {
	parts := make([]string, 0, len(cs.Commands))
	for _, c := range cs.Commands {
		parts = append(parts, c.String())
	}
	return "GW|" + strings.Join(parts, "|")
}

func (cs CommandSet) MarshalBinary() ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.LittleEndian, uint16(len(cs.Commands))); err != nil {
		return nil, err
	}
	for _, cmd := range cs.Commands {
		if len(cmd.Name) > 255 {
			return nil, fmt.Errorf("command name too long: %q", cmd.Name)
		}
		buf.WriteByte(byte(len(cmd.Name)))
		buf.WriteString(cmd.Name)
		if err := binary.Write(buf, binary.LittleEndian, uint16(len(cmd.Args))); err != nil {
			return nil, err
		}
		for _, arg := range cmd.Args {
			b := []byte(arg)
			if err := binary.Write(buf, binary.LittleEndian, uint32(len(b))); err != nil {
				return nil, err
			}
			buf.Write(b)
		}
	}
	return buf.Bytes(), nil
}

func (cs *CommandSet) UnmarshalBinary(b []byte) error {
	r := bytes.NewReader(b)
	var count uint16
	if err := binary.Read(r, binary.LittleEndian, &count); err != nil {
		return err
	}
	out := make([]Command, 0, count)
	for i := 0; i < int(count); i++ {
		nLen, err := r.ReadByte()
		if err != nil {
			return err
		}
		name := make([]byte, int(nLen))
		if _, err := r.Read(name); err != nil {
			return err
		}
		var argc uint16
		if err := binary.Read(r, binary.LittleEndian, &argc); err != nil {
			return err
		}
		cmd := Command{Name: string(name), Args: make([]string, 0, argc)}
		for j := 0; j < int(argc); j++ {
			var ln uint32
			if err := binary.Read(r, binary.LittleEndian, &ln); err != nil {
				return err
			}
			arg := make([]byte, ln)
			if _, err := r.Read(arg); err != nil {
				return err
			}
			cmd.Args = append(cmd.Args, string(arg))
		}
		out = append(out, cmd)
	}
	cs.Commands = out
	return nil
}
