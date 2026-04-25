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
	if cfg.Port != 34002 || cfg.HoleInterval != 450 || !cfg.ShowStartedRooms {
		t.Fatalf("unexpected parsed config: %#v", cfg)
	}
	if cfg.Raw["show_started_rooms"] != "1" {
		t.Fatalf("raw show_started_rooms mismatch: %q", cfg.Raw["show_started_rooms"])
	}
}

func TestLoadYAML_ParsesPerlCompatibleKeys(t *testing.T) {
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
	if cfg.Port != 34002 || cfg.HolePort != 3709 || cfg.HoleInterval != 450 {
		t.Fatalf("unexpected yaml numeric fields: %#v", cfg)
	}
	if !cfg.ShowStartedRooms {
		t.Fatalf("expected show_started_rooms=true")
	}
	if cfg.Raw["show_started_rooms"] != "1" {
		t.Fatalf("expected raw show_started_rooms=1, got %q", cfg.Raw["show_started_rooms"])
	}
}

