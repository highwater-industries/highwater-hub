package fitness

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"myproject/internal/httputil"
)

// ── Users ──

// HandleListUsers returns all fitness users.
//
//	GET /api/fitness/users
func HandleListUsers(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		users, err := store.ListUsers(r.Context())
		if err != nil {
			httputil.Encode(w, http.StatusInternalServerError, httputil.ErrorResponse{
				Error: "failed to list users",
			})
			return
		}
		httputil.Encode(w, http.StatusOK, users)
	}
}

// HandleCreateUser creates a new fitness user.
//
//	POST /api/fitness/users  {"name": "Matt"}
func HandleCreateUser(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Name string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" {
			httputil.Encode(w, http.StatusBadRequest, httputil.ErrorResponse{
				Error: "name is required",
			})
			return
		}

		user, err := store.CreateUser(r.Context(), body.Name)
		if err != nil {
			httputil.Encode(w, http.StatusConflict, httputil.ErrorResponse{
				Error: "user already exists or creation failed",
			})
			return
		}
		httputil.Encode(w, http.StatusCreated, user)
	}
}

// ── Exercises ──

// HandleListExercises returns a filtered, paginated list of exercises.
//
//	GET /api/fitness/exercises?category=strength&search=bench&user_id=1&offset=0&limit=20
func HandleListExercises(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p := httputil.ParsePagination(
			r.URL.Query().Get("offset"),
			r.URL.Query().Get("limit"),
		)

		filter := ExerciseFilter{}
		if v := r.URL.Query().Get("category"); v != "" {
			filter.Category = &v
		}
		if v := r.URL.Query().Get("search"); v != "" {
			filter.Search = &v
		}
		if v := r.URL.Query().Get("user_id"); v != "" {
			if uid, err := strconv.Atoi(v); err == nil {
				filter.UserID = &uid
			}
		}

		exercises, total, err := store.ListExercises(r.Context(), filter, p.Offset, p.Limit)
		if err != nil {
			httputil.Encode(w, http.StatusInternalServerError, httputil.ErrorResponse{
				Error: "failed to list exercises",
			})
			return
		}

		httputil.Encode(w, http.StatusOK, httputil.ListResponse[Exercise]{
			Items:  exercises,
			Total:  total,
			Offset: p.Offset,
			Limit:  p.Limit,
		})
	}
}

// HandleCreateExercise creates a new custom exercise.
//
//	POST /api/fitness/exercises  {"name": "...", "category": "strength", "muscle_group": "...", "equipment": "...", "user_id": 1}
func HandleCreateExercise(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Name        string  `json:"name"`
			Category    string  `json:"category"`
			MuscleGroup *string `json:"muscle_group"`
			Equipment   *string `json:"equipment"`
			UserID      *int    `json:"user_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" || body.Category == "" {
			httputil.Encode(w, http.StatusBadRequest, httputil.ErrorResponse{
				Error: "name and category are required",
			})
			return
		}

		ex, err := store.CreateExercise(r.Context(), body.Name, body.Category, body.MuscleGroup, body.Equipment, body.UserID)
		if err != nil {
			httputil.Encode(w, http.StatusConflict, httputil.ErrorResponse{
				Error: "exercise already exists or creation failed",
			})
			return
		}
		httputil.Encode(w, http.StatusCreated, ex)
	}
}

// HandleToggleFavorite toggles an exercise as favorite for a user.
//
//	POST /api/fitness/exercises/{id}/favorite  {"user_id": 1}
func HandleToggleFavorite(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		exerciseID, err := strconv.Atoi(r.PathValue("id"))
		if err != nil {
			httputil.Encode(w, http.StatusBadRequest, httputil.ErrorResponse{
				Error: "invalid exercise id",
			})
			return
		}

		var body struct {
			UserID int `json:"user_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.UserID == 0 {
			httputil.Encode(w, http.StatusBadRequest, httputil.ErrorResponse{
				Error: "user_id is required",
			})
			return
		}

		isFav, err := store.ToggleFavorite(r.Context(), body.UserID, exerciseID)
		if err != nil {
			httputil.Encode(w, http.StatusInternalServerError, httputil.ErrorResponse{
				Error: "failed to toggle favorite",
			})
			return
		}

		httputil.Encode(w, http.StatusOK, map[string]bool{"is_favorite": isFav})
	}
}

