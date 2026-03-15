<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import PageHeader from '$lib/components/PageHeader.svelte';
	import StatCard from '$lib/components/StatCard.svelte';
	import {
		getInventory, runAudit, listJobs, listPlayers, startImport, batchImport, fullImport,
		getJobSummary, cleanupStuckJobs, abortJob, abortAllJobs,
		startFantasyImport,
		type InventoryResponse, type InventoryRow, type AuditResult,
		type Job, type JobSummary, type InventoryFilter, type JobFilter,
		type FantasyImportRequest
	} from '$lib/api';
	import { SEASONS, COLLECTOR_TYPES, SUMMARY_LEVELS, RANK_TYPES, IMPORT_PRESETS } from '$lib/constants';

	// ── Tab state ──
	let activeTab = $state('inventory');

	// ── Page-level filters ──
	let filterSource = $state('');
	let filterSeason = $state(0);        // 0 = all
	let filterStatType = $state('');
	let filterSeasonType = $state('');
	let filterRankType = $state('');
	let filterJobStatus = $state('');
	let filterCollector = $state('');

	// Derive unique filter options from UNFILTERED data so dropdowns always show all options
	let availableSources = $derived.by(() => {
		const inv = unfilteredInventory;
		if (!inv) return [];
		const all = [...inv.players, ...inv.stats, ...inv.games, ...inv.rankings];
		return [...new Set(all.map(r => r.source).filter(Boolean))].sort();
	});
	let availableSeasons = $derived.by(() => {
		const inv = unfilteredInventory;
		if (!inv) return [];
		const all = [...inv.stats, ...inv.games, ...inv.rankings];
		return [...new Set(all.map(r => r.season).filter((s): s is number => s != null))].sort((a, b) => b - a);
	});
	let availableStatTypes = $derived.by(() => {
		const inv = unfilteredInventory;
		if (!inv) return [];
		return [...new Set(inv.stats.map(r => r.stat_type).filter((s): s is string => !!s))].sort();
	});
	let availableSeasonTypes = $derived.by(() => {
		const inv = unfilteredInventory;
		if (!inv) return [];
		return [...new Set(inv.stats.map(r => r.season_type).filter((s): s is string => !!s))].sort();
	});
	let availableRankTypes = $derived.by(() => {
		const inv = unfilteredInventory;
		if (!inv) return [];
		return [...new Set(inv.rankings.map(r => r.rank_type).filter((s): s is string => !!s))].sort();
	});
	let availableJobStatuses = $derived.by(() => {
		if (!jobs.length) return [];
		return [...new Set(jobs.map(j => j.status))].sort();
	});
	let availableCollectors = $derived.by(() => {
		if (!jobs.length) return [];
		return [...new Set(jobs.map(j => j.collector_type))].sort();
	});

	let hasAnyFilter = $derived(!!filterSource || !!filterSeason || !!filterStatType || !!filterSeasonType || !!filterRankType || !!filterJobStatus || !!filterCollector);

	function clearFilters() {
		filterSource = '';
		filterSeason = 0;
		filterStatType = '';
		filterSeasonType = '';
		filterRankType = '';
		filterJobStatus = '';
		filterCollector = '';
		loadInventory();
		refreshJobs();
	}

	function applyFilters() {
		offset = 0;
		loadInventory();
		refreshJobs();
	}

	// ── Inventory state ──
	let inventory: InventoryResponse | null = $state(null);
	let unfilteredInventory: InventoryResponse | null = $state(null); // for filter option dropdowns
	let inventoryLoading = $state(true);
	let auditResult: AuditResult | null = $state(null);
	let auditLoading = $state(false);
	let auditSeason = $state(0);

	// ── Import state (migrated from jobs page) ──
	let importing = $state(false);
	let importMessage = $state('');
	let showImportForm = $state(true);
	let selectedCollector = $state('nflreadpy');
	let selectedSeason = $state(SEASONS[0]);
	let selectedStrategy = $state('merge');
	let selectedSummaryLevel = $state('week');
	let selectedRankType = $state('draft');
	let batchImporting = $state(false);
	let batchMessage = $state('');
	let batchSeason = $state(SEASONS[0]);
	let importTab = $state('batch');
	let fullImporting = $state(false);
	let fullImportPhase = $state('');
	let fullImportMessage = $state('');
	let fullFromSeason = $state(2020);
	let fullToSeason = $state(SEASONS[0]);

	// Fantasy import state
	let fantasyPlatform = $state('yahoo');
	let fantasyLeagueId = $state('');
	let fantasySeason = $state(SEASONS[0]);
	let fantasySwid = $state('');
	let fantasyEspnS2 = $state('');
	let fantasyImporting = $state(false);
	let fantasyMessage = $state('');

	const strategies = [
		{ value: 'merge', label: 'Merge', desc: 'Upsert — update existing, insert new' },
		{ value: 'replace', label: 'Replace', desc: 'Delete all from source, insert fresh' },
		{ value: 'append', label: 'Append', desc: 'Insert all, no deduplication' },
		{ value: 'dry_run', label: 'Dry Run', desc: 'Validate only, skip DB writes' },
	];

	// ── History state (migrated from jobs page) ──
	let jobs: Job[] = $state([]);
	let totalJobs = $state(0);
	let totalPlayers = $state(0);
	let historyLoading = $state(true);
	let summary: JobSummary = $state({ pending: 0, running: 0, completed: 0, failed: 0, total: 0 });
	let pollTimer: ReturnType<typeof setInterval> | null = $state(null);
	let hasActiveJobs = $state(false);
	let cleaningUp = $state(false);
	let aborting = $state(false);
	let abortingJobId: number | null = $state(null);
	let offset = $state(0);
	const limit = 50;
	let expandedErrors: Set<number> = $state(new Set());

	// ── Lazy tab loading ──
	let inventoryLoaded = $state(false);
	let historyLoaded = $state(false);

	const COLLECTOR_LABELS: Record<string, string> = {
		nflreadpy: 'Rosters',
		nflreadpy_stats: 'Stats',
		nflreadpy_schedules: 'Schedules',
		nflreadpy_ff_rankings: 'Rankings',
	};

	// ── Inventory logic ──
	const CACHE_KEY = 'data_inventory_cache';

	function getCachedInventory(): InventoryResponse | null {
		try {
			const raw = sessionStorage.getItem(CACHE_KEY);
			if (!raw) return null;
			const { data, ts } = JSON.parse(raw);
			// Expire after 5 minutes
			if (Date.now() - ts > 5 * 60 * 1000) return null;
			return data;
		} catch { return null; }
	}

	function setCachedInventory(data: InventoryResponse) {
		try { sessionStorage.setItem(CACHE_KEY, JSON.stringify({ data, ts: Date.now() })); } catch {}
	}

	async function loadInventory() {
		// Show cached data instantly (stale-while-revalidate)
		if (!inventory && !hasAnyFilter) {
			const cached = getCachedInventory();
			if (cached) {
				inventory = cached;
				unfilteredInventory = cached;
				inventoryLoaded = true;
				// Still refresh in background, but don't show loading state
				inventoryLoading = false;
			}
		}

		if (!inventory) inventoryLoading = true;
		try {
			const f: InventoryFilter = {};
			if (filterSource) f.source = filterSource;
			if (filterSeason) f.season = filterSeason;
			if (filterStatType) f.stat_type = filterStatType;
			if (filterSeasonType) f.season_type = filterSeasonType;
			if (filterRankType) f.rank_type = filterRankType;
			inventory = await getInventory(f);
			// Cache unfiltered results
			if (!hasAnyFilter) setCachedInventory(inventory);
			if (!unfilteredInventory) {
				unfilteredInventory = hasAnyFilter ? await getInventory() : inventory;
			}
			inventoryLoaded = true;
		} catch (e) {
			console.error('Failed to load inventory', e);
		} finally {
			inventoryLoading = false;
		}
	}

	async function triggerAudit() {
		auditLoading = true;
		auditResult = null;
		try {
			auditResult = await runAudit('player_stats', auditSeason || undefined);
		} catch (e) {
			console.error('Failed to run audit', e);
		} finally {
			auditLoading = false;
		}
	}

	// Quick import from inventory row
	async function quickImport(collectorType: string, season: number) {
		importing = true;
		importMessage = '';
		try {
			const opts: Record<string, unknown> = {
				collector_type: collectorType,
				seasons: [season],
				strategy: 'merge',
			};
			if (collectorType === 'nflreadpy_stats') opts.summary_level = 'week';
			if (collectorType === 'nflreadpy_ff_rankings') opts.rank_type = 'draft';
			const res = await startImport(opts);
			importMessage = `Dispatched → ${res.job_id.slice(0, 8)}...`;
			setTimeout(async () => { await refreshJobs(); startPolling(); }, 500);
		} catch (e) {
			importMessage = `Error: ${e}`;
		} finally {
			importing = false;
		}
	}

	// ── History logic ──
	async function loadHistory() {
		historyLoading = true;
		await refreshJobs();
		historyLoaded = true;
		historyLoading = false;
	}

	async function refreshJobs() {
		try {
			const f: JobFilter = {};
			if (filterCollector) f.collector_type = filterCollector;
			if (filterJobStatus) f.status = filterJobStatus;
			if (filterSeason) f.season = filterSeason;
			const [jobRes, playerRes, summaryRes] = await Promise.all([
				listJobs(offset, limit, f),
				listPlayers({ limit: 1 }),
				getJobSummary(),
			]);
			jobs = jobRes.items;
			totalJobs = jobRes.total;
			totalPlayers = playerRes.total;
			summary = summaryRes;
			const active = summary.pending > 0 || summary.running > 0;
			hasActiveJobs = active;
			if (!active && pollTimer) stopPolling();
		} catch (e) {
			console.error('Failed to load jobs', e);
		}
	}

	function startPolling() {
		if (pollTimer) return;
		hasActiveJobs = true;
		pollTimer = setInterval(refreshJobs, 3000);
	}

	function stopPolling() {
		if (pollTimer) { clearInterval(pollTimer); pollTimer = null; }
	}

	async function cleanupStuck() {
		cleaningUp = true;
		try {
			const res = await cleanupStuckJobs();
			importMessage = res.cleaned > 0
				? `Cleaned up ${res.cleaned} stuck job${res.cleaned > 1 ? 's' : ''}`
				: 'No stuck jobs found';
			await refreshJobs();
		} catch (e) { importMessage = 'Cleanup failed'; }
		finally { cleaningUp = false; }
	}

	async function handleAbortJob(id: number) {
		abortingJobId = id;
		try {
			const res = await abortJob(id);
			importMessage = `Aborted job #${id}` + (res.revoke_error ? ' (revoke warning: ' + res.revoke_error + ')' : '');
			await refreshJobs();
		} catch (e) { importMessage = `Failed to abort job #${id}`; }
		finally { abortingJobId = null; }
	}

	async function handleAbortAll() {
		aborting = true;
		try {
			const res = await abortAllJobs();
			importMessage = res.aborted > 0
				? `Aborted ${res.aborted} job${res.aborted > 1 ? 's' : ''} (${res.celery_revoked} tasks revoked)`
				: 'No active jobs to abort';
			await refreshJobs();
		} catch (e) { importMessage = 'Abort all failed'; }
		finally { aborting = false; }
	}

	async function triggerImport() {
		importing = true;
		importMessage = '';
		try {
			const opts: Record<string, unknown> = {
				collector_type: selectedCollector,
				seasons: [selectedSeason],
				strategy: selectedStrategy,
			};
			if (selectedCollector === 'nflreadpy_stats') opts.summary_level = selectedSummaryLevel;
			if (selectedCollector === 'nflreadpy_ff_rankings') opts.rank_type = selectedRankType;
			const res = await startImport(opts);
			importMessage = `Dispatched → ${res.job_id.slice(0, 8)}...`;
			setTimeout(async () => { await refreshJobs(); startPolling(); }, 500);
		} catch (e) { importMessage = `Error: ${e}`; }
		finally { importing = false; }
	}

	async function triggerBatchPreset(presetIdx: number) {
		batchImporting = true;
		batchMessage = '';
		try {
			const preset = IMPORT_PRESETS[presetIdx];
			const imports = preset.build(batchSeason);
			const res = await batchImport(imports);
			batchMessage = `${preset.label}: ${res.dispatched} dispatched, ${res.failed} failed`;
			setTimeout(async () => { await refreshJobs(); startPolling(); }, 500);
		} catch (e) { batchMessage = `Error: ${e}`; }
		finally { batchImporting = false; }
	}

	async function triggerFantasyImport() {
		fantasyImporting = true;
		fantasyMessage = '';
		try {
			const req: FantasyImportRequest = {
				platform: fantasyPlatform,
				league_id: fantasyLeagueId,
				season: fantasySeason
			};
			if (fantasyPlatform === 'espn') {
				req.swid = fantasySwid;
				req.espn_s2 = fantasyEspnS2;
			}
			const result = await startFantasyImport(req);
			fantasyMessage = `Dispatched → ${result.job_id.slice(0, 8)}...`;
			setTimeout(async () => { await refreshJobs(); startPolling(); }, 500);
		} catch (e) { fantasyMessage = `Error: ${e}`; }
		finally { fantasyImporting = false; }
	}

	async function triggerFullImport() {
		fullImporting = true;
		fullImportPhase = 'Preparing...';
		fullImportMessage = '';
		try {
			const fromY = Math.min(fullFromSeason, fullToSeason);
			const toY = Math.max(fullFromSeason, fullToSeason);
			const seasons: number[] = [];
			for (let y = toY; y >= fromY; y--) seasons.push(y);
			fullImportPhase = `Dispatching ${seasons.length * 4} jobs...`;
			const result = await fullImport(seasons, (phase, res) => {
				fullImportPhase = `${phase}: ${res.dispatched} dispatched`;
				refreshJobs();
			});
			const dispatched = result.phase1.dispatched + result.phase2.dispatched;
			const failed = result.phase1.failed + result.phase2.failed;
			fullImportMessage = `Done — ${dispatched} dispatched, ${failed} failed`;
			fullImportPhase = '';
			setTimeout(async () => { await refreshJobs(); startPolling(); }, 500);
		} catch (e) { fullImportMessage = `Error: ${e}`; fullImportPhase = ''; }
		finally { fullImporting = false; }
	}

	function formatDuration(start: string, end: string | null): string {
		if (!end) return '—';
		const ms = new Date(end).getTime() - new Date(start).getTime();
		if (ms < 1000) return `${ms}ms`;
		if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`;
		return `${Math.floor(ms / 60000)}m ${Math.round((ms % 60000) / 1000)}s`;
	}

	function timeAgo(dateStr: string): string {
		const now = Date.now();
		const diff = now - new Date(dateStr).getTime();
		if (diff < 60000) return 'just now';
		if (diff < 3600000) return `${Math.floor(diff / 60000)}m ago`;
		if (diff < 86400000) return `${Math.floor(diff / 3600000)}h ago`;
		return new Date(dateStr).toLocaleDateString();
	}

	function badgeClass(status: string): string {
		if (status === 'completed' || status === 'success') return 'badge-success';
		if (status === 'running' || status === 'started' || status === 'STARTED') return 'badge-warning';
		if (status === 'pending' || status === 'PENDING') return 'badge-info';
		if (status === 'failed') return 'badge-error';
		return 'badge-ghost';
	}

	function jobSeasons(job: Job): string {
		const seasons = job.params?.seasons as number[] | undefined;
		if (!seasons || seasons.length === 0) return '—';
		if (seasons.length === 1) return String(seasons[0]);
		return `${Math.min(...seasons)}–${Math.max(...seasons)}`;
	}

	function jobStrategy(job: Job): string {
		return (job.params?.strategy as string) ?? '—';
	}

	function toggleError(id: number) {
		const next = new Set(expandedErrors);
		if (next.has(id)) next.delete(id); else next.add(id);
		expandedErrors = next;
	}

	function nextPage() { if (offset + limit < totalJobs) { offset += limit; loadHistory(); } }
	function prevPage() { if (offset > 0) { offset = Math.max(0, offset - limit); loadHistory(); } }

	onMount(() => {
		// Fire everything in parallel — inventory shows cached data instantly
		loadInventory();
		getJobSummary().then(s => {
			summary = s;
			hasActiveJobs = s.pending > 0 || s.running > 0;
			if (hasActiveJobs) startPolling();
		}).catch(() => {});
	});
	onDestroy(stopPolling);

	// Lazy-load data when tabs change
	$effect(() => {
		if (activeTab === 'inventory' && !inventoryLoaded) {
			loadInventory();
		} else if (activeTab === 'history' && !historyLoaded) {
			loadHistory();
		} else if (activeTab === 'import' && !historyLoaded) {
			// Import tab shows active jobs indicator, needs summary
			refreshJobs();
		}
	});
</script>

<PageHeader title="Data Management" breadcrumbs={[{ label: 'System' }, { label: 'Data' }]}>
	{#snippet actions()}
		{#if hasActiveJobs}
			<span class="loading loading-ring loading-xs text-warning"></span>
			<span class="text-xs text-warning font-semibold">
				{summary.running} running · {summary.pending} queued
			</span>
		{/if}
		{#if importMessage}
			<span class="text-xs text-success font-semibold">{importMessage}</span>
		{/if}
	{/snippet}
</PageHeader>

<!-- Filter Bar -->
<div class="card bg-base-100 shadow-sm px-4 py-2.5 mb-4">
	<div class="flex flex-wrap items-center gap-2">
		<span class="text-xs font-medium text-base-content/50">FILTERS</span>

		<select class="select select-bordered select-xs w-28" bind:value={filterSource} onchange={applyFilters}>
			<option value="">All Sources</option>
			{#each availableSources as src}<option value={src}>{src}</option>{/each}
		</select>

		<select class="select select-bordered select-xs w-24" bind:value={filterSeason} onchange={applyFilters}>
			<option value={0}>All Seasons</option>
			{#each availableSeasons as yr}<option value={yr}>{yr}</option>{/each}
		</select>

		{#if availableStatTypes.length > 0}
			<select class="select select-bordered select-xs w-24" bind:value={filterStatType} onchange={applyFilters}>
				<option value="">All Stat Types</option>
				{#each availableStatTypes as st}<option value={st}>{st}</option>{/each}
			</select>
		{/if}

		{#if availableSeasonTypes.length > 0}
			<select class="select select-bordered select-xs w-24" bind:value={filterSeasonType} onchange={applyFilters}>
				<option value="">All Szn Types</option>
				{#each availableSeasonTypes as st}<option value={st}>{st}</option>{/each}
			</select>
		{/if}

		{#if availableRankTypes.length > 0}
			<select class="select select-bordered select-xs w-24" bind:value={filterRankType} onchange={applyFilters}>
				<option value="">All Rank Types</option>
				{#each availableRankTypes as rt}<option value={rt}>{rt}</option>{/each}
			</select>
		{/if}

		<div class="border-l border-base-300 h-4 mx-1"></div>

		{#if availableCollectors.length > 0}
			<select class="select select-bordered select-xs w-28" bind:value={filterCollector} onchange={applyFilters}>
				<option value="">All Job Types</option>
				{#each availableCollectors as ct}<option value={ct}>{ct}</option>{/each}
			</select>
		{/if}

		{#if availableJobStatuses.length > 0}
			<select class="select select-bordered select-xs w-28" bind:value={filterJobStatus} onchange={applyFilters}>
				<option value="">All Statuses</option>
				{#each availableJobStatuses as st}<option value={st}>{st}</option>{/each}
			</select>
		{/if}

		{#if hasAnyFilter}
			<button class="btn btn-ghost btn-xs text-error" onclick={clearFilters}>✕ Clear</button>
		{/if}
	</div>
</div>

<!-- Tabs -->
<div role="tablist" class="tabs tabs-bordered mb-4">
	<button role="tab" class="tab" class:tab-active={activeTab === 'inventory'} onclick={() => activeTab = 'inventory'}>📦 Inventory</button>
	<button role="tab" class="tab" class:tab-active={activeTab === 'import'} onclick={() => activeTab = 'import'}>⚡ Import</button>
	<button role="tab" class="tab" class:tab-active={activeTab === 'history'} onclick={() => activeTab = 'history'}>📋 History</button>
</div>

<!-- ═══════════════════════ INVENTORY TAB ═══════════════════════ -->
{#if activeTab === 'inventory'}
	{#if inventoryLoading && !inventory}
		<!-- Skeleton placeholder — renders instantly -->
		<div class="grid grid-cols-2 md:grid-cols-4 gap-3 mb-5">
			{#each [1,2,3,4] as _}
				<div class="card bg-base-100 shadow-sm p-4">
					<div class="skeleton h-4 w-16 mb-2"></div>
					<div class="skeleton h-7 w-24"></div>
				</div>
			{/each}
		</div>
		<div class="card bg-base-100 shadow-sm p-4 mb-5">
			<div class="skeleton h-4 w-32 mb-3"></div>
			<div class="skeleton h-8 w-full"></div>
		</div>
		{#each [1,2] as _}
			<div class="skeleton h-4 w-28 mb-2"></div>
			<div class="card bg-base-100 shadow-sm overflow-hidden mb-5">
				<div class="p-3 space-y-2">
					{#each [1,2,3] as __}
						<div class="skeleton h-5 w-full"></div>
					{/each}
				</div>
			</div>
		{/each}
	{:else if inventory}
		<!-- Grand Totals -->
		<div class="grid grid-cols-2 md:grid-cols-4 gap-3 mb-5">
			<StatCard label="Players" value={inventory.totals.players} icon="lucide--users" />
			<StatCard label="Stat Rows" value={inventory.totals.stats} icon="lucide--bar-chart-3" />
			<StatCard label="Games" value={inventory.totals.games} icon="lucide--trophy" />
			<StatCard label="Rankings" value={inventory.totals.rankings} icon="lucide--star" />
		</div>

		<!-- Audit Section -->
		<div class="card bg-base-100 shadow-sm mb-5 p-4">
			<div class="flex items-center gap-3 mb-2">
				<h3 class="font-bold text-sm">🔍 Data Audit</h3>
				<select class="select select-bordered select-xs w-24" bind:value={auditSeason}>
					<option value={0}>All</option>
					{#each SEASONS as year}<option value={year}>{year}</option>{/each}
				</select>
				<button class="btn btn-primary btn-xs" onclick={triggerAudit} disabled={auditLoading}>
					{auditLoading ? '⏳ Running...' : 'Run Audit'}
				</button>
			</div>

			{#if auditResult}
				<div class="grid grid-cols-1 md:grid-cols-2 gap-3 mt-2">
					<!-- Duplicates -->
					<div class="bg-base-100 rounded-lg p-3 border border-base-300">
						<div class="text-xs font-bold mb-1 {auditResult.duplicates.length > 0 ? 'text-error' : 'text-success'}">
							{auditResult.duplicates.length > 0 ? '⚠ Duplicates Found' : '✓ No Duplicates'}
						</div>
						{#if auditResult.duplicates.length > 0}
							<div class="max-h-32 overflow-y-auto">
								<table class="table table-xs">
									<thead><tr><th>Season</th><th>Wk</th><th>Dups</th></tr></thead>
									<tbody>
										{#each auditResult.duplicates.slice(0, 20) as d}
											<tr><td>{d.season}</td><td>{d.week}</td><td class="text-error">{d.duplicates}</td></tr>
										{/each}
									</tbody>
								</table>
							</div>
							{#if auditResult.duplicates.length > 20}
								<div class="text-xs text-base-content/50 mt-1">...and {auditResult.duplicates.length - 20} more</div>
							{/if}
						{/if}
					</div>

					<!-- Completeness -->
					<div class="bg-base-100 rounded-lg p-3 border border-base-300">
						<div class="text-xs font-bold mb-1">📅 Season Completeness</div>
						<div class="max-h-32 overflow-y-auto">
							<table class="table table-xs">
								<thead><tr><th>Season</th><th>Expected</th><th>Actual</th><th>Missing</th></tr></thead>
								<tbody>
									{#each auditResult.completeness as c}
										<tr>
											<td>{c.season}</td>
											<td>{c.expected_weeks}</td>
											<td>{c.actual_weeks}</td>
											<td class="{c.missing_weeks > 0 ? 'text-warning font-bold' : 'text-success'}">
												{c.missing_weeks > 0 ? c.missing_weeks : '✓'}
											</td>
										</tr>
									{/each}
								</tbody>
							</table>
						</div>
					</div>

					<!-- Player Coverage -->
					<div class="bg-base-100 rounded-lg p-3 border border-base-300">
						<div class="text-xs font-bold mb-1">👥 Player Coverage</div>
						<div class="max-h-32 overflow-y-auto">
							<table class="table table-xs">
								<thead><tr><th>Season</th><th>Rostered</th><th>w/ Stats</th><th>Missing</th></tr></thead>
								<tbody>
									{#each auditResult.player_coverage as c}
										<tr>
											<td>{c.season}</td>
											<td>{c.rostered_players.toLocaleString()}</td>
											<td>{c.players_with_stats.toLocaleString()}</td>
											<td class="opacity-50">{c.missing_stats.toLocaleString()}</td>
										</tr>
									{/each}
								</tbody>
							</table>
						</div>
					</div>

					<!-- Ranking Resolution -->
					{#if auditResult.ranking_coverage}
						<div class="bg-base-100 rounded-lg p-3 border border-base-300">
							<div class="text-xs font-bold mb-1">🎯 Ranking Resolution</div>
							<div class="flex items-center gap-3">
								<div class="radial-progress text-primary text-xs" style="--value:{auditResult.ranking_coverage.resolution_pct}; --size:3rem;">
									{auditResult.ranking_coverage.resolution_pct}%
								</div>
								<div class="text-xs">
									<div>{auditResult.ranking_coverage.resolved_players.toLocaleString()} resolved</div>
									<div class="opacity-50">{auditResult.ranking_coverage.unresolved_players.toLocaleString()} unresolved</div>
									<div class="opacity-50">{auditResult.ranking_coverage.total_rankings.toLocaleString()} total</div>
								</div>
							</div>
						</div>
					{/if}
				</div>
			{/if}
		</div>

		<!-- Players Inventory -->
		{#if inventory.players.length > 0}
			<h3 class="font-bold text-sm mb-2 opacity-70">PLAYERS</h3>
			<div class="card bg-base-100 shadow-sm overflow-hidden mb-5">
				<table class="table table-zebra table-sm">
					<thead>
						<tr><th>Source</th><th>Rows</th><th>Players</th><th>Last Updated</th><th></th></tr>
					</thead>
					<tbody>
						{#each inventory.players as row}
							<tr class="hover">
								<td class="font-mono text-sm">{row.source}</td>
								<td>{row.rows.toLocaleString()}</td>
								<td>{row.distinct_players.toLocaleString()}</td>
								<td class="text-xs text-base-content/60">{timeAgo(row.last_updated)}</td>
								<td>
									<button class="btn btn-ghost btn-xs" onclick={() => quickImport('nflreadpy', SEASONS[0])} disabled={importing} title="Re-import latest rosters">
										↻
									</button>
								</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		{/if}

		<!-- Stats Inventory -->
		{#if inventory.stats.length > 0}
			<h3 class="font-bold text-sm mb-2 opacity-70">PLAYER STATS</h3>
			<div class="card bg-base-100 shadow-sm overflow-hidden mb-5">
				<table class="table table-zebra table-sm">
					<thead>
						<tr><th>Source</th><th>Season</th><th>Type</th><th>Szn Type</th><th>Rows</th><th>Players</th><th>Weeks</th><th>Last Updated</th><th></th></tr>
					</thead>
					<tbody>
						{#each inventory.stats as row}
							<tr class="hover">
								<td class="font-mono text-xs">{row.source}</td>
								<td class="font-mono font-bold">{row.season}</td>
								<td><span class="badge badge-sm badge-ghost">{row.stat_type ?? '—'}</span></td>
								<td><span class="badge badge-sm badge-outline">{row.season_type ?? '—'}</span></td>
								<td>{row.rows.toLocaleString()}</td>
								<td>{row.distinct_players.toLocaleString()}</td>
								<td class="text-xs">
									{#if row.week_count}
										{row.week_count} <span class="opacity-40">({row.min_week}–{row.max_week})</span>
									{:else}
										—
									{/if}
								</td>
								<td class="text-xs text-base-content/60">{timeAgo(row.last_updated)}</td>
								<td>
									<button class="btn btn-ghost btn-xs" onclick={() => quickImport('nflreadpy_stats', row.season ?? SEASONS[0])} disabled={importing} title="Re-import this season's stats">
										↻
									</button>
								</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		{/if}

		<!-- Games Inventory -->
		{#if inventory.games.length > 0}
			<h3 class="font-bold text-sm mb-2 opacity-70">GAMES / SCHEDULES</h3>
			<div class="card bg-base-100 shadow-sm overflow-hidden mb-5">
				<table class="table table-zebra table-sm">
					<thead>
						<tr><th>Source</th><th>Season</th><th>Games</th><th>Weeks</th><th>Last Updated</th><th></th></tr>
					</thead>
					<tbody>
						{#each inventory.games as row}
							<tr class="hover">
								<td class="font-mono text-xs">{row.source}</td>
								<td class="font-mono font-bold">{row.season}</td>
								<td>{row.rows.toLocaleString()}</td>
								<td class="text-xs">
									{#if row.week_count}
										{row.week_count} <span class="opacity-40">({row.min_week}–{row.max_week})</span>
									{:else}
										—
									{/if}
								</td>
								<td class="text-xs text-base-content/60">{timeAgo(row.last_updated)}</td>
								<td>
									<button class="btn btn-ghost btn-xs" onclick={() => quickImport('nflreadpy_schedules', row.season ?? SEASONS[0])} disabled={importing} title="Re-import this season's schedule">
										↻
									</button>
								</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		{/if}

		<!-- Rankings Inventory -->
		{#if inventory.rankings.length > 0}
			<h3 class="font-bold text-sm mb-2 opacity-70">FANTASY RANKINGS</h3>
			<div class="card bg-base-100 shadow-sm overflow-hidden mb-5">
				<table class="table table-zebra table-sm">
					<thead>
						<tr><th>Source</th><th>Season</th><th>Rank Type</th><th>Rows</th><th>Players</th><th>Last Updated</th><th></th></tr>
					</thead>
					<tbody>
						{#each inventory.rankings as row}
							<tr class="hover">
								<td class="font-mono text-xs">{row.source}</td>
								<td class="font-mono font-bold">{row.season}</td>
								<td><span class="badge badge-sm badge-ghost">{row.rank_type ?? '—'}</span></td>
								<td>{row.rows.toLocaleString()}</td>
								<td>{row.distinct_players.toLocaleString()}</td>
								<td class="text-xs text-base-content/60">{timeAgo(row.last_updated)}</td>
								<td>
									<button class="btn btn-ghost btn-xs" onclick={() => quickImport('nflreadpy_ff_rankings', row.season ?? SEASONS[0])} disabled={importing} title="Re-import rankings">
										↻
									</button>
								</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		{/if}

		<!-- Empty DB -->
		{#if inventory.totals.players === 0 && inventory.totals.stats === 0 && inventory.totals.games === 0 && inventory.totals.rankings === 0}
			<div class="card bg-base-100 shadow-sm p-8 text-center">
				<p class="text-lg font-bold text-warning mb-2">Database is empty</p>
				<p class="text-sm text-base-content/60">Switch to the Import tab to load data.</p>
			</div>
		{/if}

		<div class="flex justify-end mt-2">
			<button class="btn btn-ghost btn-xs" onclick={loadInventory}>↻ Refresh</button>
		</div>
	{/if}

<!-- ═══════════════════════ IMPORT TAB ═══════════════════════ -->
{:else if activeTab === 'import'}
	<div class="card bg-base-100 shadow-sm mb-5">
		<div class="card-body p-4 gap-0">
			<!-- Import sub-tabs -->
			<div role="tablist" class="tabs tabs-bordered mb-3">
				<button role="tab" class="tab tab-sm" class:tab-active={importTab === 'batch'} onclick={() => importTab = 'batch'}>Quick Batch</button>
				<button role="tab" class="tab tab-sm" class:tab-active={importTab === 'full'} onclick={() => importTab = 'full'}>Full Import</button>
				<button role="tab" class="tab tab-sm" class:tab-active={importTab === 'custom'} onclick={() => importTab = 'custom'}>Custom</button>
				<button role="tab" class="tab tab-sm" class:tab-active={importTab === 'fantasy'} onclick={() => importTab = 'fantasy'}>Fantasy League</button>
			</div>

			<!-- Quick Batch -->
			{#if importTab === 'batch'}
				<div class="flex flex-wrap gap-2 items-center">
					<select class="select select-bordered select-xs w-20" bind:value={batchSeason}>
						{#each SEASONS as year}<option value={year}>{year}</option>{/each}
					</select>
					{#each IMPORT_PRESETS as preset, i}
						<button class="btn btn-primary btn-xs" onclick={() => triggerBatchPreset(i)} disabled={batchImporting} title={preset.desc}>
							⚡ {preset.label}
						</button>
					{/each}
				</div>
				<p class="text-xs opacity-40 mt-2">{IMPORT_PRESETS.map(p => p.label + ': ' + p.desc).join(' · ')}</p>
				{#if batchMessage}<p class="text-xs text-success font-semibold mt-1">{batchMessage}</p>{/if}

			<!-- Full Import -->
			{:else if importTab === 'full'}
				<p class="text-xs text-base-content/50 mb-2">All data (rosters, stats, schedules, rankings) — merge strategy, safe to re-run.</p>
				<div class="flex flex-wrap gap-2 items-center">
					<span class="text-xs font-semibold opacity-50">From</span>
					<select class="select select-bordered select-xs w-20" bind:value={fullFromSeason}>
						{#each SEASONS as year}<option value={year}>{year}</option>{/each}
					</select>
					<span class="text-xs font-semibold opacity-50">To</span>
					<select class="select select-bordered select-xs w-20" bind:value={fullToSeason}>
						{#each SEASONS as year}<option value={year}>{year}</option>{/each}
					</select>
					<button class="btn btn-primary btn-xs" onclick={triggerFullImport} disabled={fullImporting}>
						{fullImporting ? '⏳ Running...' : '🚀 Import All'}
					</button>
					<span class="text-xs opacity-40">
						{Math.abs(fullToSeason - fullFromSeason) + 1} szn × 4 = {(Math.abs(fullToSeason - fullFromSeason) + 1) * 4} jobs
					</span>
				</div>
				{#if fullImportPhase}<p class="text-xs text-warning font-semibold mt-1">⏳ {fullImportPhase}</p>{/if}
				{#if fullImportMessage}<p class="text-xs text-success font-semibold mt-1">✓ {fullImportMessage}</p>{/if}

			<!-- Custom -->
			{:else}
				<div class="grid grid-cols-2 gap-x-3 gap-y-2 max-w-md">
					<div>
						<div class="text-xs font-semibold opacity-50 mb-0.5">Data Type</div>
						<select class="select select-bordered select-xs w-full" bind:value={selectedCollector}>
							{#each COLLECTOR_TYPES as ct}<option value={ct.value}>{ct.label}</option>{/each}
						</select>
					</div>
					<div>
						<div class="text-xs font-semibold opacity-50 mb-0.5">Season</div>
						<select class="select select-bordered select-xs w-full" bind:value={selectedSeason}>
							{#each SEASONS as year}<option value={year}>{year}</option>{/each}
						</select>
					</div>
					{#if selectedCollector === 'nflreadpy_stats'}
						<div>
							<div class="text-xs font-semibold opacity-50 mb-0.5">Summary</div>
							<select class="select select-bordered select-xs w-full" bind:value={selectedSummaryLevel}>
								{#each SUMMARY_LEVELS as sl}<option value={sl.value}>{sl.label}</option>{/each}
							</select>
						</div>
					{/if}
					{#if selectedCollector === 'nflreadpy_ff_rankings'}
						<div>
							<div class="text-xs font-semibold opacity-50 mb-0.5">Rank Type</div>
							<select class="select select-bordered select-xs w-full" bind:value={selectedRankType}>
								{#each RANK_TYPES as rt}<option value={rt.value}>{rt.label}</option>{/each}
							</select>
						</div>
					{/if}
					<div>
						<div class="text-xs font-semibold opacity-50 mb-0.5">Strategy</div>
						<select class="select select-bordered select-xs w-full" bind:value={selectedStrategy}>
							{#each strategies as s}<option value={s.value}>{s.label} — {s.desc}</option>{/each}
						</select>
					</div>
				</div>
				<button class="btn btn-primary btn-xs mt-3 self-start" onclick={triggerImport} disabled={importing}>
					{importing ? 'Dispatching...' : 'Launch Import'}
				</button>

			<!-- Fantasy League Import -->
			{:else if importTab === 'fantasy'}
				<p class="text-xs text-base-content/50 mb-2">Import a Yahoo or ESPN fantasy league — teams, rosters, and player matching.</p>
				<div class="grid grid-cols-2 gap-x-3 gap-y-2 max-w-lg">
					<div>
						<div class="text-xs font-semibold opacity-50 mb-0.5">Platform</div>
						<select class="select select-bordered select-xs w-full" bind:value={fantasyPlatform}>
							<option value="yahoo">Yahoo</option>
							<option value="espn">ESPN</option>
						</select>
					</div>
					<div>
						<div class="text-xs font-semibold opacity-50 mb-0.5">League ID</div>
						<input class="input input-bordered input-xs w-full" bind:value={fantasyLeagueId} placeholder="e.g. 12345" />
					</div>
					<div>
						<div class="text-xs font-semibold opacity-50 mb-0.5">Season</div>
						<select class="select select-bordered select-xs w-full" bind:value={fantasySeason}>
							{#each SEASONS as year}<option value={year}>{year}</option>{/each}
						</select>
					</div>
					{#if fantasyPlatform === 'espn'}
						<div>
							<div class="text-xs font-semibold opacity-50 mb-0.5">SWID Cookie</div>
							<input class="input input-bordered input-xs w-full" bind:value={fantasySwid} placeholder="XXXXXXXX-XXXX-..." />
						</div>
						<div class="col-span-2">
							<div class="text-xs font-semibold opacity-50 mb-0.5">espn_s2 Cookie</div>
							<input class="input input-bordered input-xs w-full" bind:value={fantasyEspnS2} placeholder="AEAB..." />
						</div>
					{/if}
				</div>
				<button class="btn btn-primary btn-xs mt-3 self-start" onclick={triggerFantasyImport} disabled={fantasyImporting || !fantasyLeagueId}>
					{fantasyImporting ? 'Dispatching...' : 'Import League'}
				</button>
				{#if fantasyMessage}<p class="text-xs font-semibold mt-1" class:text-success={!fantasyMessage.startsWith('Error')} class:text-error={fantasyMessage.startsWith('Error')}>{fantasyMessage}</p>{/if}
			{/if}
		</div>
	</div>

	<!-- Active jobs indicator -->
	{#if hasActiveJobs}
		<div class="card bg-base-100 shadow-sm border border-warning/30 p-3 mb-4">
			<div class="flex items-center gap-2">
				<span class="loading loading-ring loading-xs text-warning"></span>
				<span class="text-sm font-semibold">{summary.running} running · {summary.pending} queued</span>
				<button class="btn btn-error btn-xs ml-auto" onclick={handleAbortAll} disabled={aborting}>
					{aborting ? 'Aborting...' : '⛔ Abort All'}
				</button>
				<button class="btn btn-warning btn-outline btn-xs" onclick={cleanupStuck} disabled={cleaningUp}>
					{cleaningUp ? 'Cleaning...' : '🧹 Cleanup Stuck'}
				</button>
			</div>
		</div>
	{/if}

<!-- ═══════════════════════ HISTORY TAB ═══════════════════════ -->
{:else if activeTab === 'history'}
	{#if hasActiveJobs}
		<div class="flex items-center gap-2 mb-3">
			<span class="loading loading-ring loading-xs text-warning"></span>
			<span class="text-xs text-warning font-semibold">{summary.running} running · {summary.pending} queued</span>
			<button class="btn btn-error btn-xs ml-auto" onclick={handleAbortAll} disabled={aborting}>
				{aborting ? 'Aborting...' : '⛔ Abort All'}
			</button>
			<button class="btn btn-warning btn-outline btn-xs" onclick={cleanupStuck} disabled={cleaningUp}>
				{cleaningUp ? 'Cleaning...' : '🧹 Cleanup Stuck'}
			</button>
		</div>
	{/if}

	<!-- Queue Dashboard -->
	<div class="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-5 gap-3 mb-5">
		<StatCard label="Queued" value={summary.pending} icon="lucide--clock" />
		<StatCard label="Running" value={summary.running} icon="lucide--loader" />
		<StatCard label="Completed" value={summary.completed} icon="lucide--check-circle" />
		<StatCard label="Failed" value={summary.failed} icon="lucide--x-circle" />
		<StatCard label="Players" value={totalPlayers} icon="lucide--users" />
	</div>

	{#if historyLoading}
		<div class="card bg-base-100 shadow-sm overflow-hidden">
			<div class="p-3 space-y-2">
				{#each [1,2,3,4,5] as _}
					<div class="skeleton h-6 w-full"></div>
				{/each}
			</div>
		</div>
	{:else}
		<div class="card bg-base-100 shadow-sm overflow-hidden">
			<div class="table-scroll-wrap">
				<table class="table table-zebra table-pin-rows table-sm table-responsive">
					<thead>
						<tr>
							<th class="w-12">ID</th>
							<th>Type</th>
							<th>Season</th>
							<th>Strategy</th>
							<th>Status</th>
							<th>Records</th>
							<th>Duration</th>
							<th>When</th>
						</tr>
					</thead>
					<tbody>
						{#each jobs as job}
							{@const isActive = job.status === 'running' || job.status === 'started' || job.status === 'STARTED'}
							{@const isPending = job.status === 'pending' || job.status === 'PENDING'}
							{@const isFailed = job.status === 'failed'}
							{@const hasProg = isActive && job.progress !== null && job.progress !== undefined}
							<tr class="hover {isActive ? 'bg-warning/5' : ''} {isPending ? 'bg-info/5' : ''}">
								<td class="font-mono text-xs text-base-content/60">#{job.id}</td>
								<td>
									<span class="font-semibold text-sm">{COLLECTOR_LABELS[job.collector_type] ?? job.collector_type}</span>
								</td>
								<td class="font-mono text-sm">{jobSeasons(job)}</td>
								<td><span class="text-xs opacity-70">{jobStrategy(job)}</span></td>
								<td>
									<div class="flex flex-col gap-0.5">
										<span class="badge {badgeClass(job.status)} badge-sm">{job.status}</span>
										{#if hasProg}
											<progress class="progress progress-warning w-16 h-1.5" value={job.progress ?? 0} max="1"></progress>
											<span class="text-[10px] opacity-50">{Math.round((job.progress ?? 0) * 100)}%</span>
										{/if}
										{#if isPending}
											<span class="text-[10px] opacity-40">in queue</span>
										{/if}
										{#if isActive || isPending}
											<button class="btn btn-error btn-outline btn-xs mt-0.5" onclick={() => handleAbortJob(job.id)} disabled={abortingJobId === job.id}>
												{abortingJobId === job.id ? '...' : '⛔'}
											</button>
										{/if}
									</div>
								</td>
								<td>
									{#if isPending}
										<span class="opacity-30">—</span>
									{:else}
										<div class="flex flex-col text-xs leading-tight">
											<span>{job.records_fetched.toLocaleString()} fetched</span>
											{#if job.records_inserted > 0}
												<span class="text-success">+{job.records_inserted.toLocaleString()}</span>
											{/if}
											{#if job.records_updated > 0}
												<span class="text-info">↻{job.records_updated.toLocaleString()}</span>
											{/if}
											{#if job.records_skipped > 0}
												<span class="opacity-40">⊘{job.records_skipped.toLocaleString()}</span>
											{/if}
										</div>
									{/if}
								</td>
								<td class="text-sm">{formatDuration(job.started_at, job.finished_at)}</td>
								<td class="text-xs text-base-content/60" title={new Date(job.started_at).toLocaleString()}>
									{timeAgo(job.started_at)}
								</td>
							</tr>
							{#if isFailed && job.error_message}
								<tr class="bg-error/5">
									<td></td>
									<td colspan="7">
										<button class="text-xs text-error cursor-pointer hover:underline" onclick={() => toggleError(job.id)}>
											{expandedErrors.has(job.id) ? '▼' : '►'} Error details
										</button>
										{#if expandedErrors.has(job.id)}
											<pre class="text-xs text-error mt-1 whitespace-pre-wrap break-all max-h-24 overflow-y-auto">{job.error_message}</pre>
										{/if}
									</td>
								</tr>
							{/if}
						{:else}
							<tr>
								<td colspan="8" class="text-center text-base-content/50 py-8">No jobs logged yet.</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		</div>

		<div class="flex justify-between items-center mt-4 text-sm opacity-70">
			<span>{totalJobs} job{totalJobs === 1 ? '' : 's'} total</span>
			<div class="join">
				<button class="join-item btn btn-sm" onclick={prevPage} disabled={offset === 0}>◄ Prev</button>
				<button class="join-item btn btn-sm" onclick={nextPage} disabled={offset + limit >= totalJobs}>Next ►</button>
			</div>
		</div>
	{/if}
{/if}
