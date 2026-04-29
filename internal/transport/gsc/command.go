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
	"fmt"
	"strconv"
	"strings"
)

// argEncoder is hoisted to package level so it is constructed once rather
// than on every encodeArg call.
var argEncoder = strings.NewReplacer(
	"\\", "\\5C",
	"&", "\\26",
	"|", "\\7C",
	"\x00", "\\00",
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
	return argEncoder.Replace(arg)
}

func (c Command) Validate() error {
	if c.Name == "" {
		return fmt.Errorf("command name is empty")
	}

	return nil
}
