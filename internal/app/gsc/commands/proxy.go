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

package commands

import (
	"context"
	"encoding/binary"
	"net"
	"strconv"
	"strings"

	"github.com/ldmonster/cossacks-game-server/internal/port"
	"github.com/ldmonster/cossacks-game-server/internal/transport/gsc"
	tconn "github.com/ldmonster/cossacks-game-server/internal/transport/tconn"
)

// ErrProxyInvalidArgs signals that the GSC "proxy" command was issued
// with malformed or missing arguments. Reason carries the precise
// failure mode for observability.
type ErrProxyInvalidArgs struct{ Reason string }

// Error implements the error interface.
func (e ErrProxyInvalidArgs) Error() string {
	return "gsc: proxy: invalid args: " + e.Reason
}

// ErrProxyUnauthorized signals that the supplied proxy key did not
// match the configured ProxyKey.
type ErrProxyUnauthorized struct{}

// Error implements the error interface.
func (ErrProxyUnauthorized) Error() string { return "gsc: proxy: unauthorized" }

// Proxy implements the GSC "proxy" command. The host informs the
// server that subsequent traffic for this connection should be
// attributed to a different (ip, port). The supplied proxy key must
// match Proxy.Key for the rewrite to be honoured.
type Proxy struct {
	// Key is the shared secret a client must present in args[2] for
	// the rewrite to be accepted. An empty value disables the command.
	Key string
}

// Name implements gsc.CommandHandler.
func (Proxy) Name() string { return "proxy" }

// Handle implements gsc.CommandHandler.
func (p Proxy) Handle(
	_ context.Context,
	conn *tconn.Connection,
	_ *gsc.Stream,
	args []string,
) port.HandleResult {
	if len(args) < 3 {
		conn.Cancel()

		return port.HandleResult{
			HasResponse: false,
			Err:         ErrProxyInvalidArgs{Reason: "missing ip/port/key"},
		}
	}

	ipArg, portArg, keyArg := args[0], args[1], args[2]

	if p.Key == "" || keyArg != p.Key {
		conn.Cancel()
		return port.HandleResult{HasResponse: false, Err: ErrProxyUnauthorized{}}
	}

	ip := net.ParseIP(strings.TrimSpace(ipArg)).To4()
	if ip == nil {
		conn.Cancel()
		return port.HandleResult{HasResponse: false, Err: ErrProxyInvalidArgs{Reason: "bad ip"}}
	}

	portNum, err := strconv.Atoi(strings.TrimSpace(portArg))
	if err != nil || portNum <= 0 || portNum >= 0xFFFF {
		conn.Cancel()
		return port.HandleResult{HasResponse: false, Err: ErrProxyInvalidArgs{Reason: "bad port"}}
	}

	conn.IP = ip.String()
	conn.IntIP = binary.LittleEndian.Uint32(ip)
	conn.Port = portNum

	return port.HandleResult{HasResponse: false}
}
