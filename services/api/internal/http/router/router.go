package router

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/yxshee/marketplace-gumroad-inspired/services/api/internal/config"
)

type healthResponse struct {
	Status    string `json:"status"`
	Service   string `json:"service"`
	Timestamp string `json:"timestamp"`
}

func healthHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(healthResponse{
		Status:    "ok",
		Service:   "marketplace-api",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

// New creates a production-ready chi router with baseline middleware and routes.
func New(cfg config.Config) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))

	r.Get("/health", healthHandler)

	r.Route("/api/v1", func(v1 chi.Router) {
		v1.Get("/health", healthHandler)
		_ = cfg // reserved for future route wiring
	})

	return r
}
