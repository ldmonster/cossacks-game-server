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

package health

import (
	"context"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/ldmonster/cossacks-game-server/internal/port"
)

// ListenAndServe exposes only /livez and /readyz until ctx is cancelled.
func ListenAndServe(
	ctx context.Context,
	addr string,
	storage port.KVStore,
	log *zap.Logger,
) error {
	mux := http.NewServeMux()
	MountProbes(mux, storage, log)

	return ListenAndServeMux(ctx, addr, mux, log, "probes")
}

// ListenAndServeMux serves h on addr with graceful shutdown on ctx cancellation.
func ListenAndServeMux(
	ctx context.Context,
	addr string,
	h http.Handler,
	log *zap.Logger,
	name string,
) error {
	srv := &http.Server{
		Addr:              addr,
		Handler:           h,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		<-ctx.Done()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_ = srv.Shutdown(shutdownCtx)
	}()

	log.Info("http listen", zap.String("addr", addr), zap.String("server", name))

	err := srv.ListenAndServe()
	if err == http.ErrServerClosed {
		return nil
	}

	return err
}
