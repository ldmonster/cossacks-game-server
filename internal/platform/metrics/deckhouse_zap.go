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

package metrics

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"strings"

	decklog "github.com/deckhouse/deckhouse/pkg/log"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// NewDeckhouseLogger builds a Deckhouse *log.Logger that forwards every emitted
// log line to z. metrics-storage expects this type for WithLogger; this keeps a
// single application log sink (zap) without zap.ReplaceGlobals.
func NewDeckhouseLogger(z *zap.Logger) *decklog.Logger {
	w := &deckhouseZapForwarder{z: z.Named("deckhouse")}

	return decklog.NewLogger(
		decklog.WithOutput(w),
		decklog.WithHandlerType(decklog.JSONHandlerType),
		decklog.WithLevel(slog.LevelDebug),
	)
}

type deckhouseZapForwarder struct {
	z *zap.Logger
}

func (f *deckhouseZapForwarder) Write(p []byte) (int, error) {
	line := bytes.TrimSpace(p)
	if len(line) == 0 {
		return len(p), nil
	}

	var m map[string]any
	if err := json.Unmarshal(line, &m); err != nil {
		f.z.Info("deckhouse_log", zap.ByteString("raw", line))
		return len(p), nil
	}

	msg, _ := m["msg"].(string)
	if msg == "" {
		msg = "deckhouse_log"
	}

	lvlStr, _ := m["level"].(string)
	lvl := deckhouseLevelToZap(lvlStr)

	fields := make([]zap.Field, 0, len(m))
	for k, v := range m {
		switch k {
		case "msg", "level":
			continue
		default:
			fields = append(fields, zap.Any(k, v))
		}
	}

	switch lvl {
	case zapcore.ErrorLevel:
		f.z.Error(msg, fields...)
	case zapcore.WarnLevel:
		f.z.Warn(msg, fields...)
	case zapcore.DebugLevel:
		f.z.Debug(msg, fields...)
	default:
		f.z.Info(msg, fields...)
	}

	return len(p), nil
}

func deckhouseLevelToZap(s string) zapcore.Level {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "error":
		return zapcore.ErrorLevel
	case "warn", "warning":
		return zapcore.WarnLevel
	case "info":
		return zapcore.InfoLevel
	case "debug":
		return zapcore.DebugLevel
	case "trace":
		return zapcore.DebugLevel
	default:
		return zapcore.InfoLevel
	}
}

var _ io.Writer = (*deckhouseZapForwarder)(nil)
