// Centralized API client for the Go backend.
// In dev mode, Vite proxies /api to localhost:3141.
// In production, the Go binary serves both the SPA and the API.

const BASE = '/api';

export interface ListResponse<T> {
	items: T[];
	total: number;
	offset: number;
	limit: number;
}

export interface Player {
	id: number;
	player_id: string;
	player_name: string;
	team: string;
	player_position: string;
	source: string;
	metadata: Record<string, unknown>;
	created_at: string;
	updated_at: string;
}

export interface Job {
	id: number;
	collector_type: string;
	status: string;
	records_fetched: number;
	records_inserted: number;
	records_updated: number;
	records_skipped: number;
	error_message?: string;
	started_at: string;
	finished_at: string | null;
	params: Record<string, unknown>;
	progress: number | null;
}

export interface PlayerFilter {
	team?: string;
	position?: string;
	source?: string;
	search?: string;
	sort?: string;
	order?: string;
	offset?: number;
	limit?: number;
}

async function get<T>(path: string): Promise<T> {
	const res = await fetch(`${BASE}${path}`);
	if (!res.ok) throw new Error(`API error: ${res.status}`);
	return res.json();
}

async function post<T>(path: string, body?: unknown): Promise<T> {
	const res = await fetch(`${BASE}${path}`, {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: body ? JSON.stringify(body) : undefined
	});
	if (!res.ok) throw new Error(`API error: ${res.status}`);
	return res.json();
}

async function put<T>(path: string, body?: unknown): Promise<T> {
	const res = await fetch(`${BASE}${path}`, {
		method: 'PUT',
		headers: { 'Content-Type': 'application/json' },
		body: body ? JSON.stringify(body) : undefined
	});
	if (!res.ok) throw new Error(`API error: ${res.status}`);
	return res.json();
}

async function del<T>(path: string): Promise<T> {
	const res = await fetch(`${BASE}${path}`, { method: 'DELETE' });
	if (!res.ok) throw new Error(`API error: ${res.status}`);
	return res.json();
}

// ── Players ──

export function listPlayers(filter: PlayerFilter = {}): Promise<ListResponse<Player>> {
	const params = new URLSearchParams();
	if (filter.team) params.set('team', filter.team);
	if (filter.position) params.set('position', filter.position);
	if (filter.source) params.set('source', filter.source);
	if (filter.search) params.set('search', filter.search);
	if (filter.sort) params.set('sort', filter.sort);
	if (filter.order) params.set('order', filter.order);
	if (filter.offset !== undefined) params.set('offset', String(filter.offset));
	if (filter.limit !== undefined) params.set('limit', String(filter.limit));
	const qs = params.toString();
	return get(`/nflstats/players${qs ? '?' + qs : ''}`);
}

export function getPlayer(id: number): Promise<{ data: Player }> {
	return get(`/nflstats/players/${id}`);
}

// ── Jobs ──

export function listJobs(offset = 0, limit = 50): Promise<ListResponse<Job>> {
	return get(`/jobs?offset=${offset}&limit=${limit}`);
}

export interface JobSummary {
	pending: number;
	running: number;
	completed: number;
	failed: number;
	total: number;
}

export function getJobSummary(): Promise<JobSummary> {
	return get('/jobs/summary');
}

export interface ImportOptions {
	collector_type?: string;
	seasons?: number[];
	strategy?: string;
	summary_level?: string;
	rank_type?: string;
}

export function startImport(opts: ImportOptions = {}): Promise<{ job_id: string; status: string }> {
	const body: Record<string, unknown> = {
		collector_type: opts.collector_type ?? 'nflreadpy',
		seasons: opts.seasons ?? [2024],
		strategy: opts.strategy ?? 'merge',
	};
	if (opts.summary_level) body.summary_level = opts.summary_level;
	if (opts.rank_type) body.rank_type = opts.rank_type;
	return post('/jobs/import', body);
}

export function getJobStatus(jobId: string): Promise<Record<string, unknown>> {
	return get(`/jobs/${jobId}`);
}

export interface BatchImportResult {
	collector_type: string;
	job_id?: string;
	status: string;
	error?: string;
}

