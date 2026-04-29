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

// Package metrics provides centralized metric names and registration for cossacksd,
// following the layout used by Deckhouse deckhouse-controller/internal/metrics:
// name constants, label key constants (labels.go), and Register* functions that take
// metricsstorage.Storage during init.
package metrics

import (
	"fmt"

	metricsstorage "github.com/deckhouse/deckhouse/pkg/metrics-storage"
	"github.com/deckhouse/deckhouse/pkg/metrics-storage/options"
)

// Histogram buckets for GSC handler latency (seconds).
var gscRequestDurationBuckets = []float64{
	0.0001, 0.0005, 0.001, 0.002, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5,
}

// Metric name constants (same style as deckhouse-controller/internal/metrics/metrics.go).
const (
	// Process / TCP
	UpMetricName                        = "cossacks_up"
	TCPConnectionsTotalMetricName       = "cossacks_tcp_connections_total"
	TCPDisconnectionsTotalMetricName    = "cossacks_tcp_disconnections_total"
	TCPActiveClientsMetricName          = "cossacks_tcp_active_clients"
	GSCCommandsTotalMetricName          = "cossacks_gsc_commands_total"
	GSCRequestDurationSecondsMetricName = "cossacks_gsc_request_duration_seconds"
	STUNPacketsTotalMetricName          = "cossacks_stun_packets_total"
	STUNErrorsTotalMetricName           = "cossacks_stun_errors_total"
)

// RegisterCossacksdMetrics registers all cossacksd Prometheus metrics on storage.
// Call once during startup (see NewStorage).
func RegisterCossacksdMetrics(metricStorage metricsstorage.Storage) error {
	if err := registerTCPMetrics(metricStorage); err != nil {
		return fmt.Errorf("register tcp metrics: %w", err)
	}

	if err := registerGSCMetrics(metricStorage); err != nil {
		return fmt.Errorf("register gsc metrics: %w", err)
	}

	if err := registerSTUNMetrics(metricStorage); err != nil {
		return fmt.Errorf("register stun metrics: %w", err)
	}

	return nil
}

func registerTCPMetrics(metricStorage metricsstorage.Storage) error {
	_, err := metricStorage.RegisterGauge(
		UpMetricName,
		nil,
		options.WithHelp("1 if cossacksd process is running and metrics are initialized."),
	)
	if err != nil {
		return fmt.Errorf("register %s: %w", UpMetricName, err)
	}

	_, err = metricStorage.RegisterCounter(
		TCPConnectionsTotalMetricName,
		nil,
		options.WithHelp("Total TCP client connections accepted."),
	)
	if err != nil {
		return fmt.Errorf("register %s: %w", TCPConnectionsTotalMetricName, err)
	}

	_, err = metricStorage.RegisterCounter(
		TCPDisconnectionsTotalMetricName,
		nil,
		options.WithHelp("Total TCP client disconnects (normal or error)."),
	)
	if err != nil {
		return fmt.Errorf("register %s: %w", TCPDisconnectionsTotalMetricName, err)
	}

	_, err = metricStorage.RegisterGauge(
		TCPActiveClientsMetricName,
		nil,
		options.WithHelp("Current number of connected GSC clients."),
	)
	if err != nil {
		return fmt.Errorf("register %s: %w", TCPActiveClientsMetricName, err)
	}

	return nil
}

func registerGSCMetrics(metricStorage metricsstorage.Storage) error {
	cmdLabels := []string{LabelCmd}

	_, err := metricStorage.RegisterCounter(
		GSCCommandsTotalMetricName,
		cmdLabels,
		options.WithHelp("Total GSC commands received after routing (first command in frame)."),
	)
	if err != nil {
		return fmt.Errorf("register %s: %w", GSCCommandsTotalMetricName, err)
	}

	_, err = metricStorage.RegisterHistogram(
		GSCRequestDurationSecondsMetricName,
		cmdLabels,
		gscRequestDurationBuckets,
		options.WithHelp("Wall time to handle one GSC command (read already done)."),
	)
	if err != nil {
		return fmt.Errorf("register %s: %w", GSCRequestDurationSecondsMetricName, err)
	}

	return nil
}

func registerSTUNMetrics(metricStorage metricsstorage.Storage) error {
	_, err := metricStorage.RegisterCounter(
		STUNPacketsTotalMetricName,
		nil,
		options.WithHelp("Total valid CSHP STUN packets processed."),
	)
	if err != nil {
		return fmt.Errorf("register %s: %w", STUNPacketsTotalMetricName, err)
	}

	reasonLabels := []string{LabelReason}

	_, err = metricStorage.RegisterCounter(
		STUNErrorsTotalMetricName,
		reasonLabels,
		options.WithHelp("STUN UDP packets rejected (parse error or unsupported)."),
	)
	if err != nil {
		return fmt.Errorf("register %s: %w", STUNErrorsTotalMetricName, err)
	}

	return nil
}