// HandleGetExerciseHistory returns recent session history for an exercise.
//
//	GET /api/fitness/exercises/{id}/history?user_id=1&limit=8
func HandleGetExerciseHistory(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		exerciseID, err := strconv.Atoi(r.PathValue("id"))
		if err != nil {
			httputil.Encode(w, http.StatusBadRequest, httputil.ErrorResponse{
				Error: "invalid exercise id",
			})
			return
		}

		userID, err := strconv.Atoi(r.URL.Query().Get("user_id"))
		if err != nil {
			httputil.Encode(w, http.StatusBadRequest, httputil.ErrorResponse{
				Error: "user_id is required",
			})
			return
		}

		limit := 8
		if v, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && v > 0 && v <= 20 {
			limit = v
		}

		history, err := store.GetExerciseHistory(r.Context(), exerciseID, userID, limit)
		if err != nil {
			httputil.Encode(w, http.StatusInternalServerError, httputil.ErrorResponse{
				Error: "failed to get exercise history",
			})
			return
		}
		httputil.Encode(w, http.StatusOK, history)
	}
}

// HandleGetUserProgress returns exercise progress cards for a user.
//
//	GET /api/fitness/progress?user_id=1
func HandleGetUserProgress(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := strconv.Atoi(r.URL.Query().Get("user_id"))
		if err != nil {
			httputil.Encode(w, http.StatusBadRequest, httputil.ErrorResponse{
				Error: "user_id is required",
			})
			return
		}

		limit := 6
		if v, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && v > 0 && v <= 20 {
			limit = v
		}

		cards, err := store.GetUserProgress(r.Context(), userID, limit)
		if err != nil {
			httputil.Encode(w, http.StatusInternalServerError, httputil.ErrorResponse{
				Error: "failed to get user progress",
			})
			return
		}
		httputil.Encode(w, http.StatusOK, cards)
	}
}

// ── Workouts ──

// HandleListWorkouts returns a paginated list of workouts for a user.
//
//	GET /api/fitness/workouts?user_id=1&offset=0&limit=20
func HandleListWorkouts(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := strconv.Atoi(r.URL.Query().Get("user_id"))
		if err != nil {
			httputil.Encode(w, http.StatusBadRequest, httputil.ErrorResponse{
				Error: "user_id is required",
			})
			return
		}

		p := httputil.ParsePagination(
			r.URL.Query().Get("offset"),
			r.URL.Query().Get("limit"),
		)

		workouts, total, err := store.ListWorkouts(r.Context(), userID, p.Offset, p.Limit)
		if err != nil {
			httputil.Encode(w, http.StatusInternalServerError, httputil.ErrorResponse{
				Error: "failed to list workouts",
			})
			return
		}

		httputil.Encode(w, http.StatusOK, httputil.ListResponse[WorkoutSummary]{
			Items:  workouts,
			Total:  total,
			Offset: p.Offset,
			Limit:  p.Limit,
		})
	}
}

