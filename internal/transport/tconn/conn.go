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

package tconn

import (
	"context"
	"encoding/binary"
	"net"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/ldmonster/cossacks-game-server/internal/domain/session"
)

var connSeq atomic.Uint64

// Connection represents a single client TCP connection. The previous
// `Data map[string]any` field has been replaced by a typed Session.
// All controller code reads/writes
// session-scoped state via `conn.Session.<Field>` instead of stringly-
// typed map keys.
//
// Lifecycle ordering: create with NewConnection → dispatch commands →
// call cancel() to signal disconnect → closed by the server loop.
type Connection struct {
	ID      uint64
	NetConn net.Conn
	IP      string
	Port    int
	IntIP   uint32
	Ctime   time.Time
	Session *session.Session

	// ctx/cancel replace a previous Closed bool flag.
	// cancel() is called by handleProxy to signal that the connection
	// should be terminated after the current command completes.
	ctx    context.Context //nolint:containedctx // intentional per-conn lifecycle
	cancel context.CancelFunc
}

// IsClosed reports whether the connection has been cancelled (i.e. the
// proxy handler called cancel()). Safe to call from any goroutine.
func (c *Connection) IsClosed() bool {
	if c.ctx == nil {
		return false
	}

	select {
	case <-c.ctx.Done():
		return true
	default:
		return false
	}
}

// Cancel cancels the connection context, signalling the read loop to
// stop after the current command finishes.
func (c *Connection) Cancel() { c.cancel() }

func NewConnection(conn net.Conn) *Connection {
	host, portStr, _ := net.SplitHostPort(conn.RemoteAddr().String())
	ip := net.ParseIP(host).To4()

	intIP := uint32(0)
	if ip != nil {
		intIP = binary.LittleEndian.Uint32(ip)
	}

	port, _ := strconv.Atoi(portStr)

	ctx, cancel := context.WithCancel(context.Background())

	return &Connection{
		ID:      connSeq.Add(1),
		NetConn: conn,
		IP:      host,
		Port:    port,
		IntIP:   intIP,
		Ctime:   time.Now(),
		Session: session.New(),
		ctx:     ctx,
		cancel:  cancel,
	}
}
