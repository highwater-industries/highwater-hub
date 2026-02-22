package server

import (
    "net/http"
    "time"

    "myproject/internal/httputil"
)

type healthResponse struct {
    Status    string `json:"status"`
    Timestamp string `json:"timestamp"`
    Uptime    string `json:"uptime"`
}

var startTime = time.Now()

func handleHealth() http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        httputil.Encode(w, http.StatusOK, healthResponse{
            Status:    "ok",
            Timestamp: time.Now().UTC().Format(time.RFC3339),
            Uptime:    time.Since(startTime).Round(time.Second).String(),
        })
    }
}
