package server

import (
	"net/http"

	"myproject/internal/fantasy"
	"myproject/internal/fitness"
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
	mux.HandleFunc("POST /api/jobs/import/batch", jobs.HandleBatchImport(cfg.JobsClient))
	mux.HandleFunc("POST /api/jobs/import", jobs.HandleStartImport(cfg.JobsClient))
	mux.HandleFunc("POST /api/jobs/abort-all", jobs.HandleAbortAll(cfg.JobsStore, cfg.JobsClient))
	mux.HandleFunc("POST /api/jobs/{id}/abort", jobs.HandleAbortJob(cfg.JobsStore, cfg.JobsClient))
	mux.HandleFunc("POST /api/jobs/cleanup", jobs.HandleCleanupStuck(cfg.JobsStore))
	mux.HandleFunc("GET /api/jobs/summary", jobs.HandleGetJobSummary(cfg.JobsStore))
	mux.HandleFunc("GET /api/jobs/{job_id}", jobs.HandleGetJobStatus(cfg.JobsClient))
	mux.HandleFunc("GET /api/jobs", jobs.HandleListJobs(cfg.JobsStore))

	// Data Management
	mux.HandleFunc("GET /api/data/inventory", jobs.HandleGetInventory(cfg.InventoryStore))
	mux.HandleFunc("GET /api/data/audit", jobs.HandleRunAudit(cfg.InventoryStore))

	// NFL Stats
	mux.HandleFunc("GET /api/nflstats/players/{id}/summary", nflstats.HandleGetPlayerSummary(cfg.PlayerStore, cfg.StatStore))
	mux.HandleFunc("GET /api/nflstats/players/{id}", nflstats.HandleGetPlayer(cfg.PlayerStore))
	mux.HandleFunc("GET /api/nflstats/players", nflstats.HandleListPlayers(cfg.PlayerStore))
	mux.HandleFunc("GET /api/nflstats/stats", nflstats.HandleListStats(cfg.StatStore))
	mux.HandleFunc("GET /api/nflstats/leaders", nflstats.HandleGetLeaders(cfg.StatStore))
	mux.HandleFunc("GET /api/nflstats/games/{game_id}", nflstats.HandleGetGame(cfg.GameStore))
	mux.HandleFunc("GET /api/nflstats/games", nflstats.HandleListGames(cfg.GameStore))
	mux.HandleFunc("GET /api/nflstats/rankings", nflstats.HandleListRankings(cfg.RankingStore))

	// Fitness
	mux.HandleFunc("GET /api/fitness/bodyweight/latest", fitness.HandleGetLatestBodyweight(cfg.FitnessStore))
	mux.HandleFunc("GET /api/fitness/bodyweight", fitness.HandleListBodyweightHistory(cfg.FitnessStore))
	mux.HandleFunc("POST /api/fitness/bodyweight", fitness.HandleLogBodyweight(cfg.FitnessStore))
	mux.HandleFunc("DELETE /api/fitness/bodyweight/{id}", fitness.HandleDeleteBodyweight(cfg.FitnessStore))
	mux.HandleFunc("GET /api/fitness/progress", fitness.HandleGetUserProgress(cfg.FitnessStore))
	mux.HandleFunc("GET /api/fitness/users", fitness.HandleListUsers(cfg.FitnessStore))
	mux.HandleFunc("POST /api/fitness/users", fitness.HandleCreateUser(cfg.FitnessStore))
	mux.HandleFunc("GET /api/fitness/exercises/{id}/history", fitness.HandleGetExerciseHistory(cfg.FitnessStore))
	mux.HandleFunc("POST /api/fitness/exercises/{id}/favorite", fitness.HandleToggleFavorite(cfg.FitnessStore))
	mux.HandleFunc("GET /api/fitness/exercises", fitness.HandleListExercises(cfg.FitnessStore))
	mux.HandleFunc("POST /api/fitness/exercises", fitness.HandleCreateExercise(cfg.FitnessStore))
	mux.HandleFunc("POST /api/fitness/workouts/{id}/exercises", fitness.HandleAddExerciseToWorkout(cfg.FitnessStore))
	mux.HandleFunc("PUT /api/fitness/workouts/{id}/complete", fitness.HandleCompleteWorkout(cfg.FitnessStore))
	mux.HandleFunc("PUT /api/fitness/workouts/{id}/meta", fitness.HandleUpdateWorkoutMeta(cfg.FitnessStore))
	mux.HandleFunc("GET /api/fitness/workouts/{id}", fitness.HandleGetWorkout(cfg.FitnessStore))
	mux.HandleFunc("DELETE /api/fitness/workouts/{id}", fitness.HandleDeleteWorkout(cfg.FitnessStore))
	mux.HandleFunc("GET /api/fitness/workouts", fitness.HandleListWorkouts(cfg.FitnessStore))
	mux.HandleFunc("POST /api/fitness/workouts", fitness.HandleCreateWorkout(cfg.FitnessStore))
	mux.HandleFunc("PUT /api/fitness/workout-exercises/{id}", fitness.HandleUpdateWorkoutExercise(cfg.FitnessStore))
	mux.HandleFunc("DELETE /api/fitness/workout-exercises/{id}", fitness.HandleRemoveWorkoutExercise(cfg.FitnessStore))
	mux.HandleFunc("POST /api/fitness/workout-exercises/{id}/sets", fitness.HandleAddSet(cfg.FitnessStore))
	mux.HandleFunc("PUT /api/fitness/sets/{id}", fitness.HandleUpdateSet(cfg.FitnessStore))
	mux.HandleFunc("DELETE /api/fitness/sets/{id}", fitness.HandleDeleteSet(cfg.FitnessStore))

	// Fantasy Leagues
	mux.HandleFunc("POST /api/fantasy/import", fantasy.HandleStartImport(cfg.FantasyClient))
	mux.HandleFunc("GET /api/fantasy/leagues/{id}/matchups", fantasy.HandleListMatchups(cfg.FantasyStore))
	mux.HandleFunc("GET /api/fantasy/leagues/{id}", fantasy.HandleGetLeague(cfg.FantasyStore))
	mux.HandleFunc("GET /api/fantasy/leagues", fantasy.HandleListLeagues(cfg.FantasyStore))
	mux.HandleFunc("GET /api/fantasy/teams/{id}", fantasy.HandleGetTeam(cfg.FantasyStore))

	// SvelteKit SPA — catch-all for non-API routes
	mux.Handle("/", frontend.Handler())
}
