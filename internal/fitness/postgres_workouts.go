package fitness

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// --------------------------------------------------------------------------
// Workouts
// --------------------------------------------------------------------------

// ListWorkouts returns a paginated list of workouts for a user,
// including summary counts of exercises and sets.
func (s *PostgresStore) ListWorkouts(ctx context.Context, userID int, offset, limit int) ([]WorkoutSummary, int, error) {
	// Count
	var total int
	if err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM workouts WHERE user_id = $1`, userID,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count workouts: %w", err)
	}

	// Query with aggregated exercise/set counts
	rows, err := s.db.QueryContext(ctx, `
		SELECT w.id, w.user_id, w.started_at, w.completed_at, w.notes, w.is_deload, w.created_at,
		       COALESCE(COUNT(DISTINCT we.id), 0),
		       COALESCE(COUNT(ws.id), 0),
		       COALESCE(string_agg(DISTINCT e.name, ', ' ORDER BY e.name), ''),
		       COALESCE((
		           SELECT json_agg(row_to_json(sub) ORDER BY sub.name)::text
		           FROM (
		               SELECT e2.name,
		                      COUNT(ws2.id) AS sets,
		                      COALESCE(SUM(ws2.reps), 0) AS total_reps,
		                      COALESCE(MAX(ws2.weight), 0) AS max_weight,
		                      COALESCE(json_agg(ws2.reps ORDER BY ws2.set_number) FILTER (WHERE ws2.reps IS NOT NULL), '[]') AS reps_list
		               FROM workout_exercises we2
		               JOIN exercises e2 ON e2.id = we2.exercise_id
		               LEFT JOIN workout_sets ws2 ON ws2.workout_exercise_id = we2.id
		               WHERE we2.workout_id = w.id
		               GROUP BY e2.name
		           ) sub
		       ), '[]')
		FROM workouts w
		LEFT JOIN workout_exercises we ON we.workout_id = w.id
		LEFT JOIN workout_sets ws ON ws.workout_exercise_id = we.id
		LEFT JOIN exercises e ON e.id = we.exercise_id
		WHERE w.user_id = $1
		GROUP BY w.id
		ORDER BY w.started_at DESC
		OFFSET $2 LIMIT $3
	`, userID, offset, limit)
	if err != nil {
		return nil, 0, fmt.Errorf("query workouts: %w", err)
	}
	defer rows.Close()

	workouts := make([]WorkoutSummary, 0)
	for rows.Next() {
		ws, err := scanWorkoutSummaryRow(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan workout summary: %w", err)
		}
		workouts = append(workouts, ws)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate workouts: %w", err)
	}

	return workouts, total, nil
}

// CreateWorkout starts a new workout for a user, optionally at a specific date.
func (s *PostgresStore) CreateWorkout(ctx context.Context, userID int, startedAt *time.Time, isDeload bool) (WorkoutSummary, error) {
	var w WorkoutSummary
	var completedAt sql.NullTime
	var notes sql.NullString

	var saValue interface{}
	if startedAt != nil {
		saValue = *startedAt
	} else {
		saValue = nil // will use DEFAULT (NOW())
	}

	err := s.db.QueryRowContext(ctx,
		`INSERT INTO workouts (user_id, started_at, is_deload)
		 VALUES ($1, COALESCE($2::timestamptz, NOW()), $3)
		 RETURNING id, user_id, started_at, completed_at, notes, is_deload, created_at`,
		userID, saValue, isDeload,
	).Scan(&w.ID, &w.UserID, &w.StartedAt, &completedAt, &notes, &w.IsDeload, &w.CreatedAt)
	if err != nil {
		return WorkoutSummary{}, fmt.Errorf("create workout: %w", err)
	}

	return w, nil
}

// GetWorkout returns full workout detail including exercises and sets.
func (s *PostgresStore) GetWorkout(ctx context.Context, id int) (WorkoutDetail, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT
			w.id, w.user_id, w.started_at, w.completed_at, w.notes, w.is_deload, w.created_at,
			we.id, we.exercise_id, we.order_index, we.notes, we.difficulty,
			we.ready_to_progress, we.created_at,
			e.name, e.category,
			ws.id, ws.set_number, ws.reps, ws.weight,
			ws.duration_seconds, ws.distance_miles,
			ws.top_speed_mph, ws.incline_percent, ws.created_at
		FROM workouts w
		LEFT JOIN workout_exercises we ON we.workout_id = w.id
		LEFT JOIN exercises e ON e.id = we.exercise_id
		LEFT JOIN workout_sets ws ON ws.workout_exercise_id = we.id
		WHERE w.id = $1
		ORDER BY we.order_index, ws.set_number
	`, id)
	if err != nil {
		return WorkoutDetail{}, fmt.Errorf("get workout: %w", err)
	}
	defer rows.Close()

	var wd WorkoutDetail
	exerciseMap := map[int]*WorkoutExerciseDetail{}
	var exerciseOrder []int
	found := false

	for rows.Next() {
		// Workout-level columns
		var wID, wUserID int
		var wStartedAt, wCreatedAt time.Time
		var wCompletedAt sql.NullTime
		var wNotes sql.NullString
		var wIsDeload bool

		// Exercise-level (nullable — LEFT JOIN)
		var weID, weExerciseID, weOrderIndex sql.NullInt64
		var weNotes sql.NullString
		var weDifficulty sql.NullInt64
		var weReadyToProgress sql.NullBool
		var weCreatedAt sql.NullTime
		var eName, eCategory sql.NullString

		// Set-level (nullable)
		var wsID, wsSetNumber, wsReps, wsDur sql.NullInt64
		var wsWeight, wsDist, wsSpeed, wsIncline sql.NullFloat64
		var wsCreatedAt sql.NullTime

		if err := rows.Scan(
			&wID, &wUserID, &wStartedAt, &wCompletedAt, &wNotes, &wIsDeload, &wCreatedAt,
			&weID, &weExerciseID, &weOrderIndex, &weNotes, &weDifficulty,
			&weReadyToProgress, &weCreatedAt,
			&eName, &eCategory,
			&wsID, &wsSetNumber, &wsReps, &wsWeight,
			&wsDur, &wsDist, &wsSpeed, &wsIncline, &wsCreatedAt,
		); err != nil {
			return WorkoutDetail{}, fmt.Errorf("scan workout detail: %w", err)
		}

		// First row sets workout-level fields
		if !found {
			wd.ID = wID
			wd.UserID = wUserID
			wd.StartedAt = wStartedAt
			wd.CreatedAt = wCreatedAt
			wd.IsDeload = wIsDeload
			if wCompletedAt.Valid {
				wd.CompletedAt = &wCompletedAt.Time
			}
			if wNotes.Valid {
				wd.Notes = &wNotes.String
			}
			found = true
		}

		// Collect exercise
		if weID.Valid {
			exID := int(weID.Int64)
			if _, exists := exerciseMap[exID]; !exists {
				wed := &WorkoutExerciseDetail{
					WorkoutExercise: WorkoutExercise{
						ID:         exID,
						WorkoutID:  wID,
						ExerciseID: int(weExerciseID.Int64),
						OrderIndex: int(weOrderIndex.Int64),
					},
					Sets: []WorkoutSet{},
				}
				if weNotes.Valid {
					wed.Notes = &weNotes.String
				}
				if weDifficulty.Valid {
					d := int(weDifficulty.Int64)
					wed.Difficulty = &d
				}
				if weReadyToProgress.Valid {
					wed.ReadyToProgress = weReadyToProgress.Bool
				}
				if weCreatedAt.Valid {
					wed.CreatedAt = weCreatedAt.Time
				}
				if eName.Valid {
					wed.ExerciseName = eName.String
				}
				if eCategory.Valid {
					wed.ExerciseCategory = eCategory.String
				}
				exerciseMap[exID] = wed
				exerciseOrder = append(exerciseOrder, exID)
			}

			// Collect set
			if wsID.Valid {
				ws := WorkoutSet{
					ID:                int(wsID.Int64),
					WorkoutExerciseID: exID,
					SetNumber:         int(wsSetNumber.Int64),
				}
				if wsCreatedAt.Valid {
					ws.CreatedAt = wsCreatedAt.Time
				}
				setIntPtr(&ws.Reps, wsReps)
				setFloat64Ptr(&ws.Weight, wsWeight)
				setIntPtr(&ws.DurationSeconds, wsDur)
				setFloat64Ptr(&ws.DistanceMiles, wsDist)
				setFloat64Ptr(&ws.TopSpeedMph, wsSpeed)
				setFloat64Ptr(&ws.InclinePercent, wsIncline)
				exerciseMap[exID].Sets = append(exerciseMap[exID].Sets, ws)
			}
		}
	}
	if err := rows.Err(); err != nil {
		return WorkoutDetail{}, fmt.Errorf("iterate workout detail: %w", err)
	}

	if !found {
		return WorkoutDetail{}, sql.ErrNoRows
	}

	wd.Exercises = make([]WorkoutExerciseDetail, 0, len(exerciseOrder))
	for _, exID := range exerciseOrder {
		wd.Exercises = append(wd.Exercises, *exerciseMap[exID])
	}
	return wd, nil
}

