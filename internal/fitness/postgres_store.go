package fitness

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// PostgresStore implements the fitness Store interface.
type PostgresStore struct {
	db *sql.DB
}

// NewPostgresStore creates a new fitness store backed by PostgreSQL.
func NewPostgresStore(db *sql.DB) *PostgresStore {
	return &PostgresStore{db: db}
}

// --------------------------------------------------------------------------
// Table initialisation
// --------------------------------------------------------------------------

// EnsureTables creates all fitness tables if they don't already exist.
func (s *PostgresStore) EnsureTables(ctx context.Context) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS fitness_users (
			id SERIAL PRIMARY KEY,
			name TEXT NOT NULL UNIQUE,
			created_at TIMESTAMPTZ DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS exercises (
			id SERIAL PRIMARY KEY,
			name TEXT NOT NULL UNIQUE,
			category TEXT NOT NULL CHECK (category IN ('strength', 'cardio', 'bodyweight')),
			muscle_group TEXT,
			equipment TEXT,
			is_preset BOOLEAN DEFAULT FALSE,
			created_by_user_id INT REFERENCES fitness_users(id),
			created_at TIMESTAMPTZ DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS user_favorite_exercises (
			user_id INT REFERENCES fitness_users(id) ON DELETE CASCADE,
			exercise_id INT REFERENCES exercises(id) ON DELETE CASCADE,
			PRIMARY KEY (user_id, exercise_id)
		)`,
		`CREATE TABLE IF NOT EXISTS workouts (
			id SERIAL PRIMARY KEY,
			user_id INT REFERENCES fitness_users(id) NOT NULL,
			started_at TIMESTAMPTZ DEFAULT NOW(),
			completed_at TIMESTAMPTZ,
			notes TEXT,
			is_deload BOOLEAN DEFAULT FALSE,
			created_at TIMESTAMPTZ DEFAULT NOW()
		)`,
		`ALTER TABLE workouts ADD COLUMN IF NOT EXISTS is_deload BOOLEAN DEFAULT FALSE`,
		`CREATE TABLE IF NOT EXISTS workout_exercises (
			id SERIAL PRIMARY KEY,
			workout_id INT REFERENCES workouts(id) ON DELETE CASCADE,
			exercise_id INT REFERENCES exercises(id),
			order_index INT NOT NULL,
			notes TEXT,
			difficulty INT CHECK (difficulty >= 1 AND difficulty <= 5),
			ready_to_progress BOOLEAN DEFAULT FALSE,
			created_at TIMESTAMPTZ DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS workout_sets (
			id SERIAL PRIMARY KEY,
			workout_exercise_id INT REFERENCES workout_exercises(id) ON DELETE CASCADE,
			set_number INT NOT NULL,
			reps INT,
			weight REAL,
			duration_seconds INT,
			distance_miles REAL,
			top_speed_mph REAL,
			incline_percent REAL,
			created_at TIMESTAMPTZ DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_workouts_user_id ON workouts(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_workout_exercises_workout_id ON workout_exercises(workout_id)`,
		`CREATE INDEX IF NOT EXISTS idx_workout_exercises_exercise_id ON workout_exercises(exercise_id)`,
		`CREATE INDEX IF NOT EXISTS idx_workout_sets_we_id ON workout_sets(workout_exercise_id)`,
		`CREATE TABLE IF NOT EXISTS bodyweight_log (
			id SERIAL PRIMARY KEY,
			user_id INT REFERENCES fitness_users(id) ON DELETE CASCADE NOT NULL,
			weight_lbs REAL NOT NULL,
			logged_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			notes TEXT,
			created_at TIMESTAMPTZ DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_bodyweight_log_user_date ON bodyweight_log(user_id, logged_at DESC)`,
	}
	for _, q := range stmts {
		if _, err := s.db.ExecContext(ctx, q); err != nil {
			return fmt.Errorf("fitness table init: %w", err)
		}
	}
	return nil
}

// --------------------------------------------------------------------------
// Users
// --------------------------------------------------------------------------

// ListUsers returns all fitness users ordered by name.
func (s *PostgresStore) ListUsers(ctx context.Context) ([]FitnessUser, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, name, created_at FROM fitness_users ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("list fitness users: %w", err)
	}
	defer rows.Close()

	var users []FitnessUser
	for rows.Next() {
		var u FitnessUser
		if err := rows.Scan(&u.ID, &u.Name, &u.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan fitness user: %w", err)
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

// CreateUser inserts a new fitness user and returns it.
func (s *PostgresStore) CreateUser(ctx context.Context, name string) (FitnessUser, error) {
	var u FitnessUser
	err := s.db.QueryRowContext(ctx,
		`INSERT INTO fitness_users (name) VALUES ($1)
		 RETURNING id, name, created_at`,
		name,
	).Scan(&u.ID, &u.Name, &u.CreatedAt)
	if err != nil {
		return FitnessUser{}, fmt.Errorf("create fitness user: %w", err)
	}
	return u, nil
}

// --------------------------------------------------------------------------
// Exercises
// --------------------------------------------------------------------------

// ListExercises returns a filtered, paginated list of exercises.
func (s *PostgresStore) ListExercises(ctx context.Context, filter ExerciseFilter, offset, limit int) ([]Exercise, int, error) {
	where, args := buildExerciseWhere(filter)

	// Count total matching
	var total int
	countSQL := "SELECT COUNT(*) FROM exercises e" + where
	if err := s.db.QueryRowContext(ctx, countSQL, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count exercises: %w", err)
	}

	// Optionally join user_favorite_exercises for the is_favorite flag
	favSelect := "FALSE"
	favJoin := ""
	if filter.UserID != nil {
		args = append(args, *filter.UserID)
		favSelect = "CASE WHEN uf.user_id IS NOT NULL THEN TRUE ELSE FALSE END"
		favJoin = fmt.Sprintf(
			" LEFT JOIN user_favorite_exercises uf ON uf.exercise_id = e.id AND uf.user_id = $%d",
			len(args),
		)
	}

	args = append(args, limit, offset)
	querySQL := fmt.Sprintf(
		`SELECT e.id, e.name, e.category, e.muscle_group, e.equipment,
		        e.is_preset, e.created_by_user_id, %s, e.created_at
		 FROM exercises e%s%s
		 ORDER BY e.name
		 LIMIT $%d OFFSET $%d`,
		favSelect, favJoin, where, len(args)-1, len(args),
	)

	rows, err := s.db.QueryContext(ctx, querySQL, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query exercises: %w", err)
	}
	defer rows.Close()

	var exercises []Exercise
	for rows.Next() {
		ex, err := scanExerciseRow(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan exercise: %w", err)
		}
		exercises = append(exercises, ex)
	}
	return exercises, total, rows.Err()
}

// CreateExercise inserts a user-defined exercise and returns it.
func (s *PostgresStore) CreateExercise(ctx context.Context, name, category string, muscleGroup, equipment *string, userID *int) (Exercise, error) {
	var mg, eq sql.NullString
	var uid sql.NullInt64
	if muscleGroup != nil {
		mg = sql.NullString{String: *muscleGroup, Valid: true}
	}
	if equipment != nil {
		eq = sql.NullString{String: *equipment, Valid: true}
	}
	if userID != nil {
		uid = sql.NullInt64{Int64: int64(*userID), Valid: true}
	}

	var ex Exercise
	var rMG, rEQ sql.NullString
	var rUID sql.NullInt64

	err := s.db.QueryRowContext(ctx,
		`INSERT INTO exercises (name, category, muscle_group, equipment, is_preset, created_by_user_id)
		 VALUES ($1, $2, $3, $4, FALSE, $5)
		 RETURNING id, name, category, muscle_group, equipment, is_preset, created_by_user_id, created_at`,
		name, category, mg, eq, uid,
	).Scan(&ex.ID, &ex.Name, &ex.Category, &rMG, &rEQ, &ex.IsPreset, &rUID, &ex.CreatedAt)
	if err != nil {
		return Exercise{}, fmt.Errorf("create exercise: %w", err)
	}

	if rMG.Valid {
		ex.MuscleGroup = &rMG.String
	}
	if rEQ.Valid {
		ex.Equipment = &rEQ.String
	}
	if rUID.Valid {
		v := int(rUID.Int64)
		ex.CreatedByUserID = &v
	}
	return ex, nil
}

// ToggleFavorite adds or removes an exercise from the user's favorites.
// Returns true if the exercise is now favorited, false if removed.
func (s *PostgresStore) ToggleFavorite(ctx context.Context, userID, exerciseID int) (bool, error) {
	result, err := s.db.ExecContext(ctx,
		`DELETE FROM user_favorite_exercises WHERE user_id = $1 AND exercise_id = $2`,
		userID, exerciseID,
	)
	if err != nil {
		return false, fmt.Errorf("toggle favorite (delete): %w", err)
	}
	n, _ := result.RowsAffected()
	if n > 0 {
		return false, nil // was favorited, now removed
	}

	_, err = s.db.ExecContext(ctx,
		`INSERT INTO user_favorite_exercises (user_id, exercise_id) VALUES ($1, $2)`,
		userID, exerciseID,
	)
	if err != nil {
		return false, fmt.Errorf("toggle favorite (insert): %w", err)
	}
	return true, nil
}

// GetExerciseHistory returns the last N completed sessions for a given
// exercise and user — including sets, difficulty, and the progression flag.
func (s *PostgresStore) GetExerciseHistory(ctx context.Context, exerciseID, userID, limit int) ([]ExerciseHistoryEntry, error) {
	rows, err := s.db.QueryContext(ctx, `
		WITH recent AS (
			SELECT DISTINCT w.id, w.started_at
			FROM workouts w
			JOIN workout_exercises we ON we.workout_id = w.id
			WHERE we.exercise_id = $1
			  AND w.user_id = $2
			  AND w.completed_at IS NOT NULL
			ORDER BY w.started_at DESC
			LIMIT $3
		)
		SELECT r.id, r.started_at,
		       we.difficulty, we.ready_to_progress, we.notes,
		       ws.id, ws.set_number, ws.reps, ws.weight,
		       ws.duration_seconds, ws.distance_miles,
		       ws.top_speed_mph, ws.incline_percent
		FROM recent r
		JOIN workout_exercises we ON we.workout_id = r.id AND we.exercise_id = $1
		LEFT JOIN workout_sets ws ON ws.workout_exercise_id = we.id
		ORDER BY r.started_at DESC, ws.set_number ASC
	`, exerciseID, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("get exercise history: %w", err)
	}
	defer rows.Close()

	entryMap := map[int]*ExerciseHistoryEntry{}
	var order []int

	for rows.Next() {
		var wID int
		var date time.Time
		var diff sql.NullInt64
		var rtp sql.NullBool
		var notes sql.NullString
		var sID, sNum, sReps, sDur sql.NullInt64
		var sWeight, sDist, sSpeed, sIncline sql.NullFloat64

		if err := rows.Scan(
			&wID, &date,
			&diff, &rtp, &notes,
			&sID, &sNum, &sReps, &sWeight,
			&sDur, &sDist, &sSpeed, &sIncline,
		); err != nil {
			return nil, fmt.Errorf("scan exercise history: %w", err)
		}

		entry, exists := entryMap[wID]
		if !exists {
			entry = &ExerciseHistoryEntry{
				WorkoutID: wID,
				Date:      date,
				Sets:      []WorkoutSet{},
			}
			if diff.Valid {
				d := int(diff.Int64)
				entry.Difficulty = &d
			}
			if rtp.Valid {
				entry.ReadyToProgress = rtp.Bool
			}
			if notes.Valid {
				entry.Notes = &notes.String
			}
			entryMap[wID] = entry
			order = append(order, wID)
		}

		if sID.Valid {
			ws := WorkoutSet{
				ID:        int(sID.Int64),
				SetNumber: int(sNum.Int64),
			}
			setIntPtr(&ws.Reps, sReps)
			setFloat64Ptr(&ws.Weight, sWeight)
			setIntPtr(&ws.DurationSeconds, sDur)
			setFloat64Ptr(&ws.DistanceMiles, sDist)
			setFloat64Ptr(&ws.TopSpeedMph, sSpeed)
			setFloat64Ptr(&ws.InclinePercent, sIncline)
			entry.Sets = append(entry.Sets, ws)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate exercise history: %w", err)
	}

	result := make([]ExerciseHistoryEntry, 0, len(order))
	for _, id := range order {
		result = append(result, *entryMap[id])
	}
	return result, nil
}

// GetUserProgress returns progress cards for all exercises a user has performed
// in completed workouts, each with the last N sessions of history.
func (s *PostgresStore) GetUserProgress(ctx context.Context, userID, sessionLimit int) ([]ExerciseProgressCard, error) {
	// 1. Find all exercises this user has performed, ordered by most recent.
	rows, err := s.db.QueryContext(ctx, `
		SELECT e.id, e.name, e.category, e.muscle_group, e.equipment
		FROM exercises e
		JOIN workout_exercises we ON we.exercise_id = e.id
		JOIN workouts w ON w.id = we.workout_id
		WHERE w.user_id = $1 AND w.completed_at IS NOT NULL
		GROUP BY e.id
		ORDER BY MAX(w.started_at) DESC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("list progress exercises: %w", err)
	}
	defer rows.Close()

	type exInfo struct {
		ID       int
		Name     string
		Category string
		Muscle   *string
		Equip    *string
	}
	var exercises []exInfo
	for rows.Next() {
		var e exInfo
		var mg, eq sql.NullString
		if err := rows.Scan(&e.ID, &e.Name, &e.Category, &mg, &eq); err != nil {
			return nil, fmt.Errorf("scan progress exercise: %w", err)
		}
		if mg.Valid {
			e.Muscle = &mg.String
		}
		if eq.Valid {
			e.Equip = &eq.String
		}
		exercises = append(exercises, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate progress exercises: %w", err)
	}

	// 2. For each exercise, fetch session history (reuse existing method).
	cards := make([]ExerciseProgressCard, 0, len(exercises))
	for _, e := range exercises {
		sessions, err := s.GetExerciseHistory(ctx, e.ID, userID, sessionLimit)
		if err != nil {
			return nil, fmt.Errorf("progress history for exercise %d: %w", e.ID, err)
		}
		cards = append(cards, ExerciseProgressCard{
			ExerciseID:       e.ID,
			ExerciseName:     e.Name,
			ExerciseCategory: e.Category,
			MuscleGroup:      e.Muscle,
			Equipment:        e.Equip,
			Sessions:         sessions,
		})
	}
	return cards, nil
}

// --------------------------------------------------------------------------
// Bodyweight
// --------------------------------------------------------------------------

// LogBodyweight inserts a new bodyweight entry for a user.
func (s *PostgresStore) LogBodyweight(ctx context.Context, userID int, weightLbs float64, loggedAt *time.Time, notes *string) (BodyweightEntry, error) {
	var la interface{}
	if loggedAt != nil {
		la = *loggedAt
	}
	var n interface{}
	if notes != nil {
		n = *notes
	}

	var bw BodyweightEntry
	var rNotes sql.NullString
	err := s.db.QueryRowContext(ctx,
		`INSERT INTO bodyweight_log (user_id, weight_lbs, logged_at, notes)
		 VALUES ($1, $2, COALESCE($3::timestamptz, NOW()), $4)
		 RETURNING id, user_id, weight_lbs, logged_at, notes, created_at`,
		userID, weightLbs, la, n,
	).Scan(&bw.ID, &bw.UserID, &bw.WeightLbs, &bw.LoggedAt, &rNotes, &bw.CreatedAt)
	if err != nil {
		return BodyweightEntry{}, fmt.Errorf("log bodyweight: %w", err)
	}
	if rNotes.Valid {
		bw.Notes = &rNotes.String
	}
	return bw, nil
}

// GetLatestBodyweight returns the most recent bodyweight entry for a user, or nil.
func (s *PostgresStore) GetLatestBodyweight(ctx context.Context, userID int) (*BodyweightEntry, error) {
	var bw BodyweightEntry
	var rNotes sql.NullString
	err := s.db.QueryRowContext(ctx,
		`SELECT id, user_id, weight_lbs, logged_at, notes, created_at
		 FROM bodyweight_log
		 WHERE user_id = $1
		 ORDER BY logged_at DESC
		 LIMIT 1`, userID,
	).Scan(&bw.ID, &bw.UserID, &bw.WeightLbs, &bw.LoggedAt, &rNotes, &bw.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get latest bodyweight: %w", err)
	}
	if rNotes.Valid {
		bw.Notes = &rNotes.String
	}
	return &bw, nil
}

// ListBodyweightHistory returns the last N bodyweight entries for a user.
func (s *PostgresStore) ListBodyweightHistory(ctx context.Context, userID int, limit int) ([]BodyweightEntry, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, user_id, weight_lbs, logged_at, notes, created_at
		 FROM bodyweight_log
		 WHERE user_id = $1
		 ORDER BY logged_at DESC
		 LIMIT $2`, userID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list bodyweight history: %w", err)
	}
	defer rows.Close()

	var entries []BodyweightEntry
	for rows.Next() {
		var bw BodyweightEntry
		var rNotes sql.NullString
		if err := rows.Scan(&bw.ID, &bw.UserID, &bw.WeightLbs, &bw.LoggedAt, &rNotes, &bw.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan bodyweight: %w", err)
		}
		if rNotes.Valid {
			bw.Notes = &rNotes.String
		}
		entries = append(entries, bw)
	}
	return entries, rows.Err()
}

// DeleteBodyweight removes a bodyweight entry by ID.
func (s *PostgresStore) DeleteBodyweight(ctx context.Context, id int) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM bodyweight_log WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete bodyweight: %w", err)
	}
	return nil
}

// --------------------------------------------------------------------------
// Helpers
// --------------------------------------------------------------------------

// buildExerciseWhere constructs a WHERE clause from an ExerciseFilter.
func buildExerciseWhere(f ExerciseFilter) (string, []any) {
	var conditions []string
	var args []any

	if f.Category != nil {
		args = append(args, *f.Category)
		conditions = append(conditions, fmt.Sprintf("e.category = $%d", len(args)))
	}
	if f.Search != nil {
		args = append(args, "%"+*f.Search+"%")
		conditions = append(conditions, fmt.Sprintf("e.name ILIKE $%d", len(args)))
	}

	if len(conditions) == 0 {
		return "", nil
	}
	return " WHERE " + strings.Join(conditions, " AND "), args
}

// scanExerciseRow scans a single row from *sql.Rows into an Exercise.
func scanExerciseRow(rows *sql.Rows) (Exercise, error) {
	var ex Exercise
	var mg, eq sql.NullString
	var uid sql.NullInt64

	err := rows.Scan(
		&ex.ID, &ex.Name, &ex.Category, &mg, &eq,
		&ex.IsPreset, &uid, &ex.IsFavorite, &ex.CreatedAt,
	)
	if err != nil {
		return Exercise{}, err
	}

	if mg.Valid {
		ex.MuscleGroup = &mg.String
	}
	if eq.Valid {
		ex.Equipment = &eq.String
	}
	if uid.Valid {
		v := int(uid.Int64)
		ex.CreatedByUserID = &v
	}
	return ex, nil
}

// setIntPtr converts a sql.NullInt64 to *int.
func setIntPtr(dst **int, src sql.NullInt64) {
	if src.Valid {
		v := int(src.Int64)
		*dst = &v
	}
}

// setFloat64Ptr converts a sql.NullFloat64 to *float64.
func setFloat64Ptr(dst **float64, src sql.NullFloat64) {
	if src.Valid {
		*dst = &src.Float64
	}
}
