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

package identity

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/ldmonster/cossacks-game-server/internal/domain/identity"
)

// EnterKind enumerates the the try_enter outcomes.
type EnterKind int

const (
	// EnterRender means render an enter.tmpl page (TYPE may be empty).
	EnterRender EnterKind = iota
	// EnterRenderErrorPage means render error_enter.tmpl (Message set).
	EnterRenderErrorPage
	// EnterSuccess means a player can be admitted; Nick/Account hold the
	// values to write into the session before rendering ok_enter.tmpl.
	EnterSuccess
)

// EnterDecision is the discriminated outcome of TryEnterDecide. The
// handler interprets it: applies session mutations, calls the player
// store / session manager / room service as appropriate, and selects
// the template to render.
type EnterDecision struct {
	Kind EnterKind

	// EnterRender / EnterRenderErrorPage payload.
	LoginType string
	Message   string // err text for renderEnter, error_text for error_enter
	LoggedIn  string
	Nick      string
	ID        string

	// EnterSuccess payload.
	Account *identity.AccountInfo

	// DevFlag toggles connection.Session.Dev when non-nil.
	DevFlag *bool
	// SetHeight sets connection.Session.WindowH when non-zero.
	SetHeight int
	// ClearAccount asks the handler to set Session.Account = nil before
	// rendering the enter screen (RESET branch).
	ClearAccount bool

	// RunPostAccountAction asks the handler to fire
	// `post_account_action("enter")` after the session has been
	// admitted. Set only for the LOGGED_IN + existing-account branch
	RunPostAccountAction bool

	// Err carries the typed observability error for HandleResult.Err.
	Err error
}

// EnterRequest is the parsed input to TryEnterDecide; fields
// open-route parameters relevant to the decision.
type EnterRequest struct {
	Nick       string
	LoginType  string
	Password   string
	Reset      bool
	LoggedIn   bool
	HeightStr  string
	CurrentAcc *identity.AccountInfo
	ClientIP   string
}

// ParseEnterRequest extracts an EnterRequest from a decoded params map.
// The current account (if any) must be supplied separately because
// auth has no view of model.Connection.
func ParseEnterRequest(
	p map[string]string,
	currentAcc *identity.AccountInfo,
	clientIP string,
) EnterRequest {
	return EnterRequest{
		Nick:       strings.TrimSpace(p["NICK"]),
		LoginType:  strings.TrimSpace(p["TYPE"]),
		Password:   strings.TrimSpace(p["PASSWORD"]),
		Reset:      p["RESET"] != "",
		LoggedIn:   p["LOGGED_IN"] != "",
		HeightStr:  strings.TrimSpace(p["HEIGHT"]),
		CurrentAcc: currentAcc,
		ClientIP:   clientIP,
	}
}

// stripDevSuffix removes the earlier `#dev4231` developer marker and
// reports whether it was present.
func stripDevSuffix(nick string) (string, bool) {
	const marker = "#dev4231"
	if strings.HasSuffix(nick, marker) {
		return strings.TrimSuffix(nick, marker), true
	}

	return nick, false
}

