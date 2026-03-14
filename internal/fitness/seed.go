package fitness

import (
	"context"
	"fmt"
)

// seedExercise defines a single preset exercise for seeding.
type seedExercise struct {
	Name        string
	Category    string
	MuscleGroup string
	Equipment   string
}

// presetExercises is the built-in exercise library.
var presetExercises = []seedExercise{
	// ── Strength — Chest ──
	{"Bench Press", "strength", "Chest", "Barbell"},
	{"Incline Bench Press", "strength", "Chest", "Barbell"},
	{"Incline Narrow Grip Bench Press", "strength", "Chest", "Barbell"},
	{"Dumbbell Bench Press", "strength", "Chest", "Dumbbell"},
	{"Dumbbell Fly", "strength", "Chest", "Dumbbell"},
	{"Cable Fly", "strength", "Chest", "Cable"},

	// ── Strength — Shoulders ──
	{"Overhead Press", "strength", "Shoulders", "Barbell"},
	{"Dumbbell Shoulder Press", "strength", "Shoulders", "Dumbbell"},
	{"Lateral Raise", "strength", "Shoulders", "Dumbbell"},
	{"Front Raise", "strength", "Shoulders", "Dumbbell"},
	{"Face Pull", "strength", "Shoulders", "Cable"},

	// ── Strength — Back ──
	{"Barbell Row", "strength", "Back", "Barbell"},
	{"Pendlay Row", "strength", "Back", "Barbell"},
	{"Kroc Row", "strength", "Back", "Dumbbell"},
	{"Dumbbell Row", "strength", "Back", "Dumbbell"},
	{"Lat Pulldown", "strength", "Back", "Cable"},
	{"Seated Cable Row", "strength", "Back", "Cable"},
	{"T-Bar Row", "strength", "Back", "Barbell"},
	{"Deadlift", "strength", "Back", "Barbell"},

	// ── Strength — Legs ──
	{"Barbell Squat", "strength", "Legs", "Barbell"},
	{"Box Squat", "strength", "Legs", "Barbell"},
	{"Front Squat", "strength", "Legs", "Barbell"},
	{"Leg Press", "strength", "Legs", "Machine"},
	{"Romanian Deadlift", "strength", "Legs", "Barbell"},
	{"Sumo Deadlift", "strength", "Legs", "Barbell"},
	{"Leg Curl", "strength", "Legs", "Machine"},
	{"Leg Extension", "strength", "Legs", "Machine"},
	{"Calf Raise", "strength", "Legs", "Machine"},
	{"Glute-Ham Extension", "strength", "Legs", "Machine"},
	{"Hip Thrust", "strength", "Legs", "Barbell"},
	{"Bulgarian Split Squat", "strength", "Legs", "Dumbbell"},
	{"Lunge", "strength", "Legs", "Dumbbell"},

	// ── Strength — Arms ──
	{"Barbell Curl", "strength", "Arms", "Barbell"},
	{"Dumbbell Curl", "strength", "Arms", "Dumbbell"},
	{"Hammer Curl", "strength", "Arms", "Dumbbell"},
	{"Tricep Pushdown", "strength", "Arms", "Cable"},
	{"Skull Crusher", "strength", "Arms", "Barbell"},
	{"Overhead Tricep Extension", "strength", "Arms", "Dumbbell"},

	// ── Strength — Traps ──
	{"Shrug", "strength", "Traps", "Dumbbell"},

	// ── Bodyweight ──
	{"Push-up", "bodyweight", "Chest", "None"},
	{"Wide Grip Pull-up", "bodyweight", "Back", "None"},
	{"Narrow Grip Pull-up", "bodyweight", "Back", "None"},
	{"Chin-up", "bodyweight", "Back", "None"},
	{"Dip", "bodyweight", "Chest", "None"},
	{"Bodyweight Squat", "bodyweight", "Legs", "None"},
	{"Plank", "bodyweight", "Core", "None"},
	{"Sit-up", "bodyweight", "Core", "None"},
	{"Russian Twist", "bodyweight", "Core", "None"},
	{"Mountain Climber", "bodyweight", "Core", "None"},
	{"Burpee", "bodyweight", "Full Body", "None"},

	// ── Cardio ──
	{"Running", "cardio", "Cardio", "None"},
	{"Treadmill", "cardio", "Cardio", "Treadmill"},
	{"Cycling", "cardio", "Cardio", "Bike"},
	{"Stationary Bike", "cardio", "Cardio", "Bike"},
	{"Rowing Machine", "cardio", "Cardio", "Rowing Machine"},
	{"Elliptical", "cardio", "Cardio", "Elliptical"},
	{"Stair Climber", "cardio", "Cardio", "Machine"},
	{"Jump Rope", "cardio", "Cardio", "Jump Rope"},
	{"Swimming", "cardio", "Cardio", "None"},
	{"Walking", "cardio", "Cardio", "None"},
}

// SeedExercises inserts the preset exercise library. Existing entries
// (matched by name) are silently skipped via ON CONFLICT DO NOTHING.
func (s *PostgresStore) SeedExercises(ctx context.Context) error {
	for _, ex := range presetExercises {
		_, err := s.db.ExecContext(ctx,
			`INSERT INTO exercises (name, category, muscle_group, equipment, is_preset)
			 VALUES ($1, $2, $3, $4, TRUE)
			 ON CONFLICT (name) DO NOTHING`,
			ex.Name, ex.Category, ex.MuscleGroup, ex.Equipment,
		)
		if err != nil {
			return fmt.Errorf("seed exercise %q: %w", ex.Name, err)
		}
	}
	return nil
}
