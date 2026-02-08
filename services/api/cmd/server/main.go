package main

import (
	"log"
	"net/http"

	"github.com/yxshee/marketplace-gumroad-inspired/services/api/internal/config"
	"github.com/yxshee/marketplace-gumroad-inspired/services/api/internal/http/router"
)

func main() {
	cfg := config.Load()
	r, err := router.New(cfg)
	if err != nil {
		log.Fatalf("router initialization failed: %v", err)
	}

	addr := ":" + cfg.Port
	log.Printf("api listening on %s", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
