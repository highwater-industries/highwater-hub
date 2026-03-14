package fitness

import "time"

// FitnessUser represents a simple user for workout tracking.
type FitnessUser struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

// Exercise represents an exercise in the library.
type Exercise struct {
	ID              int       `json:"id"`
	Name            string    `json:"name"`
	Category        string    `json:"category"` // strength, cardio, bodyweight
	MuscleGroup     *string   `json:"muscle_group,omitempty"`
	Equipment       *string   `json:"equipment,omitempty"`
	IsPreset        bool      `json:"is_preset"`
	CreatedByUserID *int      `json:"created_by_user_id,omitempty"`
	IsFavorite      bool      `json:"is_favorite"`
	CreatedAt       time.Time `json:"created_at"`
}

// WorkoutSummary is the list/dashboard view of a workout.
type WorkoutSummary struct {
	ID            int        `json:"id"`
	UserID        int        `json:"user_id"`
	StartedAt     time.Time  `json:"started_at"`
	CompletedAt   *time.Time `json:"completed_at,omitempty"`
	Notes         *string    `json:"notes,omitempty"`
	IsDeload      bool       `json:"is_deload"`
	ExerciseCount int        `json:"exercise_count"`
	SetCount      int        `json:"set_count"`
	ExerciseNames string     `json:"exercise_names"`
	CreatedAt     time.Time  `json:"created_at"`
}

// WorkoutDetail is the full detail view including exercises and sets.
type WorkoutDetail struct {
	ID          int                     `json:"id"`
	UserID      int                     `json:"user_id"`
	StartedAt   time.Time               `json:"started_at"`
	CompletedAt *time.Time              `json:"completed_at,omitempty"`
	Notes       *string                 `json:"notes,omitempty"`
	IsDeload    bool                    `json:"is_deload"`
	CreatedAt   time.Time               `json:"created_at"`
	Exercises   []WorkoutExerciseDetail `json:"exercises"`
}

// WorkoutExercise represents an exercise within a workout.
type WorkoutExercise struct {
	ID              int       `json:"id"`
	WorkoutID       int       `json:"workout_id"`
	ExerciseID      int       `json:"exercise_id"`
	OrderIndex      int       `json:"order_index"`
	Notes           *string   `json:"notes,omitempty"`
	Difficulty      *int      `json:"difficulty,omitempty"`
	ReadyToProgress bool      `json:"ready_to_progress"`
	CreatedAt       time.Time `json:"created_at"`
}

// WorkoutExerciseDetail includes exercise metadata and sets.
type WorkoutExerciseDetail struct {
	WorkoutExercise
	ExerciseName     string       `json:"exercise_name"`
	ExerciseCategory string       `json:"exercise_category"`
	Sets             []WorkoutSet `json:"sets"`
}

// WorkoutSet represents a single set within a workout exercise.
type WorkoutSet struct {
	ID                int       `json:"id"`
	WorkoutExerciseID int       `json:"workout_exercise_id"`
	SetNumber         int       `json:"set_number"`
	Reps              *int      `json:"reps,omitempty"`
	Weight            *float64  `json:"weight,omitempty"`
	DurationSeconds   *int      `json:"duration_seconds,omitempty"`
	DistanceMiles     *float64  `json:"distance_miles,omitempty"`
	TopSpeedMph       *float64  `json:"top_speed_mph,omitempty"`
	InclinePercent    *float64  `json:"incline_percent,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
}

// ExerciseHistoryEntry represents a past session for a specific exercise.
type ExerciseHistoryEntry struct {
	WorkoutID       int          `json:"workout_id"`
	Date            time.Time    `json:"date"`
	Difficulty      *int         `json:"difficulty,omitempty"`
	ReadyToProgress bool         `json:"ready_to_progress"`
	Notes           *string      `json:"notes,omitempty"`
	Sets            []WorkoutSet `json:"sets"`
}

// ExerciseProgressCard holds an exercise and its recent session history,
// used to render progress tracking cards.
type ExerciseProgressCard struct {
	ExerciseID       int                    `json:"exercise_id"`
	ExerciseName     string                 `json:"exercise_name"`
	ExerciseCategory string                 `json:"exercise_category"`
	MuscleGroup      *string                `json:"muscle_group,omitempty"`
	Equipment        *string                `json:"equipment,omitempty"`
	Sessions         []ExerciseHistoryEntry `json:"sessions"`
}