// CompleteWorkout marks a workout as completed and optionally sets notes.
func (s *PostgresStore) CompleteWorkout(ctx context.Context, id int, notes *string) error {
	var n sql.NullString
	if notes != nil {
		n = sql.NullString{String: *notes, Valid: true}
	}

	result, err := s.db.ExecContext(ctx,
		`UPDATE workouts SET completed_at = NOW(), notes = $2
		 WHERE id = $1 AND completed_at IS NULL`,
		id, n,
	)
	if err != nil {
		return fmt.Errorf("complete workout: %w", err)
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("workout %d not found or already completed", id)
	}
	return nil
}

// DeleteWorkout removes a workout and all its exercises/sets (via CASCADE).
func (s *PostgresStore) DeleteWorkout(ctx context.Context, id int) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM workouts WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete workout: %w", err)
	}
	return nil
}

// UpdateWorkoutMeta updates is_deload and/or started_at on a workout.
func (s *PostgresStore) UpdateWorkoutMeta(ctx context.Context, id int, isDeload *bool, startedAt *time.Time) error {
	var sets []string
	args := []any{id}

	if isDeload != nil {
		args = append(args, *isDeload)
		sets = append(sets, fmt.Sprintf("is_deload = $%d", len(args)))
	}
	if startedAt != nil {
		args = append(args, *startedAt)
		sets = append(sets, fmt.Sprintf("started_at = $%d", len(args)))
	}

	if len(sets) == 0 {
		return nil
	}

	query := fmt.Sprintf("UPDATE workouts SET %s WHERE id = $1", joinStrings(sets, ", "))
	_, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update workout meta: %w", err)
	}
	return nil
}

