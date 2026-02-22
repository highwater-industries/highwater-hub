package server

import (
    "net/http"

    "myproject/internal/user"
)

func addRoutes(mux *http.ServeMux, cfg Config) {
    mux.HandleFunc("GET /api/health", handleHealth())
    mux.HandleFunc("GET /api/users/{id}", user.HandleGetById(cfg.UserStore))
    mux.HandleFunc("POST /api/users", user.HandleCreateSingle(cfg.UserStore))
    mux.HandleFunc("GET /api/users", user.HandleGetAll(cfg.UserStore))
}
