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

// Package handler is the GSC protocol dispatcher.
//
// It owns the per-connection request loop wiring (HandleWithMeta), the
// `open`/`go` route table (dispatch_routes.go), and the typed error
// surface (errors.go). Heavy business logic — authentication, room
// lifecycle, game stats, ranking, rendering, and session bookkeeping —
// lives in the dedicated services under internal/server/{auth,room,
// game,ranking,render,session}; this package wires them together and
// adapts their results to gsc.Command output.
package gsc
