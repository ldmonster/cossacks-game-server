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

package main

import (
	"context"
	"fmt"
	"os"

	metricsstorage "github.com/deckhouse/deckhouse/pkg/metrics-storage"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	integration "github.com/ldmonster/cossacks-game-server/internal/adapter/kvmemory"
	"github.com/ldmonster/cossacks-game-server/internal/adapter/rooms"
	"github.com/ldmonster/cossacks-game-server/internal/adapter/stun"
	gsc "github.com/ldmonster/cossacks-game-server/internal/app/gsc"
	"github.com/ldmonster/cossacks-game-server/internal/platform/config"
	"github.com/ldmonster/cossacks-game-server/internal/platform/health"
	"github.com/ldmonster/cossacks-game-server/internal/platform/logging"
	"github.com/ldmonster/cossacks-game-server/internal/platform/metrics"
	core "github.com/ldmonster/cossacks-game-server/internal/transport/tcp"
)

func main() {
	configPath := "./config/simple-cossacks-server.yaml"

	var logFormatFlag, logFileFlag, metricsAddrFlag, probeAddrFlag string

	rootCmd := &cobra.Command{
		Use:   "cossacksd",
		Short: "Run cossacks game server",
		Run: func(cmd *cobra.Command, args []string) {
			cfg, err := config.Load(configPath, config.WithEnvOverrides())
			if err != nil {
				fmt.Fprintf(os.Stderr, "load config: %v\n", err)
				os.Exit(1)
			}

			if cfg.Server.Host == "localhost" {
				// Keep external behavior but allow containerized port publishing.
				cfg.Server.Host = "0.0.0.0"
			}

			logOpts, err := logging.ResolveOptions(cfg, logFormatFlag, logFileFlag)
			if err != nil {
				fmt.Fprintf(os.Stderr, "logging: %v\n", err)
				os.Exit(1)
			}

			logger, err := logging.New(logOpts)
			if err != nil {
				fmt.Fprintf(os.Stderr, "logging: %v\n", err)
				os.Exit(1)
			}

			defer func() { _ = logger.Sync() }()

			metricsAddr := metrics.ResolveAddr(cfg, metricsAddrFlag)
			probeAddr := health.ResolveProbeAddr(cfg, probeAddrFlag)

			var ms *metricsstorage.MetricStorage
			if metricsAddr != "" {
				ms, err = metrics.NewStorage(logger)
				if err != nil {
					fmt.Fprintf(os.Stderr, "metrics: %v\n", err)
					os.Exit(1)
				}
			}

			store := rooms.NewStore()
			runtimeStorage := integration.NewRuntimeStorage()
			ctrl := gsc.NewController(cfg, store, runtimeStorage, logger)

			s := &core.Server{
				Host:    cfg.Server.Host,
				Port:    cfg.Server.Port,
				MaxSize: gscMaxFrameBytes(cfg.GSC),
				Log:     logger,
				Metrics: ms,
				Handler: ctrl,
			}

			stunAddr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.HolePort)
			logger.Info("listen udp (stun)", zap.String("addr", stunAddr))

			g, ctx := errgroup.WithContext(context.Background())
			g.Go(func() error {
				return stun.ServeUDP(
					ctx,
					stunAddr,
					runtimeStorage,
					cfg.Game.HoleInterval,
					logger,
					ms,
				)
			})
			g.Go(func() error {
				return s.ListenAndServe(ctx)
			})

			if metricsAddr != "" {
				g.Go(func() error {
					return metrics.ListenAndServe(ctx, metricsAddr, ms, runtimeStorage, logger)
				})

				if probeAddr != "" && probeAddr != metricsAddr {
					g.Go(func() error {
						return health.ListenAndServe(ctx, probeAddr, runtimeStorage, logger)
					})
				}
			} else if probeAddr != "" {
				g.Go(func() error {
					return health.ListenAndServe(ctx, probeAddr, runtimeStorage, logger)
				})
			}

			if err := g.Wait(); err != nil {
				logger.Error("server exit", zap.Error(err))
				os.Exit(1)
			}
		},
	}

	rootCmd.Flags().StringVar(&configPath, "config", configPath, "config file (.conf or .yaml)")
	rootCmd.Flags().StringVar(
		&logFormatFlag,
		"log-format",
		"",
		`log encoding: "user" (console) or "json"; empty uses config/env (default user)`,
	)
	rootCmd.Flags().StringVar(
		&logFileFlag,
		"log-file",
		"",
		"if set, write logs to this path with lumberjack rotation (overrides config/env when non-empty)",
	)
	rootCmd.Flags().StringVar(
		&metricsAddrFlag,
		"metrics-addr",
		"",
		`if set (e.g. ":9100"), serve Prometheus metrics at /metrics; overrides config/env METRICS_ADDR`,
	)
	rootCmd.Flags().StringVar(
		&probeAddrFlag,
		"probe-addr",
		"",
		`if set (e.g. ":8080"), serve /livez and /readyz only; when metrics-addr is set to the same value, probes are on that server instead (overrides config/env PROBE_ADDR)`,
	)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

// gscMaxFrameBytes returns the configured GSC frame cap, falling back
// to the package-level default when the operator has not specified
// one in the configuration file.
func gscMaxFrameBytes(c config.GSCConfig) uint32 {
	if c.MaxFrameBytes == 0 {
		return config.DefaultGSCMaxFrameBytes
	}

	return c.MaxFrameBytes
}
