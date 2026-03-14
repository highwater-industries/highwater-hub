package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"time"

	_ "github.com/lib/pq" // Postgres driver — the underscore import registers it

	"myproject/internal/fitness"
	"myproject/internal/jobs"
	"myproject/internal/nflstats"
	"myproject/internal/server"
	"myproject/internal/user"
)

func main() {
	if err := run(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt)
	defer stop()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	// ---- Database ----
	dbURL := getEnv("DATABASE_URL", "postgres://appuser:apppass@localhost:5432/nflstats?sslmode=disable")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer db.Close()

	// Verify the connection actually works
	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("ping database: %w", err)
	}
	logger.Info("connected to database")

	// ---- Python service client ----
	pythonURL := getEnv("PYTHON_SERVICE_URL", "http://localhost:3142")
	jobsClient := jobs.NewClient(pythonURL)

	// ---- Stores ----
	userStore := user.NewMemoryStore()
	jobsStore := jobs.NewPostgresStore(db)
	playerStore := nflstats.NewPostgresStore(db)
	statStore := nflstats.NewPostgresStatStore(db)
	gameStore := nflstats.NewPostgresGameStore(db)
	rankingStore := nflstats.NewPostgresRankingStore(db)
	fitnessStore := fitness.NewPostgresStore(db)

	// ---- Fitness table init ----
	if err := fitnessStore.EnsureTables(ctx); err != nil {
		return fmt.Errorf("fitness table init: %w", err)
	}
	if err := fitnessStore.SeedExercises(ctx); err != nil {
		return fmt.Errorf("fitness seed exercises: %w", err)
	}
	logger.Info("fitness tables initialized")

	// ---- Server ----
	srv := server.NewServer(server.Config{
		Logger:       logger,
		UserStore:    userStore,
		JobsClient:   jobsClient,
		JobsStore:    jobsStore,
		PlayerStore:  playerStore,
		StatStore:    statStore,
		GameStore:    gameStore,
		RankingStore: rankingStore,
		FitnessStore: fitnessStore,
	})

	httpServer := &http.Server{
		Addr:    ":3141",
		Handler: srv,
	}

	go func() {
		logger.Info("server starting", "addr", httpServer.Addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server error", "err", err)
		}
	}()

	<-ctx.Done()
	logger.Info("shutting down gracefully")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return httpServer.Shutdown(shutdownCtx)
}

// getEnv returns the value of an environment variable, or a default if unset.
func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return fallback
}
