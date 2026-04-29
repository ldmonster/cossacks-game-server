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
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestDeckhouseZapForwarder_JSONLine(t *testing.T) {
	core, obs := observer.New(zapcore.InfoLevel)
	z := zap.New(core)
	f := &deckhouseZapForwarder{z: z}

	line := []byte(`{"level":"error","logger":"metrics-storage","msg":"Counter","name":"m","time":"2026-01-01T00:00:00Z"}` + "\n")
	if _, err := f.Write(line); err != nil {
		t.Fatal(err)
	}

	entries := obs.All()
	if len(entries) != 1 {
		t.Fatalf("got %d log entries, want 1", len(entries))
	}
	if entries[0].Level != zapcore.ErrorLevel {
		t.Fatalf("level: got %v want error", entries[0].Level)
	}
	if entries[0].Message != "Counter" {
		t.Fatalf("message: got %q", entries[0].Message)
	}
}
