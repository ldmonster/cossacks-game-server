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

// Package render owns the LW response builders and the show-template
// renderer service contract.
package render

import "github.com/ldmonster/cossacks-game-server/internal/transport/gsc"

// ShowCmd returns a bare LW_show gsc.Command (no slice wrapper). Use this
// when composing multi-command response slices.
func ShowCmd(body string) gsc.Command {
	return gsc.Command{Name: "LW_show", Args: []string{body}}
}

// Show returns a single LW_show command with body as its only argument.
// This is the most common response shape across the controller.
func Show(body string) []gsc.Command {
	return []gsc.Command{ShowCmd(body)}
}

// Echo returns an LW_echo response carrying args verbatim.
func Echo(args []string) []gsc.Command {
	return []gsc.Command{{Name: "LW_echo", Args: args}}
}

// Time returns an LW_time response. Cossacks uses LW_time to schedule a
// follow-up open(...) navigation after a delay.
func Time(delay, action string) []gsc.Command {
	return []gsc.Command{{Name: "LW_time", Args: []string{delay, action}}}
}

// Alert returns an alert_dgl response with the given header and message.
func Alert(header, message string) []gsc.Command {
	return []gsc.Command{{Name: "alert_dgl", Args: []string{header, message}}}
}
