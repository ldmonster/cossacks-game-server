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
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/caarlos0/env/v11"
	"gopkg.in/yaml.v3"
)

// ProviderConfig holds per-auth-provider settings (LCN, WCL, …). Lookup is
// keyed by lower-cased account type via Config.Provider.
type ProviderConfig struct {
	Host       string
	Key        string
	ServerName string
}

// ServerConfig groups TCP/UDP listener settings.
type ServerConfig struct {
	Host     string
	Port     int
	HolePort int
	// HostName is the public hostname/IP advertised to clients in STUN
	// hole-punch responses. Populated from HOST_NAME env at startup.
	// Empty means fall back to the connecting client's reported IP.
	HostName string
}

// LogConfig groups logging-related settings.
type LogConfig struct {
	Format    string
	File      string
	AccessLog string
	ErrorLog  string
}

// MetricsConfig groups Prometheus / probe HTTP listener settings.
type MetricsConfig struct {
	Addr      string
	ProbeAddr string
}

// GSCConfig groups protocol-level transport settings for the GSC
// listener.
type GSCConfig struct {
	// MaxFrameBytes caps the size, in bytes, of a single GSC frame the
	// TCP server is willing to accept. A zero value means "use the
	// default" (DefaultGSCMaxFrameBytes).
	MaxFrameBytes uint32
}

// DefaultGSCMaxFrameBytes is the 4 MiB cap inherited from the
// pre-config implementation. It is applied when GSCConfig.MaxFrameBytes
// is left at its zero value.
const DefaultGSCMaxFrameBytes uint32 = 4 * 1024 * 1024

// GameConfig groups game-domain configuration. These fields are consumed
// by the dispatcher and game services, not by infrastructure code.
type GameConfig struct {
	HoleInterval        int
	TableTimeout        int
	GetTblLogInterval   int
	ChatServer          string
	ShowStartedRooms    bool
	ProxyKey            string
	Templates           string
	LCNRanking          string
	GGCupFile           string
	ShowStartedRoomInfo bool
}

// Config is the typed application configuration.
type Config struct {
	Server    ServerConfig
	Log       LogConfig
	Metrics   MetricsConfig
	GSC       GSCConfig
	Game      GameConfig
	Providers map[string]*ProviderConfig
}

// AuthConfig contains only the authentication-provider slice of the full
// Config. auth.Service depends on this value type instead of *Config so
// that it only sees what it needs.
type AuthConfig struct {
	Providers map[string]*ProviderConfig
}

// Provider returns the (lazily-allocated) provider config for the given
// account type. The lookup is case-insensitive. Callers may mutate the
// returned struct in tests.
func (a *AuthConfig) Provider(accType string) *ProviderConfig {
	if a == nil {
		return &ProviderConfig{}
	}

	if a.Providers == nil {
		a.Providers = map[string]*ProviderConfig{}
	}

	k := strings.ToLower(strings.TrimSpace(accType))

	p, ok := a.Providers[k]
	if !ok {
		p = &ProviderConfig{}
		a.Providers[k] = p
	}

	return p
}

// AuthConfig extracts the auth-specific slice of this Config.
// The returned AuthConfig shares the same Providers map pointer as the
// Config so mutations via Config.Provider remain visible to services
// that hold an AuthConfig.
func (c *Config) AuthConfig() AuthConfig {
	if c.Providers == nil {
		c.Providers = map[string]*ProviderConfig{}
	}

	return AuthConfig{Providers: c.Providers}
}

// Provider returns the (lazily-allocated) provider config for the given
// account type. The lookup is case-insensitive. Callers may mutate the
// returned struct in tests.
func (c *Config) Provider(accType string) *ProviderConfig {
	if c == nil {
		return &ProviderConfig{}
	}

	if c.Providers == nil {
		c.Providers = map[string]*ProviderConfig{}
	}

	k := strings.ToLower(strings.TrimSpace(accType))

	p, ok := c.Providers[k]
	if !ok {
		p = &ProviderConfig{}
		c.Providers[k] = p
	}

	return p
}

type LoadOption func(*Config)

// WithEnvOverrides applies HOST_NAME / LOG_FORMAT / LOG_FILE / METRICS_ADDR /
// PROBE_ADDR / UDP_KEEP_ALIVE_INTERVAL after the configuration file has been
// parsed. Equivalent to calling (*Config).ApplyEnv on the result.
func WithEnvOverrides() LoadOption {
	return func(c *Config) {
		c.ApplyEnv()
	}
}

func Load(path string, opts ...LoadOption) (*Config, error) {
	ext := strings.ToLower(filepath.Ext(path))

	var (
		cfg *Config
		err error
	)

	switch ext {
	case ".yaml", ".yml":
		cfg, err = loadYAML(path)
	default:
		cfg, err = loadLegacyConf(path)
	}

	if err != nil {
		return nil, err
	}

	for _, opt := range opts {
		opt(cfg)
	}

	return cfg, nil
}

func loadLegacyConf(path string) (*Config, error) {
	raw := map[string]string{}

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		raw[key] = val
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return configFromRaw(raw), nil
}

