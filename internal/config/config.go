package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Host               string
	Port               int
	HolePort           int
	HoleInterval       int
	Templates          string
	AccessLog          string
	ErrorLog           string
	TableTimeout       int
	GetTblLogInterval  int
	ChatServer         string
	ShowStartedRooms   bool
	ProxyKey           string
	Raw                map[string]string
}

func Load(path string) (*Config, error) {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".yaml", ".yml":
		return loadYAML(path)
	default:
		return loadLegacyConf(path)
	}
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

func configFromRaw(raw map[string]string) *Config {
	cfg := &Config{
		Host:              strOr(raw["host"], "localhost"),
		Port:              intOr(raw["port"], 34001),
		HolePort:          intOr(raw["hole_port"], 3708),
		HoleInterval:      intOr(raw["hole_int"], 300),
		Templates:         strOr(raw["templates"], "./templates"),
		AccessLog:         raw["access_log"],
		ErrorLog:          raw["error_log"],
		TableTimeout:      intOr(raw["table_timeout"], 10000),
		GetTblLogInterval: intOr(raw["gettbl_log_interval"], 1),
		ChatServer:        raw["chat_server"],
		ShowStartedRooms:  intOr(raw["show_started_rooms"], 0) != 0,
		ProxyKey:          raw["proxy_key"],
		Raw:               raw,
	}
	return cfg
}

func (c *Config) ApplyEnv() {
	if host := os.Getenv("HOST_NAME"); host != "" {
		c.ChatServer = host
		c.Raw["chat_server"] = host
	}
	if interval := os.Getenv("UDP_KEEP_ALIVE_INTERVAL"); interval != "" {
		c.HoleInterval = intOr(interval, c.HoleInterval)
		c.Raw["hole_int"] = strconv.Itoa(c.HoleInterval)
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
