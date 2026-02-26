package server

import (
	"net/http"

	"myproject/internal/frontend"
	"myproject/internal/jobs"
	"myproject/internal/nflstats"
	"myproject/internal/user"
)

func addRoutes(mux *http.ServeMux, cfg Config) {
	// Health
	mux.HandleFunc("GET /api/health", handleHealth())

	// Users
	mux.HandleFunc("GET /api/users/{id}", user.HandleGetById(cfg.UserStore))
	mux.HandleFunc("POST /api/users", user.HandleCreateSingle(cfg.UserStore))
	mux.HandleFunc("GET /api/users", user.HandleGetAll(cfg.UserStore))

	// Jobs
	mux.HandleFunc("POST /api/jobs/import", jobs.HandleStartImport(cfg.JobsClient))
	mux.HandleFunc("GET /api/jobs/{job_id}", jobs.HandleGetJobStatus(cfg.JobsClient))
	mux.HandleFunc("GET /api/jobs", jobs.HandleListJobs(cfg.JobsStore))

	// NFL Stats
	mux.HandleFunc("GET /api/nflstats/players/{id}", nflstats.HandleGetPlayer(cfg.PlayerStore))
	mux.HandleFunc("GET /api/nflstats/players", nflstats.HandleListPlayers(cfg.PlayerStore))

	// SvelteKit SPA — catch-all for non-API routes
	mux.Handle("/", frontend.Handler())
}
