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

// ── Player Summary ──

export interface SeasonTotals {
	season: number;
	season_type: string;
	games_played: number;
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
}

export interface PlayerSummary {
	player: Player;
	career_totals: SeasonTotals;
	seasons: SeasonTotals[];
	recent_games: PlayerStat[];
	rankings: FantasyRank[];
}

export function getPlayerSummary(id: number): Promise<PlayerSummary> {
	return get(`/nflstats/players/${id}/summary`);
}

// ── Jobs ──

export interface JobFilter {
	collector_type?: string;
	status?: string;
	season?: number;
}

export function listJobs(offset = 0, limit = 50, filter: JobFilter = {}): Promise<ListResponse<Job>> {
	const params = new URLSearchParams({ offset: String(offset), limit: String(limit) });
	if (filter.collector_type) params.set('collector_type', filter.collector_type);
	if (filter.status) params.set('status', filter.status);
	if (filter.season) params.set('season', String(filter.season));
	return get(`/jobs?${params}`);
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

export function cleanupStuckJobs(): Promise<{ cleaned: number }> {
	return post('/jobs/cleanup');
}

export function abortJob(id: number): Promise<{ aborted: number; celery_task: string; revoke_error: string }> {
	return post(`/jobs/${id}/abort`);
}

export function abortAllJobs(): Promise<{ aborted: number; celery_revoked: number }> {
	return post('/jobs/abort-all');
}

// ── Data Inventory & Audit ──

export interface InventoryRow {
	source: string;
	table: string;
	season?: number;
	stat_type?: string;
	season_type?: string;
	rank_type?: string;
	rows: number;
	distinct_players: number;
	min_week?: number;
	max_week?: number;
	week_count?: number;
	last_updated: string;
}

export interface InventoryTotals {
	players: number;
	stats: number;
	games: number;
	rankings: number;
}

export interface InventoryResponse {
	stats: InventoryRow[];
	players: InventoryRow[];
	games: InventoryRow[];
	rankings: InventoryRow[];
	totals: InventoryTotals;
}

export interface InventoryFilter {
	source?: string;
	season?: number;
	stat_type?: string;
	season_type?: string;
	rank_type?: string;
}

export function getInventory(filter: InventoryFilter = {}): Promise<InventoryResponse> {
	const params = new URLSearchParams();
	if (filter.source) params.set('source', filter.source);
	if (filter.season) params.set('season', String(filter.season));
	if (filter.stat_type) params.set('stat_type', filter.stat_type);
	if (filter.season_type) params.set('season_type', filter.season_type);
	if (filter.rank_type) params.set('rank_type', filter.rank_type);
	const qs = params.toString();
	return get(`/data/inventory${qs ? '?' + qs : ''}`);
}

export interface AuditDuplicate {
	season: number;
	week: number;
	stat_type: string;
	source: string;
	duplicates: number;
}

export interface AuditCompleteness {
	season: number;
	expected_weeks: number;
	actual_weeks: number;
	missing_weeks: number;
}

export interface AuditPlayerCoverage {
	season: number;
	rostered_players: number;
	players_with_stats: number;
	missing_stats: number;
}

export interface AuditRankingCoverage {
	total_rankings: number;
	resolved_players: number;
	unresolved_players: number;
	resolution_pct: number;
}

export interface AuditResult {
	duplicates: AuditDuplicate[];
	completeness: AuditCompleteness[];
	player_coverage: AuditPlayerCoverage[];
	ranking_coverage?: AuditRankingCoverage;
}

export function runAudit(table?: string, season?: number): Promise<AuditResult> {
	const params = new URLSearchParams();
	if (table) params.set('table', table);
	if (season !== undefined) params.set('season', String(season));
	const qs = params.toString();
	return get(`/data/audit${qs ? '?' + qs : ''}`);
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
	player_db_id?: number;
	player_id?: string;
	player_name: string;
	player_display_name?: string;
	position?: string;
	team?: string;
	season: number;
	week: number;
	stat_type?: string;
	season_type?: string;
	opponent_team?: string;
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
	season_type?: string;
	source?: string;
	search?: string;
	group_by?: string;
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
	if (filter.season_type) params.set('season_type', filter.season_type);
	if (filter.source) params.set('source', filter.source);
	if (filter.search) params.set('search', filter.search);
	if (filter.group_by) params.set('group_by', filter.group_by);
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
	player_db_id?: number;
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
	exercise_details?: string;
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

export interface ExerciseProgressCard {
	exercise_id: number;
	exercise_name: string;
	exercise_category: string;
	muscle_group?: string;
	equipment?: string;
	sessions: ExerciseHistoryEntry[];
}

export interface BodyweightEntry {
	id: number;
	user_id: number;
	weight_lbs: number;
	logged_at: string;
	notes?: string;
	created_at: string;
}

// ── Fitness: Users ──

export function listFitnessUsers(): Promise<FitnessUser[]> {
	return get('/fitness/users');
}

export function createFitnessUser(name: string): Promise<FitnessUser> {
	return post('/fitness/users', { name });
}

// ── Fitness: Bodyweight ──

export function logBodyweight(
	userId: number,
	weightLbs: number,
	loggedAt?: string,
	notes?: string
): Promise<BodyweightEntry> {
	return post('/fitness/bodyweight', {
		user_id: userId,
		weight_lbs: weightLbs,
		logged_at: loggedAt,
		notes
	});
}

export function getLatestBodyweight(userId: number): Promise<BodyweightEntry | null> {
	return get(`/fitness/bodyweight/latest?user_id=${userId}`).then((r) => {
		if (r && r.entry === null) return null;
		return r as BodyweightEntry;
	});
}

export function listBodyweightHistory(userId: number, limit = 30): Promise<BodyweightEntry[]> {
	return get(`/fitness/bodyweight?user_id=${userId}&limit=${limit}`);
}

export function deleteBodyweight(id: number): Promise<{ status: string }> {
	return del(`/fitness/bodyweight/${id}`);
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

export function getUserProgress(userId: number, limit = 6): Promise<ExerciseProgressCard[]> {
	return get(`/fitness/progress?user_id=${userId}&limit=${limit}`);
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

// ── Fantasy Leagues ──

export interface FantasyLeague {
	id: number;
	external_league_id: string;
	league_name: string;
	platform: string;
	season: number;
	num_teams?: number;
	scoring_type?: string;
	settings?: Record<string, unknown>;
	created_at: string;
	updated_at: string;
}

export interface FantasyTeam {
	id: number;
	league_id: number;
	external_team_id?: string;
	team_name: string;
	owner_name?: string;
	wins: number;
	losses: number;
	ties: number;
	points_for: number;
	points_against: number;
	standing_rank?: number;
	playoff_seed?: number;
	logo_url?: string;
	streak_type?: string;
	streak_value: number;
	waiver_priority: number;
	number_of_moves: number;
	number_of_trades: number;
	clinched_playoffs: boolean;
	draft_grade?: string;
	created_at: string;
	updated_at: string;
}

export interface FantasyRosterEntry {
	id: number;
	team_id: number;
	player_id?: string;
	player_name: string;
	player_position: string;
	nfl_team?: string;
	roster_position?: string;
	external_player_id?: string;
	matched: boolean;
	created_at: string;
}

export interface LeagueDetail {
	league: FantasyLeague;
	teams: FantasyTeam[];
}

export interface TeamDetail {
	team: FantasyTeam;
	roster: FantasyRosterEntry[];
}

export interface FantasyMatchup {
	id: number;
	league_id: number;
	week: number;
	matchup_id: number;
	team_name: string;
	external_team_id?: string;
	points: number;
	result?: string;
	is_playoff: boolean;
	created_at: string;
}

export interface FantasyLeagueFilter {
	platform?: string;
	season?: number;
	offset?: number;
	limit?: number;
}

export interface FantasyImportRequest {
	platform: string;
	league_id: string;
	season: number;
	swid?: string;
	espn_s2?: string;
}

export interface FantasyImportAccepted {
	job_id: string;
	status: string;
	platform: string;
	league_id: string;
	season: number;
}

export function listFantasyLeagues(filter: FantasyLeagueFilter = {}): Promise<ListResponse<FantasyLeague>> {
	const params = new URLSearchParams();
	if (filter.platform) params.set('platform', filter.platform);
	if (filter.season !== undefined) params.set('season', String(filter.season));
	if (filter.offset !== undefined) params.set('offset', String(filter.offset));
	if (filter.limit !== undefined) params.set('limit', String(filter.limit));
	const qs = params.toString();
	return get(`/fantasy/leagues${qs ? '?' + qs : ''}`);
}

export function getFantasyLeague(id: number): Promise<LeagueDetail> {
	return get(`/fantasy/leagues/${id}`);
}

export function getFantasyTeam(id: number): Promise<TeamDetail> {
	return get(`/fantasy/teams/${id}`);
}

export function getFantasyMatchups(leagueId: number): Promise<FantasyMatchup[]> {
	return get(`/fantasy/leagues/${leagueId}/matchups`);
}

export function startFantasyImport(req: FantasyImportRequest): Promise<FantasyImportAccepted> {
	return post('/fantasy/import', req);
}

