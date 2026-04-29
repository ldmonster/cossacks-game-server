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

package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadLegacyConf(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "simple-cossacks-server.conf")
	body := `host = 127.0.0.1
port = 34002
hole_port = 3709
hole_int = 450
templates = ./templates
access_log = ./logs/access_log
error_log = ./logs/error_log
table_timeout = 5000
gettbl_log_interval = 2
chat_server = osiris.2gw.net
show_started_rooms = 1
proxy_key = KEY
`
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write conf: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("load conf: %v", err)
	}
	if cfg.Server.Port != 34002 || cfg.Game.HoleInterval != 450 || !cfg.Game.ShowStartedRooms {
		t.Fatalf("unexpected parsed config: %#v", cfg)
	}
}

func TestLoadYAML_ParsesYAMLKeys(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "simple-cossacks-server.yaml")
	body := `host: 127.0.0.1
port: 34002
hole_port: 3709
hole_int: 450
templates: ./templates
access_log: ./logs/access_log
error_log: ./logs/error_log
table_timeout: 5000
gettbl_log_interval: 2
chat_server: osiris.2gw.net
show_started_rooms: true
proxy_key: KEY
`
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write yaml: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("load yaml: %v", err)
	}
	if cfg.Server.Port != 34002 || cfg.Server.HolePort != 3709 || cfg.Game.HoleInterval != 450 {
		t.Fatalf("unexpected yaml numeric fields: %#v", cfg)
	}
	if !cfg.Game.ShowStartedRooms {
		t.Fatalf("expected show_started_rooms=true")
	}
}

func TestApplyEnv_OverridesFromEnvironment(t *testing.T) {
	cfg := &Config{
		Game: GameConfig{HoleInterval: 300},
	}
	t.Setenv("HOST_NAME", "osiris.2gw.net")
	t.Setenv("UDP_KEEP_ALIVE_INTERVAL", "450")

	cfg.ApplyEnv()
	if cfg.Game.ChatServer != "osiris.2gw.net" {
		t.Fatalf("chat server mismatch: %q", cfg.Game.ChatServer)
	}
	if cfg.Game.HoleInterval != 450 {
		t.Fatalf("hole interval mismatch: %d", cfg.Game.HoleInterval)
	}
}

func TestApplyEnv_InvalidIntervalFallsBack(t *testing.T) {
	cfg := &Config{
		Game: GameConfig{HoleInterval: 321},
	}
	t.Setenv("UDP_KEEP_ALIVE_INTERVAL", "invalid")

	cfg.ApplyEnv()
	if cfg.Game.HoleInterval != 321 {
		t.Fatalf("hole interval should remain default, got %d", cfg.Game.HoleInterval)
	}
}
