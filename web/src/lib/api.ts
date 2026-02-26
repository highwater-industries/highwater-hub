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

export function startImport(seasons: number[] = [2024], strategy = 'merge'): Promise<{ job_id: string; status: string }> {
	return post('/jobs/import', { seasons, strategy });
}

export function getJobStatus(jobId: string): Promise<Record<string, unknown>> {
	return get(`/jobs/${jobId}`);
}