export interface BatchImportResponse {
	results: BatchImportResult[];
	dispatched: number;
	failed: number;
}

export function batchImport(imports: ImportOptions[]): Promise<BatchImportResponse> {
	return post('/jobs/import/batch', { imports });
}

// Full import: dispatches rosters first for all seasons, then stats+schedules+rankings.
// Calls onPhase callback between phases so the UI can update.
export async function fullImport(
	seasons: number[],
	onPhase?: (phase: string, result: BatchImportResponse) => void
): Promise<{ phase1: BatchImportResponse; phase2: BatchImportResponse }> {
	// Phase 1: Rosters (creates/updates player records)
	const rosterImports: ImportOptions[] = seasons.map((s) => ({
		collector_type: 'nflreadpy',
		seasons: [s],
		strategy: 'merge',
	}));
	const phase1 = await batchImport(rosterImports);
	onPhase?.('ROSTERS', phase1);

	// Phase 2: Stats + Schedules + Rankings (reference player data)
	const dataImports: ImportOptions[] = seasons.flatMap((s) => [
		{ collector_type: 'nflreadpy_stats', seasons: [s], strategy: 'merge', summary_level: 'week' },
		{ collector_type: 'nflreadpy_schedules', seasons: [s], strategy: 'merge' },
		{ collector_type: 'nflreadpy_ff_rankings', seasons: [s], strategy: 'merge', rank_type: 'draft' },
	]);
	const phase2 = await batchImport(dataImports);
	onPhase?.('STATS + SCHEDULES + RANKINGS', phase2);

	return { phase1, phase2 };
}

// ── Player Stats ──

export interface PlayerStat {
	id: number;
	player_id?: string;
	player_name: string;
	player_display_name?: string;
	position?: string;
	team?: string;
	season: number;
	week: number;
	stat_type?: string;
	completions?: number;
	attempts?: number;
	passing_yards?: number;
	passing_tds?: number;
	interceptions?: number;
	carries?: number;
	rushing_yards?: number;
	rushing_tds?: number;
	receptions?: number;
	targets?: number;
	receiving_yards?: number;
	receiving_tds?: number;
	fantasy_points?: number;
	fantasy_points_ppr?: number;
	source?: string;
}

export interface StatFilter {
	player_id?: string;
	team?: string;
	position?: string;
	season?: number;
	week?: number;
	stat_type?: string;
	source?: string;
	search?: string;
	sort?: string;
	order?: string;
	offset?: number;
	limit?: number;
}

export function listStats(filter: StatFilter = {}): Promise<ListResponse<PlayerStat>> {
	const params = new URLSearchParams();
	if (filter.player_id) params.set('player_id', filter.player_id);
	if (filter.team) params.set('team', filter.team);
	if (filter.position) params.set('position', filter.position);
	if (filter.season !== undefined) params.set('season', String(filter.season));
	if (filter.week !== undefined) params.set('week', String(filter.week));
	if (filter.stat_type) params.set('stat_type', filter.stat_type);
	if (filter.source) params.set('source', filter.source);
	if (filter.search) params.set('search', filter.search);
	if (filter.sort) params.set('sort', filter.sort);
	if (filter.order) params.set('order', filter.order);
	if (filter.offset !== undefined) params.set('offset', String(filter.offset));
	if (filter.limit !== undefined) params.set('limit', String(filter.limit));
	const qs = params.toString();
	return get(`/nflstats/stats${qs ? '?' + qs : ''}`);
}

export function getLeaders(
	stat: string,
	season: number,
	week?: number,
	position?: string,
	limit = 25
): Promise<{ stat: string; season: number; week: number; position: string; items: PlayerStat[] }> {
	const params = new URLSearchParams({ stat, season: String(season), limit: String(limit) });
	if (week) params.set('week', String(week));
	if (position) params.set('position', position);
	return get(`/nflstats/leaders?${params}`);
}

// ── Games ──

export interface GameData {
	id: number;
	game_id?: string;
	season?: number;
	game_type?: string;
	week?: number;
	gameday?: string;
	away_team?: string;
	home_team?: string;
	away_score?: number;
	home_score?: number;
	overtime?: boolean;
	stadium?: string;
}

