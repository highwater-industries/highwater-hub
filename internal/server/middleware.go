package server

import (
    "log/slog"
    "net/http"
    "time"
)

func withLogging(logger *slog.Logger, next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        next.ServeHTTP(w,r)
        logger.Info("request",
                    "method",
                    "path",
                    "duration", time.Since(start),
                )
    })
}


func withRecovery(logger *slog.Logger, next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request){
        defer func() {
            if err:= recover(); err != nil {
                logger.Error("panic recovered", "err", err)
                http.Error(w, "internal server error", http.StatusInternalServerError)
            }
        }()
        next.ServeHTTP(w,r)
    })
}
