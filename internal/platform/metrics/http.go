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

package metrics

import (
	"context"
	"net/http"

	metricsstorage "github.com/deckhouse/deckhouse/pkg/metrics-storage"
	"go.uber.org/zap"

	"github.com/ldmonster/cossacks-game-server/internal/platform/health"
	"github.com/ldmonster/cossacks-game-server/internal/port"
)

// ListenAndServe exposes /metrics (when ms != nil), /livez, and /readyz on addr until ctx is cancelled.
func ListenAndServe(
	ctx context.Context,
	addr string,
	ms *metricsstorage.MetricStorage,
	storage port.KVStore,
	log *zap.Logger,
) error {
	if addr == "" {
		<-ctx.Done()
		return nil
	}

	mux := http.NewServeMux()
	health.MountProbes(mux, storage, log)

	if ms != nil {
		mux.Handle("/metrics", ms.Handler())
	}

	return health.ListenAndServeMux(ctx, addr, mux, log, "metrics")
}
