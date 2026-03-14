<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { listJobs, listPlayers, startImport, batchImport, fullImport, getJobSummary, cleanupStuckJobs, type Job, type JobSummary } from '$lib/api';
	import { SEASONS, COLLECTOR_TYPES, SUMMARY_LEVELS, RANK_TYPES, IMPORT_PRESETS } from '$lib/constants';

	let jobs: Job[] = $state([]);
	let total = $state(0);
	let totalPlayers = $state(0);
	let loading = $state(true);
	let importing = $state(false);
	let importMessage = $state('');
	let showImportForm = $state(false);

	// Job summary (queue dashboard)
	let summary: JobSummary = $state({ pending: 0, running: 0, completed: 0, failed: 0, total: 0 });

	// Import form state
	let selectedCollector = $state('nflreadpy');
	let selectedSeason = $state(SEASONS[0]);
	let selectedStrategy = $state('merge');
	let selectedSummaryLevel = $state('week');
	let selectedRankType = $state('draft');

	// Batch import state
	let batchImporting = $state(false);
	let batchMessage = $state('');
	let batchSeason = $state(SEASONS[0]);

	// Import tab
	let importTab = $state('batch');

	// Full import state
	let fullImporting = $state(false);
	let fullImportPhase = $state('');
	let fullImportMessage = $state('');
	let fullFromSeason = $state(2020);
	let fullToSeason = $state(SEASONS[0]);

	// Polling
	let pollTimer: ReturnType<typeof setInterval> | null = $state(null);
	let hasActiveJobs = $state(false);
	let cleaningUp = $state(false);

	let offset = $state(0);
	const limit = 50;

	// Expanded error rows
	let expandedErrors: Set<number> = $state(new Set());

	const strategies = [
		{ value: 'merge', label: 'Merge', desc: 'Upsert — update existing, insert new' },
		{ value: 'replace', label: 'Replace', desc: 'Delete all from source, insert fresh' },
		{ value: 'append', label: 'Append', desc: 'Insert all, no deduplication' },
		{ value: 'dry_run', label: 'Dry Run', desc: 'Validate only, skip DB writes' },
	];

	// Human-readable collector type names
	const COLLECTOR_LABELS: Record<string, string> = {
		nflreadpy: 'Rosters',
		nflreadpy_stats: 'Stats',
		nflreadpy_schedules: 'Schedules',
		nflreadpy_ff_rankings: 'Rankings',
	};

	async function loadJobs() {
		loading = true;
		await refreshJobs();
		loading = false;
	}

	async function refreshJobs() {
		try {
			const [jobRes, playerRes, summaryRes] = await Promise.all([
				listJobs(offset, limit),
				listPlayers({ limit: 1 }),
				getJobSummary(),
			]);
			jobs = jobRes.items;
			total = jobRes.total;
			totalPlayers = playerRes.total;
			summary = summaryRes;

			const active = summary.pending > 0 || summary.running > 0;
			hasActiveJobs = active;

			if (!active && pollTimer) {
				stopPolling();
			}
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
		if (pollTimer) {
			clearInterval(pollTimer);
			pollTimer = null;
		}
	}

	async function cleanupStuck() {
		cleaningUp = true;
		try {
			const res = await cleanupStuckJobs();
			importMessage = res.cleaned > 0
				? `Cleaned up ${res.cleaned} stuck job${res.cleaned > 1 ? 's' : ''}`
				: 'No stuck jobs found';
			await refreshJobs();
		} catch (e) {
			console.error('Failed to cleanup stuck jobs', e);
			importMessage = 'Cleanup failed';
		} finally {
			cleaningUp = false;
		}
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
			if (selectedCollector === 'nflreadpy_stats') {
				opts.summary_level = selectedSummaryLevel;
			}
			if (selectedCollector === 'nflreadpy_ff_rankings') {
				opts.rank_type = selectedRankType;
			}
			const res = await startImport(opts);
			importMessage = `Dispatched → ${res.job_id.slice(0, 8)}...`;
			showImportForm = false;
			setTimeout(async () => { await refreshJobs(); startPolling(); }, 500);
		} catch (e) {
			importMessage = `Error: ${e}`;
		} finally {
			importing = false;
		}
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
		} catch (e) {
			batchMessage = `Error: ${e}`;
		} finally {
			batchImporting = false;
		}
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

			const totalJobs = seasons.length * 4;
			fullImportPhase = `Dispatching ${totalJobs} jobs (${seasons.length} seasons × 4 types)...`;

			const result = await fullImport(seasons, (phase, res) => {
				fullImportPhase = `${phase}: ${res.dispatched} dispatched`;
				refreshJobs();
			});

			const dispatched = result.phase1.dispatched + result.phase2.dispatched;
			const failed = result.phase1.failed + result.phase2.failed;
			fullImportMessage = `Done — ${dispatched} jobs dispatched, ${failed} failed`;
			fullImportPhase = '';
			setTimeout(async () => { await refreshJobs(); startPolling(); }, 500);
		} catch (e) {
			fullImportMessage = `Error: ${e}`;
			fullImportPhase = '';
		} finally {
			fullImporting = false;
		}
	}

	function nextPage() {
		if (offset + limit < total) {
			offset += limit;
			loadJobs();
		}
	}

	function prevPage() {
		if (offset > 0) {
			offset = Math.max(0, offset - limit);
			loadJobs();
		}
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
		const then = new Date(dateStr).getTime();
		const diff = now - then;

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
		if (next.has(id)) next.delete(id);
		else next.add(id);
		expandedErrors = next;
	}

	onMount(loadJobs);
	onDestroy(stopPolling);
