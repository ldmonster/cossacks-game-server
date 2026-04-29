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
	metricsstorage "github.com/deckhouse/deckhouse/pkg/metrics-storage"
	"go.uber.org/zap"
)

// NewStorage builds an isolated MetricStorage with a zap-backed Deckhouse logger and
// registers all cossacksd metrics via RegisterCossacksdMetrics.
func NewStorage(z *zap.Logger) (*metricsstorage.MetricStorage, error) {
	dh := NewDeckhouseLogger(z)
	ms := metricsstorage.NewMetricStorage(
		metricsstorage.WithNewRegistry(),
		metricsstorage.WithLogger(dh.Named("metrics-storage")),
	)

	if err := RegisterCossacksdMetrics(ms); err != nil {
		return nil, err
	}

	ms.GaugeSet(UpMetricName, 1, nil)

	return ms, nil
}

// TCPClientConnected records a new TCP client (nil ms is a no-op).
func TCPClientConnected(ms *metricsstorage.MetricStorage) {
	if ms == nil {
		return
	}

	ms.CounterAdd(TCPConnectionsTotalMetricName, 1, nil)
	ms.GaugeAdd(TCPActiveClientsMetricName, 1, nil)
}

// TCPClientDisconnected records a closed TCP client (nil ms is a no-op).
func TCPClientDisconnected(ms *metricsstorage.MetricStorage) {
	if ms == nil {
		return
	}

	ms.CounterAdd(TCPDisconnectionsTotalMetricName, 1, nil)
	ms.GaugeAdd(TCPActiveClientsMetricName, -1, nil)
}

// IncGSCCommand increments the per-command received counter (nil ms is a no-op).
func IncGSCCommand(ms *metricsstorage.MetricStorage, cmd string) {
	if ms == nil {
		return
	}

	ms.CounterAdd(GSCCommandsTotalMetricName, 1, map[string]string{LabelCmd: cmd})
}

// ObserveGSCRequest records handler latency for one GSC command (nil ms is a no-op).
func ObserveGSCRequest(ms *metricsstorage.MetricStorage, cmd string, seconds float64) {
	if ms == nil {
		return
	}

	ms.HistogramObserve(
		GSCRequestDurationSecondsMetricName,
		seconds,
		map[string]string{LabelCmd: cmd},
		gscRequestDurationBuckets,
	)
}

// IncSTUNPacket counts one successfully processed STUN packet (nil ms is a no-op).
func IncSTUNPacket(ms *metricsstorage.MetricStorage) {
	if ms == nil {
		return
	}

	ms.CounterAdd(STUNPacketsTotalMetricName, 1, nil)
}

// IncSTUNError counts a rejected STUN packet (nil ms is a no-op). Use STUNReason* constants for reason.
func IncSTUNError(ms *metricsstorage.MetricStorage, reason string) {
	if ms == nil {
		return
	}

	ms.CounterAdd(STUNErrorsTotalMetricName, 1, map[string]string{LabelReason: reason})
}
