package main

import (
    "context"
    "fmt"
    "log/slog"
    "net/http"
    "os"
    "os/signal"
    "time"

    "myproject/internal/server"
    "myproject/internal/user"
)

func main(){
    if err:=run(context.Background()); err != nil {
        fmt.Fprintf(os.Stderr, "error: %v\n", err)
        os.Exit(1)
    }
}

func run (ctx context.Context) error {
    ctx, stop := signal.NotifyContext(ctx, os.Interrupt)
    defer stop()

    logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
    userStore := user.NewMemoryStore()
    
    srv := server.NewServer(server.Config{
        Logger: logger,
        UserStore: userStore,
    })

    httpServer := &http.Server{
        Addr: ":3141",
        Handler: srv,
    }

    go func() {
        logger.Info("server starting", "addr", httpServer.Addr)
        if err := httpServer.ListenAndServe(); err!= nil && err != http.ErrServerClosed {
            logger.Error("server error", "err", err)
        }
    }()

    <-ctx.Done()
    logger.Info("shutting down gracefully")

    shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    return httpServer.Shutdown(shutdownCtx)
}
