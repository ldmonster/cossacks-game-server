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

// Package logging builds zap loggers (user-facing console or JSON) with optional
// lumberjack file rotation. It does not register a global logger.
package logging

import (
	"fmt"
	"os"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/ldmonster/cossacks-game-server/internal/platform/config"
)

// Format selects the zap encoder layout.
type Format string

const (
	// FormatUser is multi-field console output (development-style).
	FormatUser Format = "user"
	// FormatJSON is one JSON object per line.
	FormatJSON Format = "json"
)

// Options configures a non-global zap logger.
type Options struct {
	Format Format
	// File, if non-empty, receives all log output via lumberjack rotation.
	File string
	// Level defaults to Info if unset (zero value).
	Level zapcore.Level
}

// ParseFormat normalizes and validates a format name.
func ParseFormat(s string) (Format, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "user":
		return FormatUser, nil
	case "json":
		return FormatJSON, nil
	default:
		return "", fmt.Errorf("log format must be %q or %q, got %q", FormatUser, FormatJSON, s)
	}
}

// ResolveOptions merges CLI flags with config and defaults. Non-empty flags override cfg.
func ResolveOptions(cfg *config.Config, logFormatFlag, logFileFlag string) (Options, error) {
	formatStr := strings.TrimSpace(logFormatFlag)
	if formatStr == "" && cfg != nil {
		formatStr = strings.TrimSpace(cfg.Log.Format)
	}

	if formatStr == "" {
		formatStr = "user"
	}

	f, err := ParseFormat(formatStr)
	if err != nil {
		return Options{}, err
	}

	file := strings.TrimSpace(logFileFlag)
	if file == "" && cfg != nil {
		file = strings.TrimSpace(cfg.Log.File)
	}

	return Options{Format: f, File: file}, nil
}

// New builds a zap.Logger. The caller must not use zap.ReplaceGlobals; retain and pass the pointer.
func New(opts Options) (*zap.Logger, error) {
	toFile := opts.File != ""

	level := opts.Level
	if level == 0 {
		// Match prior stdlib logging: debug-style lines were always printed.
		// JSON (typical production) stays at info; console user mode is verbose.
		if opts.Format == FormatUser && !toFile {
			level = zapcore.DebugLevel
		} else {
			level = zapcore.InfoLevel
		}
	}

	var enc zapcore.Encoder

	switch opts.Format {
	case FormatJSON:
		encCfg := zap.NewProductionEncoderConfig()
		encCfg.EncodeTime = zapcore.ISO8601TimeEncoder
		enc = zapcore.NewJSONEncoder(encCfg)
	case FormatUser:
		encCfg := zap.NewDevelopmentEncoderConfig()

		encCfg.EncodeTime = zapcore.ISO8601TimeEncoder
		if toFile {
			encCfg.EncodeLevel = zapcore.CapitalLevelEncoder
		} else {
			encCfg.EncodeLevel = zapcore.CapitalColorLevelEncoder
		}

		enc = zapcore.NewConsoleEncoder(encCfg)
	default:
		return nil, fmt.Errorf("unknown log format: %q", opts.Format)
	}

	var ws zapcore.WriteSyncer
	if toFile {
		ws = zapcore.AddSync(&lumberjack.Logger{
			Filename:   opts.File,
			MaxSize:    100,
			MaxBackups: 3,
			MaxAge:     28,
			Compress:   true,
		})
	} else {
		ws = zapcore.AddSync(os.Stdout)
	}

	core := zapcore.NewCore(enc, ws, level)

	return zap.New(core), nil
}