func loadYAML(path string) (*Config, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var parsed map[string]any
	if err := yaml.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("parse yaml config: %w", err)
	}

	raw := map[string]string{}
	for k, v := range parsed {
		raw[strings.TrimSpace(k)] = toRawString(v)
	}

	return configFromRaw(raw), nil
}

// configFromRaw is the internal bridge between the file parser (which
// produces a flat string map) and the typed Config struct. The raw map
// is no longer retained on Config.
func configFromRaw(raw map[string]string) *Config {
	cfg := &Config{
		Server: ServerConfig{
			Host:     strOr(raw["host"], "localhost"),
			Port:     intOr(raw["port"], 34001),
			HolePort: intOr(raw["hole_port"], 3708),
		},
		Log: LogConfig{
			Format:    raw["log_format"],
			File:      raw["log_file"],
			AccessLog: raw["access_log"],
			ErrorLog:  raw["error_log"],
		},
		Metrics: MetricsConfig{
			Addr:      raw["metrics_addr"],
			ProbeAddr: raw["probe_addr"],
		},
		GSC: GSCConfig{
			MaxFrameBytes: uint32OrDefault(raw["gsc_max_frame_bytes"], DefaultGSCMaxFrameBytes),
		},
		Game: GameConfig{
			HoleInterval:        intOr(raw["hole_int"], 1000),
			TableTimeout:        intOr(raw["table_timeout"], 10000),
			GetTblLogInterval:   intOr(raw["gettbl_log_interval"], 1),
			ChatServer:          raw["chat_server"],
			ShowStartedRooms:    intOr(raw["show_started_rooms"], 0) != 0,
			ProxyKey:            raw["proxy_key"],
			Templates:           strOr(raw["templates"], "./templates"),
			LCNRanking:          raw["lcn_ranking"],
			GGCupFile:           raw["gg_cup_file"],
			ShowStartedRoomInfo: parseBoolish(raw["show_started_room_info"]),
		},
		Providers: map[string]*ProviderConfig{},
	}

	// Provider blocks: scan keys of the form "<accType>_host" and lift the
	// matching {key, server_name} fields into the typed Provider entry.
	for k, v := range raw {
		switch {
		case strings.HasSuffix(k, "_host"):
			cfg.Provider(strings.TrimSuffix(k, "_host")).Host = v
		case strings.HasSuffix(k, "_key") && k != "proxy_key":
			cfg.Provider(strings.TrimSuffix(k, "_key")).Key = v
		case strings.HasSuffix(k, "_server_name"):
			cfg.Provider(strings.TrimSuffix(k, "_server_name")).ServerName = v
		}
	}

	return cfg
}

// parseBoolish accepts the the "0"/"1"/"true"/"false" values used in the
// config files.
func parseBoolish(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "", "0", "false", "no", "off":
		return false
	default:
		return true
	}
}

func (c *Config) ApplyEnv() {
	type envConfig struct {
		HostName             string `env:"HOST_NAME"`
		UDPKeepAliveInterval string `env:"UDP_KEEP_ALIVE_INTERVAL"`
		LogFormat            string `env:"LOG_FORMAT"`
		LogFile              string `env:"LOG_FILE"`
		MetricsAddr          string `env:"METRICS_ADDR"`
		ProbeAddr            string `env:"PROBE_ADDR"`
	}

	var e envConfig
	if err := env.Parse(&e); err != nil {
		return
	}

	if e.HostName != "" {
		c.Game.ChatServer = e.HostName
		c.Server.HostName = e.HostName
	}

	if e.UDPKeepAliveInterval != "" {
		c.Game.HoleInterval = intOr(e.UDPKeepAliveInterval, c.Game.HoleInterval)
	}

	if e.LogFormat != "" {
		c.Log.Format = e.LogFormat
	}

	if e.LogFile != "" {
		c.Log.File = e.LogFile
	}

	if e.MetricsAddr != "" {
		c.Metrics.Addr = e.MetricsAddr
	}

	if e.ProbeAddr != "" {
		c.Metrics.ProbeAddr = e.ProbeAddr
	}
}

func strOr(v, d string) string {
	if v == "" {
		return d
	}

	return v
}

func intOr(v string, d int) int {
	n, err := strconv.Atoi(strings.TrimSpace(v))
	if err != nil {
		return d
	}

	return n
}

// uint32OrDefault parses v as an unsigned 32-bit integer, returning d
// when v is empty or otherwise unparseable.
func uint32OrDefault(v string, d uint32) uint32 {
	n, err := strconv.ParseUint(strings.TrimSpace(v), 10, 32)
	if err != nil {
		return d
	}

	return uint32(n)
}

func toRawString(v any) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(x)
	case int:
		return strconv.Itoa(x)
	case int64:
		return strconv.FormatInt(x, 10)
	case uint64:
		return strconv.FormatUint(x, 10)
	case float64:
		return strconv.FormatInt(int64(x), 10)
	case bool:
		if x {
			return "1"
		}

		return "0"
	default:
		return strings.TrimSpace(fmt.Sprint(v))
	}
}
