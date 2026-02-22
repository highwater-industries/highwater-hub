package server 

import (
    "log/slog"
    "net/http"

    "myproject/internal/user"
)

type Config struct {
    Logger *slog.Logger
    UserStore user.Store
}

func NewServer(cfg Config) http.Handler {
    
    mux := http.NewServeMux()
    addRoutes(mux, cfg)

    var handler http.Handler = mux
    handler = withLogging(cfg.Logger, handler)
    handler = withRecovery(cfg.Logger, handler)
    return handler 
}
