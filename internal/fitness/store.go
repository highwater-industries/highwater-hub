package fitness

import (
	"context"
	"time"
)

// ExerciseFilter controls exercise listing/searching.
type ExerciseFilter struct {
	Category *string
	Search   *string
	UserID   *int // for marking favorites
}

// Store is the unified interface for all fitness data operations.
type Store interface {
	EnsureTables(ctx context.Context) error
	SeedExercises(ctx context.Context) error

	// Users
	ListUsers(ctx context.Context) ([]FitnessUser, error)
	CreateUser(ctx context.Context, name string) (FitnessUser, error)

	// Exercises
	ListExercises(ctx context.Context, filter ExerciseFilter, offset, limit int) ([]Exercise, int, error)
	CreateExercise(ctx context.Context, name, category string, muscleGroup, equipment *string, userID *int) (Exercise, error)
	ToggleFavorite(ctx context.Context, userID, exerciseID int) (bool, error)
	GetExerciseHistory(ctx context.Context, exerciseID, userID, limit int) ([]ExerciseHistoryEntry, error)

	// Workouts
	ListWorkouts(ctx context.Context, userID int, offset, limit int) ([]WorkoutSummary, int, error)
	CreateWorkout(ctx context.Context, userID int, startedAt *time.Time, isDeload bool) (WorkoutSummary, error)
	GetWorkout(ctx context.Context, id int) (WorkoutDetail, error)
	UpdateWorkoutMeta(ctx context.Context, id int, isDeload *bool, startedAt *time.Time) error
	CompleteWorkout(ctx context.Context, id int, notes *string) error
	DeleteWorkout(ctx context.Context, id int) error

	// Workout exercises
	AddExercise(ctx context.Context, workoutID, exerciseID int) (WorkoutExercise, error)
	UpdateExercise(ctx context.Context, id int, notes *string, difficulty *int, readyToProgress *bool) error
	RemoveExercise(ctx context.Context, id int) error

	// Sets
	AddSet(ctx context.Context, workoutExerciseID int, set WorkoutSet) (WorkoutSet, error)
	UpdateSet(ctx context.Context, set WorkoutSet) error
	DeleteSet(ctx context.Context, id int) error
}
