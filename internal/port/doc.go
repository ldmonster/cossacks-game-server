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

// Package port collects the small, role-shaped interfaces that the
// application layer consumes (KVStore, RankingService, RoomRepository,
// PlayerRepository, AuthProvider, GameService, SessionStore,
// TemplateRenderer, RequestHandler, HandleResult, …).
//
// Each interface declares only the methods one consumer needs, per
// the Interface Segregation Principle. Implementations live under
// internal/adapter or internal/transport and are wired at the
// composition root in cmd/cossacksd.
package port