// --------------------------------------------------------------------------
// Workout Exercises
// --------------------------------------------------------------------------

// AddExercise adds an exercise to an in-progress workout.
func (s *PostgresStore) AddExercise(ctx context.Context, workoutID, exerciseID int) (WorkoutExercise, error) {
	// Auto-calculate next order_index
	var maxOrder sql.NullInt64
	_ = s.db.QueryRowContext(ctx,
		`SELECT MAX(order_index) FROM workout_exercises WHERE workout_id = $1`,
		workoutID,
	).Scan(&maxOrder)

	nextOrder := 0
	if maxOrder.Valid {
		nextOrder = int(maxOrder.Int64) + 1
	}

	var we WorkoutExercise
	var notes sql.NullString
	var diff sql.NullInt64
	err := s.db.QueryRowContext(ctx,
		`INSERT INTO workout_exercises (workout_id, exercise_id, order_index)
		 VALUES ($1, $2, $3)
		 RETURNING id, workout_id, exercise_id, order_index,
		           notes, difficulty, ready_to_progress, created_at`,
		workoutID, exerciseID, nextOrder,
	).Scan(&we.ID, &we.WorkoutID, &we.ExerciseID, &we.OrderIndex,
		&notes, &diff, &we.ReadyToProgress, &we.CreatedAt)
	if err != nil {
		return WorkoutExercise{}, fmt.Errorf("add exercise to workout: %w", err)
	}
	if notes.Valid {
		we.Notes = &notes.String
	}
	if diff.Valid {
		d := int(diff.Int64)
		we.Difficulty = &d
	}
	return we, nil
}

// UpdateExercise updates the notes, difficulty, or progression flag on a
// workout exercise.
func (s *PostgresStore) UpdateExercise(ctx context.Context, id int, notes *string, difficulty *int, readyToProgress *bool) error {
	var sets []string
	args := []any{id}

	if notes != nil {
		args = append(args, *notes)
		sets = append(sets, fmt.Sprintf("notes = $%d", len(args)))
	}
	if difficulty != nil {
		args = append(args, *difficulty)
		sets = append(sets, fmt.Sprintf("difficulty = $%d", len(args)))
	}
	if readyToProgress != nil {
		args = append(args, *readyToProgress)
		sets = append(sets, fmt.Sprintf("ready_to_progress = $%d", len(args)))
	}

	if len(sets) == 0 {
		return nil
	}

	query := fmt.Sprintf("UPDATE workout_exercises SET %s WHERE id = $1",
		joinStrings(sets, ", "))
	_, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update workout exercise: %w", err)
	}
	return nil
}

// RemoveExercise deletes an exercise (and its sets via CASCADE) from a workout.
func (s *PostgresStore) RemoveExercise(ctx context.Context, id int) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM workout_exercises WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("remove workout exercise: %w", err)
	}
	return nil
}

// --------------------------------------------------------------------------
// Sets
// --------------------------------------------------------------------------

