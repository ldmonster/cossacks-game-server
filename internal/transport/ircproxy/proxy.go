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

// Package ircproxy provides a transparent TCP reverse-proxy that forwards
// IRC client connections to an upstream Ergo IRC server.
package ircproxy

import (
	"context"
	"io"
	"net"

	"go.uber.org/zap"
)

// Proxy accepts IRC client connections on Listen and forwards all traffic
// bidirectionally to the upstream Ergo server at Addr.
type Proxy struct {
	// Listen is the address to bind for incoming IRC client connections
	// (e.g. ":6667").
	Listen string
	// Addr is the upstream Ergo IRC server address
	// (e.g. "ergo:6667" or "127.0.0.1:6667").
	Addr string
	// Log is used for connection lifecycle and error events.
	Log *zap.Logger
}

// ListenAndServe starts accepting connections and blocks until ctx is
// cancelled or a fatal listener error occurs.
func (p *Proxy) ListenAndServe(ctx context.Context) error {
	ln, err := net.Listen("tcp", p.Listen)
	if err != nil {
		return err
	}

	p.Log.Info("irc proxy listen", zap.String("listen", p.Listen), zap.String("upstream", p.Addr))

	go func() {
		<-ctx.Done()
		_ = ln.Close()
	}()

	for {
		client, err := ln.Accept()
		if err != nil {
			// Return nil on context cancellation so errgroup treats it as
			// a clean shutdown.
			if ctx.Err() != nil {
				return nil
			}

			return err
		}

		go p.pipe(ctx, client)
	}
}

func (p *Proxy) pipe(ctx context.Context, client net.Conn) {
	defer client.Close()

	remote, err := net.Dial("tcp", p.Addr)
	if err != nil {
		p.Log.Warn("irc proxy: upstream dial failed",
			zap.String("upstream", p.Addr),
			zap.String("client", client.RemoteAddr().String()),
			zap.Error(err),
		)

		return
	}

	defer remote.Close()

	p.Log.Debug("irc proxy: connected",
		zap.String("client", client.RemoteAddr().String()),
		zap.String("upstream", p.Addr),
	)

	// Cancel both copy goroutines as soon as one side closes.
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	go func() {
		<-ctx.Done()
		_ = client.Close()
		_ = remote.Close()
	}()

	done := make(chan struct{}, 2)

	go func() {
		_, _ = io.Copy(remote, client)
		done <- struct{}{}
	}()

	go func() {
		_, _ = io.Copy(client, remote)
		done <- struct{}{}
	}()

	// Wait for either direction to finish then cancel the other.
	<-done
}