// TryEnterDecide is the pure-orchestration core of the earlier
// try_enter open route. It performs the LCN/WCL HTTP call where
// needed (via *Service.Authenticate) but does not touch state, the
// session manager, or the renderer; the caller is responsible for
// applying side effects in the order returned via EnterDecision.
func (s *Service) TryEnterDecide(ctx context.Context, req EnterRequest) EnterDecision {
	d := EnterDecision{}

	nick, isDev := stripDevSuffix(req.Nick)
	devVal := isDev
	d.DevFlag = &devVal

	// explicit logout flow returns enter screen.
	if req.Reset {
		d.Kind = EnterRender
		d.ClearAccount = true

		return d
	}

	// already logged-in account can proceed without password branch.
	if req.LoggedIn {
		if req.CurrentAcc != nil && req.CurrentAcc.Login != "" {
			d.Kind = EnterSuccess
			d.Account = req.CurrentAcc
			d.Nick = SanitizeAccountNick(req.CurrentAcc.Login)
			d.RunPostAccountAction = true

			return d
		}

		d.Kind = EnterRender

		return d
	}

	// LCN/WCL branch asks for login/password and performs remote auth.
	if req.LoginType == "LCN" || req.LoginType == "WCL" {
		if nick == "" {
			d.Kind = EnterRender
			d.LoginType = req.LoginType
			d.Message = "enter nick"
			d.Err = ErrEnterMissingNick{LoginType: req.LoginType}

			return d
		}

		if req.Password == "" {
			d.Kind = EnterRender
			d.LoginType = req.LoginType
			d.Message = "enter password"
			d.Err = ErrEnterMissingPassword{LoginType: req.LoginType}

			return d
		}

		acc, err := s.Authenticate(ctx, AuthRequest{
			LoginType: req.LoginType,
			Login:     nick,
			Password:  req.Password,
			ClientIP:  req.ClientIP,
		})
		if err != nil {
			errText := authErrMessage(err)
			d.Kind = EnterRender
			d.LoginType = req.LoginType
			d.Message = errText
			d.Err = ErrEnterAuthFailed{Message: errText}

			return d
		}

		d.Kind = EnterSuccess
		d.Account = &identity.AccountInfo{
			Type:    identity.AccountType(acc.Type),
			Login:   acc.Login,
			ID:      acc.ID,
			Profile: acc.Profile,
		}
		d.Nick = SanitizeAccountNick(acc.Login)

		return d
	}

	// Plain guest nick branch.
	if h, err := strconv.Atoi(req.HeightStr); err == nil {
		d.SetHeight = h
	}

	if nerr := ValidateNick(nick); nerr != NickOK {
		d.Kind = EnterRenderErrorPage
		d.Message = nerr.Message()
		d.Err = ErrEnterIllegalNick{Message: nerr.Message()}

		return d
	}

	d.Kind = EnterSuccess
	d.Nick = TruncateNick(nick)

	return d
}

// Typed observability errors returned via EnterDecision.Err. These
// mirror the handler-package errors that previously lived inline; the
// handler keeps thin wrappers (or aliases) for backwards compatibility.

// ErrEnterMissingNick: LCN/WCL nick prompt branch.
type ErrEnterMissingNick struct{ LoginType string }

func (e ErrEnterMissingNick) Error() string {
	return "auth: try_enter: missing nick (" + e.LoginType + ")"
}

// ErrEnterMissingPassword: LCN/WCL password prompt branch.
type ErrEnterMissingPassword struct{ LoginType string }

func (e ErrEnterMissingPassword) Error() string {
	return "auth: try_enter: missing password (" + e.LoginType + ")"
}

// ErrEnterAuthFailed wraps the user-facing message from the upstream
// account server.
type ErrEnterAuthFailed struct{ Message string }

func (e ErrEnterAuthFailed) Error() string {
	return "auth: try_enter: account auth failed: " + e.Message
}

// ErrEnterIllegalNick wraps the message produced by ValidateNick.
type ErrEnterIllegalNick struct{ Message string }

func (e ErrEnterIllegalNick) Error() string {
	return "auth: try_enter: illegal nick: " + e.Message
}

// IsEnterAuthFailed reports whether err originated from a failed
// upstream auth call (used by tests to distinguish branches).
func IsEnterAuthFailed(err error) bool {
	var x ErrEnterAuthFailed

	return errors.As(err, &x)
}

// authErrMessage maps a typed Authenticate error to the user-facing
// string the the reference implementation returned. The byte sequence of
// these strings is part of the GSC wire contract.
func authErrMessage(err error) string {
	var unreachable ErrProviderUnreachable
	if errors.As(err, &unreachable) {
		return "problem with " + unreachable.ServerName + " server"
	}

	var bad ErrBadCredentials
	if errors.As(err, &bad) {
		return "incorrect login or password"
	}

	var misconfig ErrProviderMisconfigured
	if errors.As(err, &misconfig) {
		return "problem with " + misconfig.ServerName + " server"
	}

	return err.Error()
}