// HandleCreateWorkout starts a new workout.
//
//	POST /api/fitness/workouts  {"user_id": 1, "started_at": "2026-01-15T10:00:00Z", "is_deload": false}
func HandleCreateWorkout(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			UserID    int     `json:"user_id"`
			StartedAt *string `json:"started_at"`
			IsDeload  bool    `json:"is_deload"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.UserID == 0 {
			httputil.Encode(w, http.StatusBadRequest, httputil.ErrorResponse{
				Error: "user_id is required",
			})
			return
		}

		var startedAt *time.Time
		if body.StartedAt != nil && *body.StartedAt != "" {
			t, err := time.Parse(time.RFC3339, *body.StartedAt)
			if err != nil {
				httputil.Encode(w, http.StatusBadRequest, httputil.ErrorResponse{
					Error: "invalid started_at format, use RFC3339",
				})
				return
			}
			startedAt = &t
		}

		workout, err := store.CreateWorkout(r.Context(), body.UserID, startedAt, body.IsDeload)
		if err != nil {
			httputil.Encode(w, http.StatusInternalServerError, httputil.ErrorResponse{
				Error: "failed to create workout",
			})
			return
		}
		httputil.Encode(w, http.StatusCreated, workout)
	}
}

// HandleGetWorkout returns full workout detail.
//
//	GET /api/fitness/workouts/{id}
func HandleGetWorkout(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(r.PathValue("id"))
		if err != nil {
			httputil.Encode(w, http.StatusBadRequest, httputil.ErrorResponse{
				Error: "invalid workout id",
			})
			return
		}

		workout, err := store.GetWorkout(r.Context(), id)
		if err != nil {
			httputil.Encode(w, http.StatusNotFound, httputil.ErrorResponse{
				Error: "workout not found",
			})
			return
		}
		httputil.Encode(w, http.StatusOK, workout)
	}
}

// HandleCompleteWorkout marks a workout as finished.
//
//	PUT /api/fitness/workouts/{id}/complete  {"notes": "great session"}
func HandleCompleteWorkout(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(r.PathValue("id"))
		if err != nil {
			httputil.Encode(w, http.StatusBadRequest, httputil.ErrorResponse{
				Error: "invalid workout id",
			})
			return
		}

		var body struct {
			Notes *string `json:"notes"`
		}
		json.NewDecoder(r.Body).Decode(&body)

		if err := store.CompleteWorkout(r.Context(), id, body.Notes); err != nil {
			httputil.Encode(w, http.StatusBadRequest, httputil.ErrorResponse{
				Error: err.Error(),
			})
			return
		}
		httputil.Encode(w, http.StatusOK, map[string]string{"status": "completed"})
	}
}

// HandleDeleteWorkout deletes a workout and all its exercises/sets.
//
//	DELETE /api/fitness/workouts/{id}
func HandleDeleteWorkout(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(r.PathValue("id"))
		if err != nil {
			httputil.Encode(w, http.StatusBadRequest, httputil.ErrorResponse{
				Error: "invalid workout id",
			})
			return
		}

		if err := store.DeleteWorkout(r.Context(), id); err != nil {
			httputil.Encode(w, http.StatusInternalServerError, httputil.ErrorResponse{
				Error: "failed to delete workout",
			})
			return
		}
		httputil.Encode(w, http.StatusOK, map[string]string{"status": "deleted"})
	}
}

// HandleUpdateWorkoutMeta updates deload flag and/or started_at on a workout.
//
//	PUT /api/fitness/workouts/{id}/meta  {"is_deload": true, "started_at": "2026-01-15T10:00:00Z"}
func HandleUpdateWorkoutMeta(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(r.PathValue("id"))
		if err != nil {
			httputil.Encode(w, http.StatusBadRequest, httputil.ErrorResponse{
				Error: "invalid workout id",
			})
			return
		}

		var body struct {
			IsDeload  *bool   `json:"is_deload"`
			StartedAt *string `json:"started_at"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			httputil.Encode(w, http.StatusBadRequest, httputil.ErrorResponse{
				Error: "invalid request body",
			})
			return
		}

		var startedAt *time.Time
		if body.StartedAt != nil {
			t, err := time.Parse(time.RFC3339, *body.StartedAt)
			if err != nil {
				httputil.Encode(w, http.StatusBadRequest, httputil.ErrorResponse{
					Error: "invalid started_at format, use RFC3339",
				})
				return
			}
			startedAt = &t
		}

		if err := store.UpdateWorkoutMeta(r.Context(), id, body.IsDeload, startedAt); err != nil {
			httputil.Encode(w, http.StatusInternalServerError, httputil.ErrorResponse{
				Error: "failed to update workout",
			})
			return
		}
		httputil.Encode(w, http.StatusOK, map[string]string{"status": "updated"})
	}
}

// ── Workout Exercises ──

