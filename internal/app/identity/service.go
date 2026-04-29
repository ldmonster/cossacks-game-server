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

// Package auth owns the authenticate / post-account-action HTTP flow
// extracted from the god-object controller per.
// The handler package now delegates here instead of
// inlining HTTP calls.
package identity

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/ldmonster/cossacks-game-server/internal/platform/config"
	"github.com/ldmonster/cossacks-game-server/internal/port"
)

// Service performs LCN/WCL account-server HTTP calls
type Service struct {
	cfg  config.AuthConfig
	http *http.Client
	log  *zap.Logger
}

// NewService constructs a Service. A nil httpClient is replaced with a
// default 5-second-timeout client (wire fidelity).
func NewService(cfg config.AuthConfig, httpClient *http.Client, log *zap.Logger) *Service {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 5 * time.Second}
	}

	if log == nil {
		log = zap.NewNop()
	}

	return &Service{cfg: cfg, http: httpClient, log: log}
}

// Account is the per-call result of Authenticate; mirrors the earlier
// map[string]string the controller used to pass around.
type Account struct {
	Type    string
	Login   string
	ID      string
	Profile string
}

// AuthRequest carries the inputs required to authenticate a player
// against an upstream account provider.
type AuthRequest struct {
	LoginType string
	Login     string
	Password  string
	ClientIP  string
}

// ErrProviderUnreachable indicates the account provider could not be
// reached (network error, bad HTTP status, malformed payload, or
// missing configuration). ServerName is the user-visible identifier
// used to render the "problem with X server" message.
type ErrProviderUnreachable struct {
	ServerName string
}

func (e ErrProviderUnreachable) Error() string {
	return "account provider unreachable: " + e.ServerName
}

// ErrBadCredentials indicates the provider responded successfully but
// rejected the supplied login/password.
type ErrBadCredentials struct{}

func (ErrBadCredentials) Error() string { return "incorrect login or password" }

// ErrProviderMisconfigured indicates the provider host or key is
// missing in the server configuration.
type ErrProviderMisconfigured struct {
	ServerName string
}

func (e ErrProviderMisconfigured) Error() string {
	return "account provider misconfigured: " + e.ServerName
}

// Authenticate performs the remote logon round-trip against the account
// server identified by req.LoginType (LCN/WCL). Returns (account, nil)
// on success or (zero, typed error) on failure. The user-visible error
// string is computed by callers from the typed error class.
func (s *Service) Authenticate(
	ctx context.Context,
	req AuthRequest,
) (Account, error) {
	loginType := req.LoginType
	login := req.Login
	password := req.Password
	clientIP := req.ClientIP

	prov := s.cfg.Provider(loginType)
	host := prov.Host
	secret := prov.Key

	serverName := prov.ServerName
	if serverName == "" {
		serverName = host
	}

	if host == "" || secret == "" {
		return Account{}, ErrProviderMisconfigured{ServerName: serverName}
	}

	form := url.Values{}
	form.Set("action", "logon")
	form.Set("key", secret)
	form.Set("login", login)
	form.Set("password", password)

	endpoint := "http://" + host + "/api/server.php"

	httpReq, err := http.NewRequestWithContext(
		ctx, http.MethodPost, endpoint, bytes.NewBufferString(form.Encode()),
	)
	if err != nil {
		return Account{}, ErrProviderUnreachable{ServerName: serverName}
	}

	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	httpReq.Header.Set("X-Client-IP", clientIP)

	resp, err := s.http.Do(httpReq)
	if err != nil {
		s.log.Warn("auth request failed", zap.Error(err))
		return Account{}, ErrProviderUnreachable{ServerName: serverName}
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		s.log.Warn("auth bad response",
			zap.String("endpoint", endpoint),
			zap.String("http_status", resp.Status),
		)

		return Account{}, ErrProviderUnreachable{ServerName: serverName}
	}

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return Account{}, ErrProviderUnreachable{ServerName: serverName}
	}

	var payload struct {
		Success bool   `json:"success"`
		ID      any    `json:"id"`
		Profile string `json:"profile"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		s.log.Warn("auth bad json",
			zap.String("endpoint", endpoint),
			zap.Error(err),
		)

		return Account{}, ErrProviderUnreachable{ServerName: serverName}
	}

	if !payload.Success {
		return Account{}, ErrBadCredentials{}
	}

	id := fmt.Sprintf("%v", payload.ID)

	acc := Account{
		Type:    loginType,
		Login:   login,
		ID:      id,
		Profile: payload.Profile,
	}

	if lcnHost := s.cfg.Provider("LCN").Host; loginType == "LCN" && lcnHost != "" {
		acc.Profile = "http://" + lcnHost + "/lang_redir.php?path=player.php?plid=" + id
	}

	return acc, nil
}

// PostAccountAction reports a player event to the auth provider. Errors
// are logged but never returned (wire fidelity: best-effort fire-and-forget).
func (s *Service) PostAccountAction(
	ctx context.Context,
	accountType, accountID, clientIP, action string,
	payload map[string]any,
) {
	if accountType == "" || accountID == "" {
		return
	}

	prov := s.cfg.Provider(strings.ToLower(accountType))
	host := prov.Host

	key := prov.Key
	if host == "" || key == "" {
		return
	}

	form := url.Values{}
	form.Set("action", action)
	form.Set("time", fmt.Sprintf("%d", time.Now().UTC().Unix()))
	form.Set("key", key)
	form.Set("account_id", accountID)

	if payload != nil {
		if raw, err := json.Marshal(payload); err == nil {
			form.Set("data", string(raw))
		}
	}

	endpoint := "http://" + host + "/api/server.php"

	req, err := http.NewRequestWithContext(
		ctx, http.MethodPost, endpoint, bytes.NewBufferString(form.Encode()),
	)
	if err != nil {
		return
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "my-server.example bot")
	req.Header.Set("X-Client-IP", clientIP)

	resp, err := s.http.Do(req)
	if err != nil {
		s.log.Warn("account action request failed",
			zap.String("action", action),
			zap.Error(err),
		)

		return
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		s.log.Warn("account action bad response",
			zap.String("action", action),
			zap.String("http_status", resp.Status),
		)
	}
}

// Compile-time interface check.
var _ port.AuthService = (*Service)(nil)
