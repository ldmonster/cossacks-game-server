package stun

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	"cossacksgameserver/golang/internal/integration"
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

func ServeUDP(ctx context.Context, addr string, redisClient *integration.RedisClient, keepAliveMs int) error {
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
			log.Printf("invalid packet: %v", err)
			continue
		}
		if pkt.Tag != "CSHP" || pkt.Version != 1 {
			log.Printf("unsupported packet: tag=%s version=%d", pkt.Tag, pkt.Version)
			continue
		}
		payload, _ := json.Marshal(map[string]any{
			"host":       remote.IP.String(),
			"port":       remote.Port,
			"version":    pkt.Version,
			"access_key": pkt.AccessKey,
		})
		if err := redisClient.SetPX(ctx, fmt.Sprintf("%d", pkt.PlayerID), string(payload), ttl); err != nil {
			log.Printf("redis set failed: %v", err)
			continue
		}
		_, _ = conn.WriteToUDP([]byte("ok"), remote)
	}
}
