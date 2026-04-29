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
	"strings"

	"go.uber.org/zap"

	"github.com/ldmonster/cossacks-game-server/internal/port"
	"github.com/ldmonster/cossacks-game-server/internal/transport/gsc"
	tconn "github.com/ldmonster/cossacks-game-server/internal/transport/tconn"
)

// RouteDispatcher routes a normalised open/go method to a per-route
// handler. The Controller satisfies this port today via DispatchOpen;
// the routes table lives in
// internal/app/gsc/routes/ and an explicit Routes registry will satisfy
// this port instead.
type RouteDispatcher interface {
	DispatchOpen(
		ctx context.Context,
		conn *tconn.Connection,
		req *gsc.Stream,
		method string,
		params map[string]string,
	) ([]gsc.Command, error)
}

// Open implements the GSC `open` command. It normalises the URL and
// parameter map, then delegates to a RouteDispatcher.
type Open struct {
	Routes RouteDispatcher
	Log    *zap.Logger
}

// Name returns the GSC command name handled by this command.
func (Open) Name() string { return "open" }

// Handle parses the URL/parameters and delegates to the route dispatcher.
func (o Open) Handle(
	ctx context.Context,
	conn *tconn.Connection,
	req *gsc.Stream,
	args []string,
) port.HandleResult {
	if len(args) < 1 {
		return port.HandleResult{Commands: []gsc.Command{}, HasResponse: true}
	}

	rawURL := strings.TrimSpace(strings.ReplaceAll(args[0], "\x00", ""))
	url := strings.TrimSuffix(rawURL, ".dcml")

	params := map[string]string{}
	if len(args) > 1 {
		params = ParseOpenParams(strings.ReplaceAll(args[1], "\x00", ""))
	}

	if o.Log != nil {
		o.Log.Debug("open route",
			zap.Uint64("conn_id", conn.ID),
			zap.String("raw_url", rawURL),
			zap.String("parsed_method", url),
			zap.Any("params", params),
		)
	}

	cmds, err := o.Routes.DispatchOpen(ctx, conn, req, url, params)

	return port.HandleResult{Commands: cmds, HasResponse: true, Err: err}
}

// Go implements the GSC `go` command. The argument format is
// "method key=value [key2 := value2 ...]"; values containing whitespace
// are split via the ":=" indirection used by the earlier client.
type Go struct {
	Routes RouteDispatcher
	Log    *zap.Logger
}

// Name returns the GSC command name handled by this command.
func (Go) Name() string { return "go" }

// Handle parses the parameters and delegates to the route dispatcher.
func (g Go) Handle(
	ctx context.Context,
	conn *tconn.Connection,
	req *gsc.Stream,
	args []string,
) port.HandleResult {
	if len(args) < 1 {
		return port.HandleResult{Commands: []gsc.Command{}, HasResponse: true}
	}

	method := args[0]
	params := map[string]string{}

	for i := 1; i < len(args); i++ {
		arg := args[i]
		if k, v, ok := strings.Cut(arg, "="); ok {
			params[k] = v
			continue
		}

		if strings.HasSuffix(arg, ":=") && i+1 < len(args) {
			params[strings.TrimSuffix(arg, ":=")] = args[i+1]
			i++
		}
	}

	if g.Log != nil {
		g.Log.Debug("go route",
			zap.Uint64("conn_id", conn.ID),
			zap.String("method", method),
			zap.Any("params", params),
		)
	}

	cmds, err := g.Routes.DispatchOpen(ctx, conn, req, method, params)

	return port.HandleResult{Commands: cmds, HasResponse: true, Err: err}
}

// ParseOpenParams decodes the earlier-encoded open-route parameter
// blob. The encoding is `key=value^KEY=value^...` where keys are
// alphanumeric/underscore and the `^` separator is significant only
// when followed by `key=`. Values may contain `^` literally.
func ParseOpenParams(params string) map[string]string {
	out := map[string]string{}

	for len(params) > 0 {
		eq := strings.IndexByte(params, '=')
		if eq <= 0 {
			break
		}

		key := params[:eq]
		rest := params[eq+1:]
		next := -1

		for i := 0; i < len(rest)-1; i++ {
			if rest[i] != '^' {
				continue
			}

			j := i + 1
			for j < len(rest) && isOpenParamKeyByte(rest[j]) {
				j++
			}

			if j < len(rest) && rest[j] == '=' {
				next = i
				break
			}
		}

		if next == -1 {
			out[key] = rest
			break
		}

		out[key] = rest[:next]
		params = rest[next+1:]
	}

	return out
}

func isOpenParamKeyByte(b byte) bool {
	return (b >= 'A' && b <= 'Z') ||
		(b >= 'a' && b <= 'z') ||
		(b >= '0' && b <= '9') ||
		b == '_'
}