// AddSet adds a new set to a workout exercise.
func (s *PostgresStore) AddSet(ctx context.Context, workoutExerciseID int, set WorkoutSet) (WorkoutSet, error) {
	// Auto-calculate next set_number
	var maxNum sql.NullInt64
	_ = s.db.QueryRowContext(ctx,
		`SELECT MAX(set_number) FROM workout_sets WHERE workout_exercise_id = $1`,
		workoutExerciseID,
	).Scan(&maxNum)

	nextNum := 1
	if maxNum.Valid {
		nextNum = int(maxNum.Int64) + 1
	}

	reps := toNullInt64(set.Reps)
	weight := toNullFloat64(set.Weight)
	dur := toNullInt64(set.DurationSeconds)
	dist := toNullFloat64(set.DistanceMiles)
	speed := toNullFloat64(set.TopSpeedMph)
	incline := toNullFloat64(set.InclinePercent)

	var ws WorkoutSet
	var rReps, rDur sql.NullInt64
	var rWeight, rDist, rSpeed, rIncline sql.NullFloat64

	err := s.db.QueryRowContext(ctx,
		`INSERT INTO workout_sets
		     (workout_exercise_id, set_number, reps, weight,
		      duration_seconds, distance_miles, top_speed_mph, incline_percent)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 RETURNING id, workout_exercise_id, set_number,
		           reps, weight, duration_seconds, distance_miles,
		           top_speed_mph, incline_percent, created_at`,
		workoutExerciseID, nextNum, reps, weight, dur, dist, speed, incline,
	).Scan(
		&ws.ID, &ws.WorkoutExerciseID, &ws.SetNumber,
		&rReps, &rWeight, &rDur, &rDist,
		&rSpeed, &rIncline, &ws.CreatedAt,
	)
	if err != nil {
		return WorkoutSet{}, fmt.Errorf("add set: %w", err)
	}

	setIntPtr(&ws.Reps, rReps)
	setFloat64Ptr(&ws.Weight, rWeight)
	setIntPtr(&ws.DurationSeconds, rDur)
	setFloat64Ptr(&ws.DistanceMiles, rDist)
	setFloat64Ptr(&ws.TopSpeedMph, rSpeed)
	setFloat64Ptr(&ws.InclinePercent, rIncline)
	return ws, nil
}

// UpdateSet performs a partial update — only fields present in the map are changed.
func (s *PostgresStore) UpdateSet(ctx context.Context, id int, fields map[string]interface{}) error {
	allowed := map[string]string{
		"reps": "reps", "weight": "weight",
		"duration_seconds": "duration_seconds", "distance_miles": "distance_miles",
		"top_speed_mph": "top_speed_mph", "incline_percent": "incline_percent",
	}

	var setClauses []string
	var args []interface{}
	argN := 1

	for jsonKey, col := range allowed {
		if val, ok := fields[jsonKey]; ok {
			setClauses = append(setClauses, fmt.Sprintf("%s = $%d", col, argN))
			args = append(args, val) // nil → NULL, number → value
			argN++
		}
	}

	if len(setClauses) == 0 {
		return nil // nothing to update
	}

	args = append(args, id)
	query := fmt.Sprintf("UPDATE workout_sets SET %s WHERE id = $%d",
		strings.Join(setClauses, ", "), argN)

	_, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update set: %w", err)
	}
	return nil
}

// DeleteSet removes a set from a workout exercise.
func (s *PostgresStore) DeleteSet(ctx context.Context, id int) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM workout_sets WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete set: %w", err)
	}
	return nil
}

// --------------------------------------------------------------------------
// Helpers
// --------------------------------------------------------------------------

// scanWorkoutSummaryRow scans a single row from the ListWorkouts query.
func scanWorkoutSummaryRow(rows *sql.Rows) (WorkoutSummary, error) {
	var ws WorkoutSummary
	var completedAt sql.NullTime
	var notes sql.NullString

	err := rows.Scan(
		&ws.ID, &ws.UserID, &ws.StartedAt, &completedAt, &notes, &ws.IsDeload, &ws.CreatedAt,
		&ws.ExerciseCount, &ws.SetCount, &ws.ExerciseNames, &ws.ExerciseDetails,
	)
	if err != nil {
		return WorkoutSummary{}, err
	}

	if completedAt.Valid {
		ws.CompletedAt = &completedAt.Time
	}
	if notes.Valid {
		ws.Notes = &notes.String
	}
	return ws, nil
}

// toNullInt64 converts an *int to sql.NullInt64.
func toNullInt64(v *int) sql.NullInt64 {
	if v != nil {
		return sql.NullInt64{Int64: int64(*v), Valid: true}
	}
	return sql.NullInt64{}
}

// toNullFloat64 converts a *float64 to sql.NullFloat64.
func toNullFloat64(v *float64) sql.NullFloat64 {
	if v != nil {
		return sql.NullFloat64{Float64: *v, Valid: true}
	}
	return sql.NullFloat64{}
}

// joinStrings joins a slice of strings with a separator.
func joinStrings(ss []string, sep string) string {
	result := ""
	for i, s := range ss {
		if i > 0 {
			result += sep
		}
		result += s
	}
	return result
}
