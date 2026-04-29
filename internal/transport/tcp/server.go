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

package tcp

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	metricsstorage "github.com/deckhouse/deckhouse/pkg/metrics-storage"
	"go.uber.org/zap"

	"github.com/ldmonster/cossacks-game-server/internal/platform/metrics"
	"github.com/ldmonster/cossacks-game-server/internal/port"
	"github.com/ldmonster/cossacks-game-server/internal/transport/gsc"
	"github.com/ldmonster/cossacks-game-server/internal/transport/tconn"
)

type Server struct {
	Host    string
	Port    int
	MaxSize uint32

	Log     *zap.Logger
	Metrics *metricsstorage.MetricStorage
	Handler port.RequestHandler
}

func (s *Server) ListenAndServe(ctx context.Context) error {
	addr := fmt.Sprintf("%s:%d", s.Host, s.Port)

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	defer ln.Close()

	s.Log.Info("listen tcp", zap.String("addr", addr))

	for {
		nc, err := ln.Accept()
		if err != nil {
			return err
		}

		go s.handleConn(ctx, nc)
	}
}

func (s *Server) handleConn(ctx context.Context, nc net.Conn) {
	defer nc.Close()

	c := tconn.NewConnection(nc)
	s.Log.Info("client connect",
		zap.Uint64("conn_id", c.ID),
		zap.String("ip", c.IP),
		zap.Int("port", c.Port),
	)

	metrics.TCPClientConnected(s.Metrics)

	defer func() {
		s.Handler.OnDisconnect(c)
		metrics.TCPClientDisconnected(s.Metrics)
		s.Log.Info("client disconnect",
			zap.Uint64("conn_id", c.ID),
			zap.String("ip", c.IP),
			zap.Int("port", c.Port),
		)
	}()

	for {
		req, err := gsc.ReadFrom(nc, s.MaxSize)
		if err != nil {
			s.Log.Warn("read error",
				zap.Uint64("conn_id", c.ID),
				zap.String("ip", c.IP),
				zap.Error(err),
			)

			return
		}

		if len(req.CmdSet.Commands) == 0 {
			s.Log.Warn("empty command set",
				zap.Uint64("conn_id", c.ID),
				zap.String("ip", c.IP),
			)

			continue
		}

		cmd := req.CmdSet.Commands[0]
		if len(req.CmdSet.Commands) > 1 {
			s.Log.Warn("more than one command in request, ignoring rest")
		}

		s.Log.Info("recv",
			zap.Uint64("conn_id", c.ID),
			zap.String("cmd", cmd.Name),
			zap.Int("arg_count", len(cmd.Args)),
		)

		var coreArgs []string

		win := ""
		key := ""

		if len(cmd.Args) >= 2 {
			coreArgs = append([]string(nil), cmd.Args[:len(cmd.Args)-2]...)
			win = cmd.Args[len(cmd.Args)-2]
			key = cmd.Args[len(cmd.Args)-1]
		} else {
			// the reference warns on short args but still dispatches command.
			s.Log.Warn("args count < 2; dispatching with empty win/key",
				zap.String("cmd", cmd.Name),
			)
			coreArgs = append([]string(nil), cmd.Args...)
		}

		start := time.Now()
		result := s.Handler.HandleWithMeta(ctx, c, req, cmd.Name, coreArgs, win, key)
		metrics.IncGSCCommand(s.Metrics, cmd.Name)
		metrics.ObserveGSCRequest(s.Metrics, cmd.Name, time.Since(start).Seconds())

		if result.Err != nil {
			s.Log.Warn("handler error",
				zap.Uint64("conn_id", c.ID),
				zap.String("cmd", cmd.Name),
				zap.Error(result.Err),
			)
		}

		if !result.HasResponse {
			s.Log.Info("no response",
				zap.Uint64("conn_id", c.ID),
				zap.String("cmd", cmd.Name),
			)

			continue
		}

		response := result.Commands
		for i := range response {
			response[i].Args = append(response[i].Args, win)
			if response[i].Name == "LW_show" && len(response[i].Args) > 0 {
				body := response[i].Args[0]

				preview := body
				if len(preview) > 140 {
					preview = preview[:140]
				}

				preview = strings.ReplaceAll(preview, "\n", "\\n")
				s.Log.Debug("show payload",
					zap.Uint64("conn_id", c.ID),
					zap.Int("body_len", len(body)),
					zap.String("preview", preview),
				)

				if strings.Contains(body, "%CG_HOLEHOST&") ||
					strings.Contains(body, "%CG_HOLEPORT&") {
					s.Log.Debug("send to client stun vars",
						zap.Uint64("conn_id", c.ID),
						zap.String("CG_HOLEHOST", extractGVar(body, "CG_HOLEHOST")),
						zap.String("CG_HOLEPORT", extractGVar(body, "CG_HOLEPORT")),
						zap.String("CG_HOLEINT", extractGVar(body, "CG_HOLEINT")),
					)
				}
			}
		}

		out := gsc.Stream{
			Num:  req.Num,
			Lang: req.Lang,
			Ver:  req.Ver,
			CmdSet: gsc.CommandSet{
				Commands: response,
			},
		}

		bin, err := out.MarshalBinary()
		if err != nil {
			s.Log.Error("marshal error",
				zap.Uint64("conn_id", c.ID),
				zap.String("cmd", cmd.Name),
				zap.Error(err),
			)

			return
		}

		if _, err := nc.Write(bin); err != nil {
			s.Log.Warn("write error",
				zap.Uint64("conn_id", c.ID),
				zap.String("cmd", cmd.Name),
				zap.Error(err),
			)

			return
		}

		s.Log.Info("sent",
			zap.Uint64("conn_id", c.ID),
			zap.String("cmd", cmd.Name),
			zap.Int("responses", len(response)),
		)

		if c.IsClosed() {
			return
		}
	}
}

func extractGVar(body, key string) string {
	token := "%" + key + "&"

	start := strings.Index(body, token)
	if start < 0 {
		return ""
	}

	rest := body[start+len(token):]

	end := strings.Index(rest, "&%")
	if end < 0 {
		if zero := strings.IndexByte(rest, 0); zero >= 0 {
			end = zero
		} else {
			end = len(rest)
		}
	}

	return rest[:end]
}
