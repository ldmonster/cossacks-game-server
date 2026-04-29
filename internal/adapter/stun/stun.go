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

package stun

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"time"

	metricsstorage "github.com/deckhouse/deckhouse/pkg/metrics-storage"
	"go.uber.org/zap"

	"github.com/ldmonster/cossacks-game-server/internal/platform/metrics"
	"github.com/ldmonster/cossacks-game-server/internal/port"
)

type Packet struct {
	Tag       string
	Version   uint8
	PlayerID  uint32
	AccessKey string
}

func ParsePacket(b []byte) (Packet, error) {
	if len(b) < 25 {
		return Packet{}, fmt.Errorf("invalid packet size")
	}

	tag := string(b[0:4])
	version := b[4]
	playerID := binary.BigEndian.Uint32(b[5:9])
	accessKey := string(b[9:25])
	accessKey = strings.TrimRight(accessKey, "\x00")

	return Packet{Tag: tag, Version: version, PlayerID: playerID, AccessKey: accessKey}, nil
}

func ServeUDP(
	ctx context.Context,
	addr string,
	storage port.KVStore,
	keepAliveMs int,
	logger *zap.Logger,
	ms *metricsstorage.MetricStorage,
) error {
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return err
	}

	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return err
	}
	defer conn.Close()

	buf := make([]byte, 512)
	ttl := time.Duration(float64(keepAliveMs)*1.5) * time.Millisecond

	for {
		n, remote, err := conn.ReadFromUDP(buf)
		if err != nil {
			return err
		}

		pkt, err := ParsePacket(buf[:n])
		if err != nil {
			logger.Warn("invalid packet", zap.Error(err))
			metrics.IncSTUNError(ms, metrics.STUNReasonParseError)

			continue
		}

		if pkt.Tag != "CSHP" || pkt.Version != 1 {
			logger.Warn("unsupported packet",
				zap.String("tag", pkt.Tag),
				zap.Uint8("version", pkt.Version),
			)
			metrics.IncSTUNError(ms, metrics.STUNReasonUnsupportedPacket)

			continue
		}

		payload, _ := json.Marshal(map[string]any{
			"host":       remote.IP.String(),
			"port":       remote.Port,
			"version":    pkt.Version,
			"access_key": pkt.AccessKey,
		})
		if err := storage.SetPX(
			ctx,
			fmt.Sprintf("%d", pkt.PlayerID),
			string(payload),
			ttl,
		); err != nil {
			logger.Warn("runtime storage set failed",
				zap.Uint32("player_id", pkt.PlayerID),
				zap.Error(err),
			)
			metrics.IncSTUNError(ms, metrics.STUNReasonStorageError)

			continue
		}

		metrics.IncSTUNPacket(ms)

		_, _ = conn.WriteToUDP([]byte("ok"), remote)
	}
}
