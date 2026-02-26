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
	started_at: string;
	finished_at: string | null;
	params: Record<string, unknown>;
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

export function listJobs(offset = 0, limit = 20): Promise<ListResponse<Job>> {
	return get(`/jobs?offset=${offset}&limit=${limit}`);
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
