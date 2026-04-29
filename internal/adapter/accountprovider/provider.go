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

// Package httpauth provides the concrete HTTP-backed implementation of
// port.AuthProvider used by the LCN/WCL account-server protocol.
// It is a thin port adapter: all
// HTTP calls, response parsing and error classification live here so the
// game-logic layer can depend on the interface, not the transport.
package httpauth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ldmonster/cossacks-game-server/internal/domain/identity"
	"github.com/ldmonster/cossacks-game-server/internal/port"
)

// ErrBadCredentials is returned by Authenticate when the remote service
// reports the login/password pair is invalid (HTTP 200 + success=false).
var ErrBadCredentials = errors.New("httpauth: bad credentials")

// ErrServiceUnavailable is returned for transport-level or non-2xx errors.
var ErrServiceUnavailable = errors.New("httpauth: service unavailable")

// HostMap maps each identity.AccountType to the upstream host (without scheme)
// and shared HMAC key.
type HostMap map[identity.AccountType]Host

// Host describes the upstream account service for a single AccountType.
type Host struct {
	Host string // e.g. "auth.example.com"
	Key  string // shared secret bound into form field "key"
}

// Provider is the concrete port.AuthProvider implementation.
type Provider struct {
	hosts  HostMap
	client *http.Client
}

// New constructs a Provider. A nil http.Client is replaced with a default
// 5-second-timeout client (matches the earlier behavior).
func New(hosts HostMap, client *http.Client) *Provider {
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}

	return &Provider{hosts: hosts, client: client}
}

// Authenticate satisfies port.AuthProvider.
func (p *Provider) Authenticate(
	ctx context.Context,
	accountType identity.AccountType,
	login, password string,
) (identity.AccountInfo, error) {
	host, ok := p.hosts[accountType]
	if !ok || host.Host == "" || host.Key == "" {
		return identity.AccountInfo{}, fmt.Errorf("%w: %s", ErrServiceUnavailable, accountType)
	}

	form := url.Values{}
	form.Set("action", "logon")
	form.Set("key", host.Key)
	form.Set("login", login)
	form.Set("password", password)

	endpoint := "http://" + host.Host + "/api/server.php"

	req, err := http.NewRequestWithContext(
		ctx, http.MethodPost, endpoint, bytes.NewBufferString(form.Encode()),
	)
	if err != nil {
		return identity.AccountInfo{}, fmt.Errorf("%w: %v", ErrServiceUnavailable, err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.client.Do(req)
	if err != nil {
		return identity.AccountInfo{}, fmt.Errorf("%w: %v", ErrServiceUnavailable, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return identity.AccountInfo{}, fmt.Errorf(
			"%w: status %s",
			ErrServiceUnavailable,
			resp.Status,
		)
	}

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return identity.AccountInfo{}, fmt.Errorf("%w: %v", ErrServiceUnavailable, err)
	}

	var payload struct {
		Success bool   `json:"success"`
		ID      any    `json:"id"`
		Profile string `json:"profile"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return identity.AccountInfo{}, fmt.Errorf("%w: %v", ErrServiceUnavailable, err)
	}

	if !payload.Success {
		return identity.AccountInfo{}, ErrBadCredentials
	}

	id := strings.TrimSpace(fmt.Sprintf("%v", payload.ID))

	return identity.AccountInfo{
		Type:    accountType,
		ID:      id,
		Login:   login,
		Profile: payload.Profile,
	}, nil
}

// Compile-time interface check.
var _ port.AuthProvider = (*Provider)(nil)