</script>

<div class="flex justify-between items-center mb-4">
	<h1 class="text-2xl font-bold text-primary tracking-wide">// IMPORT JOBS</h1>
	<div class="flex gap-3 items-center">
		{#if hasActiveJobs}
			<span class="loading loading-ring loading-xs text-warning"></span>
			<span class="text-xs text-warning font-semibold">
				{summary.running} running · {summary.pending} queued
			</span>
		{/if}
		{#if importMessage}
			<span class="text-xs text-success font-semibold">{importMessage}</span>
		{/if}
		{#if summary.running > 0 || summary.pending > 0}
			<button class="btn btn-error btn-outline btn-sm" onclick={cleanupStuck} disabled={cleaningUp}>
				{cleaningUp ? 'Cleaning...' : '🧹 Cleanup Stuck'}
			</button>
		{/if}
		<button class="btn btn-primary btn-sm" onclick={() => showImportForm = !showImportForm}>
			{showImportForm ? 'Cancel' : '+ New Import'}
		</button>
	</div>
</div>

{#if !loading}
	<!-- Queue Dashboard -->
	<div class="grid grid-cols-5 gap-3 mb-5">
		<div class="bg-base-200 border border-base-300 rounded-lg px-4 py-2 text-center">
			<div class="text-xs font-semibold opacity-50 mb-0.5">Queued</div>
			<div class="text-xl font-bold text-info">{summary.pending}</div>
		</div>
		<div class="bg-base-200 border border-base-300 rounded-lg px-4 py-2 text-center">
			<div class="text-xs font-semibold opacity-50 mb-0.5">Running</div>
			<div class="text-xl font-bold text-warning">{summary.running}</div>
		</div>
		<div class="bg-base-200 border border-base-300 rounded-lg px-4 py-2 text-center">
			<div class="text-xs font-semibold opacity-50 mb-0.5">Completed</div>
			<div class="text-xl font-bold text-success">{summary.completed}</div>
		</div>
		<div class="bg-base-200 border border-base-300 rounded-lg px-4 py-2 text-center">
			<div class="text-xs font-semibold opacity-50 mb-0.5">Failed</div>
			<div class="text-xl font-bold text-error">{summary.failed}</div>
		</div>
		<div class="bg-base-200 border border-base-300 rounded-lg px-4 py-2 text-center">
			<div class="text-xs font-semibold opacity-50 mb-0.5">Players</div>
			<div class="text-xl font-bold text-primary">{totalPlayers.toLocaleString()}</div>
		</div>
	</div>
{/if}

{#if showImportForm}
	<div class="card bg-base-200 shadow-md border border-base-300 mb-5">
		<div class="card-body p-4 gap-0">
			<!-- Tabs -->
			<div role="tablist" class="tabs tabs-bordered mb-3">
				<button role="tab" class="tab tab-sm" class:tab-active={importTab === 'batch'} onclick={() => importTab = 'batch'}>Quick Batch</button>
				<button role="tab" class="tab tab-sm" class:tab-active={importTab === 'full'} onclick={() => importTab = 'full'}>Full Import</button>
				<button role="tab" class="tab tab-sm" class:tab-active={importTab === 'custom'} onclick={() => importTab = 'custom'}>Custom</button>
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
				<p class="text-xs opacity-50 mb-2">All data (rosters, stats, schedules, rankings) — merge strategy, safe to re-run.</p>
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
			{/if}
		</div>
	</div>
{/if}

{#if loading}
	<div class="card bg-base-200 shadow-md border border-base-300 p-8 text-center">
		<span class="loading loading-dots loading-md text-primary"></span>
		<p class="text-sm opacity-60 mt-2">Querying logs...</p>
	</div>
{:else}
	<div class="card bg-base-100 shadow-md border border-base-300 overflow-hidden">
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
							<td class="font-mono text-xs opacity-60">#{job.id}</td>
							<td>
								<span class="font-semibold text-sm">{COLLECTOR_LABELS[job.collector_type] ?? job.collector_type}</span>
							</td>
							<td class="font-mono text-sm">{jobSeasons(job)}</td>
							<td>
								<span class="text-xs opacity-70">{jobStrategy(job)}</span>
							</td>
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
							<td class="text-xs opacity-60" title={new Date(job.started_at).toLocaleString()}>
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
							<td colspan="8" class="text-center opacity-50 py-8">No jobs logged. Click "+ New Import" to begin.</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	</div>

	<div class="flex justify-between items-center mt-4 text-sm opacity-70">
		<span>{total} job{total === 1 ? '' : 's'} total</span>
		<div class="join">
			<button class="join-item btn btn-sm" onclick={prevPage} disabled={offset === 0}>◄ Prev</button>
			<button class="join-item btn btn-sm" onclick={nextPage} disabled={offset + limit >= total}>Next ►</button>
		</div>
	</div>
{/if}