// HandleAddExerciseToWorkout adds an exercise to a workout.
//
//	POST /api/fitness/workouts/{id}/exercises  {"exercise_id": 5}
func HandleAddExerciseToWorkout(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		workoutID, err := strconv.Atoi(r.PathValue("id"))
		if err != nil {
			httputil.Encode(w, http.StatusBadRequest, httputil.ErrorResponse{
				Error: "invalid workout id",
			})
			return
		}

		var body struct {
			ExerciseID int `json:"exercise_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.ExerciseID == 0 {
			httputil.Encode(w, http.StatusBadRequest, httputil.ErrorResponse{
				Error: "exercise_id is required",
			})
			return
		}

		we, err := store.AddExercise(r.Context(), workoutID, body.ExerciseID)
		if err != nil {
			httputil.Encode(w, http.StatusInternalServerError, httputil.ErrorResponse{
				Error: "failed to add exercise",
			})
			return
		}
		httputil.Encode(w, http.StatusCreated, we)
	}
}

// HandleUpdateWorkoutExercise updates notes/difficulty/ready_to_progress on a workout exercise.
//
//	PUT /api/fitness/workout-exercises/{id}  {"notes": "...", "difficulty": 3, "ready_to_progress": true}
func HandleUpdateWorkoutExercise(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(r.PathValue("id"))
		if err != nil {
			httputil.Encode(w, http.StatusBadRequest, httputil.ErrorResponse{
				Error: "invalid workout exercise id",
			})
			return
		}

		var body struct {
			Notes           *string `json:"notes"`
			Difficulty      *int    `json:"difficulty"`
			ReadyToProgress *bool   `json:"ready_to_progress"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			httputil.Encode(w, http.StatusBadRequest, httputil.ErrorResponse{
				Error: "invalid request body",
			})
			return
		}

		if err := store.UpdateExercise(r.Context(), id, body.Notes, body.Difficulty, body.ReadyToProgress); err != nil {
			httputil.Encode(w, http.StatusInternalServerError, httputil.ErrorResponse{
				Error: "failed to update exercise",
			})
			return
		}
		httputil.Encode(w, http.StatusOK, map[string]string{"status": "updated"})
	}
}

// HandleRemoveWorkoutExercise removes an exercise from a workout.
//
//	DELETE /api/fitness/workout-exercises/{id}
func HandleRemoveWorkoutExercise(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(r.PathValue("id"))
		if err != nil {
			httputil.Encode(w, http.StatusBadRequest, httputil.ErrorResponse{
				Error: "invalid workout exercise id",
			})
			return
		}

		if err := store.RemoveExercise(r.Context(), id); err != nil {
			httputil.Encode(w, http.StatusInternalServerError, httputil.ErrorResponse{
				Error: "failed to remove exercise",
			})
			return
		}
		httputil.Encode(w, http.StatusOK, map[string]string{"status": "removed"})
	}
}

// ── Sets ──

// HandleAddSet adds a set to a workout exercise.
//
//	POST /api/fitness/workout-exercises/{id}/sets  {"reps": 8, "weight": 185}
func HandleAddSet(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		weID, err := strconv.Atoi(r.PathValue("id"))
		if err != nil {
			httputil.Encode(w, http.StatusBadRequest, httputil.ErrorResponse{
				Error: "invalid workout exercise id",
			})
			return
		}

		var set WorkoutSet
		if err := json.NewDecoder(r.Body).Decode(&set); err != nil {
			httputil.Encode(w, http.StatusBadRequest, httputil.ErrorResponse{
				Error: "invalid request body",
			})
			return
		}

		created, err := store.AddSet(r.Context(), weID, set)
		if err != nil {
			httputil.Encode(w, http.StatusInternalServerError, httputil.ErrorResponse{
				Error: "failed to add set",
			})
			return
		}
		httputil.Encode(w, http.StatusCreated, created)
	}
}

// HandleUpdateSet performs a partial update on an existing set.
//
//	PUT /api/fitness/sets/{id}  {"reps": 10}  — only updates reps, leaves weight etc. unchanged
func HandleUpdateSet(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(r.PathValue("id"))
		if err != nil {
			httputil.Encode(w, http.StatusBadRequest, httputil.ErrorResponse{
				Error: "invalid set id",
			})
			return
		}

		var fields map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&fields); err != nil {
			httputil.Encode(w, http.StatusBadRequest, httputil.ErrorResponse{
				Error: "invalid request body",
			})
			return
		}

		if err := store.UpdateSet(r.Context(), id, fields); err != nil {
			httputil.Encode(w, http.StatusInternalServerError, httputil.ErrorResponse{
				Error: "failed to update set",
			})
			return
		}
		httputil.Encode(w, http.StatusOK, map[string]string{"status": "updated"})
	}
}

// HandleDeleteSet deletes a set.
//
//	DELETE /api/fitness/sets/{id}
func HandleDeleteSet(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(r.PathValue("id"))
		if err != nil {
			httputil.Encode(w, http.StatusBadRequest, httputil.ErrorResponse{
				Error: "invalid set id",
			})
			return
		}

		if err := store.DeleteSet(r.Context(), id); err != nil {
			httputil.Encode(w, http.StatusInternalServerError, httputil.ErrorResponse{
				Error: "failed to delete set",
			})
			return
		}
		httputil.Encode(w, http.StatusOK, map[string]string{"status": "deleted"})
	}
}
