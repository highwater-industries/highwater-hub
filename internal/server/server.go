package server

import (
	"log/slog"
	"net/http"

	"myproject/internal/fitness"
	"myproject/internal/jobs"
	"myproject/internal/nflstats"
	"myproject/internal/user"
)

type Config struct {
	Logger       *slog.Logger
	UserStore    user.Store
	JobsClient   *jobs.Client
	JobsStore    jobs.Store
	PlayerStore  nflstats.Store
	StatStore    nflstats.StatStore
	GameStore    nflstats.GameStore
	RankingStore nflstats.RankingStore
	FitnessStore fitness.Store
}

func NewServer(cfg Config) http.Handler {

	mux := http.NewServeMux()
	addRoutes(mux, cfg)

	var handler http.Handler = mux
	handler = withLogging(cfg.Logger, handler)
	handler = withRecovery(cfg.Logger, handler)
	return handler
}
