package gsc

import (
	"fmt"
	"strconv"
	"strings"
)

func CommandFromString(s string) Command {
	parts := strings.Split(s, "&")
	cmd := Command{Name: parts[0]}
	for _, arg := range parts[1:] {
		cmd.Args = append(cmd.Args, decodeArg(arg))
	}
	return cmd
}

func (c Command) String() string {
	args := make([]string, 0, len(c.Args)+1)
	args = append(args, c.Name)
	for _, arg := range c.Args {
		args = append(args, encodeArg(arg))
	}
	return strings.Join(args, "&")
}

func decodeArg(arg string) string {
	var b strings.Builder
	for i := 0; i < len(arg); i++ {
		if arg[i] == '\\' && i+2 < len(arg) {
			if n, err := strconv.ParseUint(arg[i+1:i+3], 16, 8); err == nil {
				b.WriteByte(byte(n))
				i += 2
				continue
			}
		}
		b.WriteByte(arg[i])
	}
	return b.String()
}

func encodeArg(arg string) string {
	repl := strings.NewReplacer(
		"\\", "\\5C",
		"&", "\\26",
		"|", "\\7C",
		"\x00", "\\00",
	)
	return repl.Replace(arg)
}

func (c Command) Validate() error {
	if c.Name == "" {
		return fmt.Errorf("command name is empty")
	}
	return nil
}
