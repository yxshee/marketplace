package main

import (
	"errors"
	"log"
	"net/http"
	"time"

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
	server := &http.Server{
		Addr:              addr,
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("server failed: %v", err)
	}
}
