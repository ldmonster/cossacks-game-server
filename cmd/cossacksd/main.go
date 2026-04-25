package main

import (
	"context"
	"flag"
	"log"

	"cossacksgameserver/golang/internal/config"
	"cossacksgameserver/golang/internal/integration"
	"cossacksgameserver/golang/internal/server/commands"
	"cossacksgameserver/golang/internal/server/core"
	"cossacksgameserver/golang/internal/server/state"
)

func main() {
	configPath := flag.String("config", "./config/simple-cossacks-server.yaml", "config file (.conf or .yaml)")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	cfg.ApplyEnv()
	commands.ConfigureTemplateRoots(cfg.Templates)
	if cfg.Host == "localhost" {
		// Keep external behavior but allow containerized port publishing.
		cfg.Host = "0.0.0.0"
	}

	store := state.NewStore()
	ctrl := &commands.Controller{
		Config: cfg,
		Store:  store,
		Redis:  integration.NewRedis("redis:6379"),
	}
	s := &core.Server{
		Host:       cfg.Host,
		Port:       cfg.Port,
		MaxSize:    4 * 1024 * 1024,
		Store:      store,
		Controller: ctrl,
	}
	if err := s.ListenAndServe(context.Background()); err != nil {
		log.Fatal(err)
	}
}
