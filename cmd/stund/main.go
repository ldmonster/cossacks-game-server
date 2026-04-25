package main

import (
	"context"
	"log"
	"os"
	"strconv"

	"cossacksgameserver/golang/internal/integration"
	"cossacksgameserver/golang/internal/stun"
)

func main() {
	keepAlive := 1000
	if raw := os.Getenv("UDP_KEEP_ALIVE_INTERVAL"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil {
			keepAlive = n
		}
	}
	redisHost := os.Getenv("REDIS_HOST")
	if redisHost == "" {
		redisHost = "redis:6379"
	}
	redisClient := integration.NewRedis(redisHost)
	if err := stun.ServeUDP(context.Background(), ":3708", redisClient, keepAlive); err != nil {
		log.Fatal(err)
	}
}
