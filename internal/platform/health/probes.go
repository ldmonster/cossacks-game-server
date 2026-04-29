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

// Package health implements Kubernetes-style HTTP probes (/livez, /readyz).
package health

import (
	"context"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/ldmonster/cossacks-game-server/internal/port"
)

const readyPingTimeout = 2 * time.Second

// MountProbes registers /livez and /readyz on mux. Readiness uses runtime storage when storage is non-nil.
func MountProbes(mux *http.ServeMux, storage port.KVStore, log *zap.Logger) {
	mux.HandleFunc("/livez", livezHandler())
	mux.HandleFunc("/readyz", readyzHandler(storage, log))
}

func livezHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	}
}

func readyzHandler(storage port.KVStore, log *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")

		if storage == nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte("runtime storage not configured\n"))

			return
		}

		pctx, cancel := context.WithTimeout(r.Context(), readyPingTimeout)
		defer cancel()

		if err := storage.Ping(pctx); err != nil {
			log.Warn("readiness probe failed", zap.Error(err))
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte("runtime storage unavailable\n"))

			return
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	}
}