export interface GameFilter {
	season?: number;
	week?: number;
	team?: string;
	sort?: string;
	order?: string;
	offset?: number;
	limit?: number;
}

export function listGames(filter: GameFilter = {}): Promise<ListResponse<GameData>> {
	const params = new URLSearchParams();
	if (filter.season !== undefined) params.set('season', String(filter.season));
	if (filter.week !== undefined) params.set('week', String(filter.week));
	if (filter.team) params.set('team', filter.team);
	if (filter.sort) params.set('sort', filter.sort);
	if (filter.order) params.set('order', filter.order);
	if (filter.offset !== undefined) params.set('offset', String(filter.offset));
	if (filter.limit !== undefined) params.set('limit', String(filter.limit));
	const qs = params.toString();
	return get(`/nflstats/games${qs ? '?' + qs : ''}`);
}

// ── Fantasy Rankings ──

export interface FantasyRank {
	id: number;
	player_id?: string;
	player_name: string;
	pos?: string;
	team?: string;
	rank?: number;
	ecr?: number;
	sd?: number;
	best?: number;
	worst?: number;
	avg?: number;
	rank_type?: string;
	season?: number;
	week?: number;
	source?: string;
}

export interface RankingFilter {
	rank_type?: string;
	pos?: string;
	team?: string;
	season?: number;
	week?: number;
	source?: string;
	search?: string;
	sort?: string;
	order?: string;
	offset?: number;
	limit?: number;
}

export function listRankings(filter: RankingFilter = {}): Promise<ListResponse<FantasyRank>> {
	const params = new URLSearchParams();
	if (filter.rank_type) params.set('rank_type', filter.rank_type);
	if (filter.pos) params.set('pos', filter.pos);
	if (filter.team) params.set('team', filter.team);
	if (filter.search) params.set('search', filter.search);
	if (filter.season !== undefined) params.set('season', String(filter.season));
	if (filter.week !== undefined) params.set('week', String(filter.week));
	if (filter.source) params.set('source', filter.source);
	if (filter.sort) params.set('sort', filter.sort);
	if (filter.order) params.set('order', filter.order);
	if (filter.offset !== undefined) params.set('offset', String(filter.offset));
	if (filter.limit !== undefined) params.set('limit', String(filter.limit));
	const qs = params.toString();
	return get(`/nflstats/rankings${qs ? '?' + qs : ''}`);
}

// ── Fitness: Types ──

export interface FitnessUser {
	id: number;
	name: string;
	created_at: string;
}

export interface Exercise {
	id: number;
	name: string;
	category: string;
	muscle_group?: string;
	equipment?: string;
	is_preset: boolean;
	created_by_user_id?: number;
	is_favorite?: boolean;
}

export interface WorkoutSummary {
	id: number;
	user_id: number;
	started_at: string;
	completed_at?: string;
	notes?: string;
	is_deload: boolean;
	exercise_count: number;
	set_count: number;
	exercise_names?: string;
}

export interface WorkoutSet {
	id: number;
	workout_exercise_id: number;
	set_number: number;
	reps?: number;
	weight?: number;
	duration_seconds?: number;
	distance_miles?: number;
	top_speed_mph?: number;
	incline_percent?: number;
	created_at: string;
}

export interface WorkoutExerciseDetail {
	id: number;
	workout_id: number;
	exercise_id: number;
	order_index: number;
	notes?: string;
	difficulty?: number;
	ready_to_progress: boolean;
	exercise_name: string;
	exercise_category: string;
	sets: WorkoutSet[];
}

export interface WorkoutDetail {
	id: number;
	user_id: number;
	started_at: string;
	completed_at?: string;
	notes?: string;
	is_deload: boolean;
	exercises: WorkoutExerciseDetail[];
}

export interface ExerciseHistoryEntry {
	workout_id: number;
	date: string;
	difficulty?: number;
	ready_to_progress: boolean;
	notes?: string;
	sets: WorkoutSet[];
}

// ── Fitness: Users ──

export function listFitnessUsers(): Promise<FitnessUser[]> {
	return get('/fitness/users');
}

export function createFitnessUser(name: string): Promise<FitnessUser> {
	return post('/fitness/users', { name });
}

// ── Fitness: Exercises ──

export interface ExerciseFilter {
	category?: string;
	search?: string;
	user_id?: number;
	offset?: number;
	limit?: number;
}

export function listExercises(filter: ExerciseFilter = {}): Promise<ListResponse<Exercise>> {
	const params = new URLSearchParams();
	if (filter.category) params.set('category', filter.category);
	if (filter.search) params.set('search', filter.search);
	if (filter.user_id !== undefined) params.set('user_id', String(filter.user_id));
	if (filter.offset !== undefined) params.set('offset', String(filter.offset));
	if (filter.limit !== undefined) params.set('limit', String(filter.limit));
	const qs = params.toString();
	return get(`/fitness/exercises${qs ? '?' + qs : ''}`);
}

export function createExercise(
	name: string,
	category: string,
	muscleGroup?: string,
	equipment?: string,
	userId?: number
): Promise<Exercise> {
	return post('/fitness/exercises', {
		name,
		category,
		muscle_group: muscleGroup,
		equipment: equipment,
		user_id: userId
	});
}

export function toggleFavorite(exerciseId: number, userId: number): Promise<{ is_favorite: boolean }> {
	return post(`/fitness/exercises/${exerciseId}/favorite`, { user_id: userId });
}

export function getExerciseHistory(
	exerciseId: number,
	userId: number,
	limit = 6
): Promise<ExerciseHistoryEntry[]> {
	return get(`/fitness/exercises/${exerciseId}/history?user_id=${userId}&limit=${limit}`);
}

// ── Fitness: Workouts ──

export function listWorkouts(userId: number, offset = 0, limit = 20): Promise<ListResponse<WorkoutSummary>> {
	return get(`/fitness/workouts?user_id=${userId}&offset=${offset}&limit=${limit}`);
}

export function createWorkout(userId: number, startedAt?: string, isDeload = false): Promise<WorkoutSummary> {
	return post('/fitness/workouts', { user_id: userId, started_at: startedAt, is_deload: isDeload });
}

export function updateWorkoutMeta(id: number, updates: { is_deload?: boolean; started_at?: string }): Promise<{ status: string }> {
	return put(`/fitness/workouts/${id}/meta`, updates);
}

export function getWorkout(id: number): Promise<WorkoutDetail> {
	return get(`/fitness/workouts/${id}`);
}

export function completeWorkout(id: number, notes?: string): Promise<{ status: string }> {
	return put(`/fitness/workouts/${id}/complete`, { notes });
}

export function deleteWorkout(id: number): Promise<{ status: string }> {
	return del(`/fitness/workouts/${id}`);
}

// ── Fitness: Workout Exercises ──

export function addExerciseToWorkout(
	workoutId: number,
	exerciseId: number
): Promise<{ id: number; workout_id: number; exercise_id: number; order_index: number }> {
	return post(`/fitness/workouts/${workoutId}/exercises`, { exercise_id: exerciseId });
}

export function updateWorkoutExercise(
	id: number,
	updates: { notes?: string; difficulty?: number; ready_to_progress?: boolean }
): Promise<{ status: string }> {
	return put(`/fitness/workout-exercises/${id}`, updates);
}

export function removeWorkoutExercise(id: number): Promise<{ status: string }> {
	return del(`/fitness/workout-exercises/${id}`);
}

// ── Fitness: Sets ──

export function addSet(
	workoutExerciseId: number,
	data: { reps?: number; weight?: number; duration_seconds?: number; distance_miles?: number; top_speed_mph?: number; incline_percent?: number }
): Promise<WorkoutSet> {
	return post(`/fitness/workout-exercises/${workoutExerciseId}/sets`, data);
}

export function updateSet(
	id: number,
	data: { reps?: number; weight?: number; duration_seconds?: number; distance_miles?: number; top_speed_mph?: number; incline_percent?: number }
): Promise<{ status: string }> {
	return put(`/fitness/sets/${id}`, data);
}

export function deleteSet(id: number): Promise<{ status: string }> {
	return del(`/fitness/sets/${id}`);
}
