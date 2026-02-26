<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { listJobs, listPlayers, startImport, batchImport, type Job } from '$lib/api';
	import { SEASONS, COLLECTOR_TYPES, SUMMARY_LEVELS, RANK_TYPES, IMPORT_PRESETS } from '$lib/constants';

	let jobs: Job[] = $state([]);
	let total = $state(0);
	let totalPlayers = $state(0);
	let latestJob: Job | null = $state(null);
	let loading = $state(true);        // only true on first load
	let importing = $state(false);
	let importMessage = $state('');
	let showImportForm = $state(false);

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

	// Polling
	let pollTimer: ReturnType<typeof setInterval> | null = $state(null);
	let hasActiveJobs = $state(false);

	let offset = $state(0);
	const limit = 20;

	const strategies = [
		{ value: 'merge', label: 'MERGE', desc: 'Upsert — update existing, insert new' },
		{ value: 'replace', label: 'REPLACE', desc: 'Delete all from source, insert fresh' },
		{ value: 'append', label: 'APPEND', desc: 'Insert all, no deduplication' },
		{ value: 'dry_run', label: 'DRY RUN', desc: 'Validate only, skip DB writes' },
	];

	// Initial load — shows loading spinner
	async function loadJobs() {
		loading = true;
		await refreshJobs();
		loading = false;
	}

	// Silent background refresh — no loading flash
	async function refreshJobs() {
		try {
			const [jobRes, playerRes] = await Promise.all([
				listJobs(offset, limit),
				listPlayers({ limit: 1 }),
			]);
			jobs = jobRes.items;
			total = jobRes.total;
			totalPlayers = playerRes.total;
			latestJob = jobRes.items[0] ?? null;

			// Check if any jobs are still running
			const active = jobs.some(j => j.status === 'running' || j.status === 'pending' || j.status === 'PENDING' || j.status === 'STARTED');
			hasActiveJobs = active;

			// If nothing active, stop polling
			if (!active && pollTimer) {
				stopPolling();
			}
		} catch (e) {
			console.error('Failed to load jobs', e);
		}
	}

	function startPolling() {
		if (pollTimer) return; // already polling
		hasActiveJobs = true;
		pollTimer = setInterval(refreshJobs, 3000);
	}

	function stopPolling() {
		if (pollTimer) {
			clearInterval(pollTimer);
			pollTimer = null;
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
			importMessage = `DISPATCHED → ${res.job_id.slice(0, 8)}...`;
			showImportForm = false;
			// Start auto-polling — refreshes every 3s until done
			setTimeout(async () => { await refreshJobs(); startPolling(); }, 1000);
		} catch (e) {
			importMessage = `ERR: ${e}`;
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
			setTimeout(async () => { await refreshJobs(); startPolling(); }, 1000);
		} catch (e) {
			batchMessage = `ERR: ${e}`;
		} finally {
			batchImporting = false;
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
		if (!end) return 'RUNNING...';
		const ms = new Date(end).getTime() - new Date(start).getTime();
		if (ms < 1000) return `${ms}ms`;
		return `${(ms / 1000).toFixed(1)}s`;
	}

	onMount(loadJobs);
	onDestroy(stopPolling);
</script>

<div class="page-header">
	<h1>// IMPORT JOBS</h1>
	<div style="display: flex; gap: 0.75rem; align-items: center">
		{#if hasActiveJobs}
			<span class="pulse-dot"></span>
			<span style="font-family: var(--font-pixel); font-size: 0.4rem; color: var(--warning, #f0ad4e)">
				JOBS RUNNING...
			</span>
		{/if}
		{#if importMessage}
			<span style="font-family: var(--font-pixel); font-size: 0.4rem; color: var(--success)">
				{importMessage}
			</span>
		{/if}
		<button class="primary" onclick={() => showImportForm = !showImportForm}>
			{showImportForm ? 'CANCEL' : '+ NEW IMPORT'}
		</button>
	</div>
</div>

{#if !loading}
	<div class="stats-grid">
		<div class="card">
			<h3>» ROSTER</h3>
			<div class="stat-value">{totalPlayers.toLocaleString()}</div>
			<p style="color: var(--text-muted); font-size: 0.9rem; margin-top: 0.25rem">NFL PLAYERS</p>
		</div>
		<div class="card">
			<h3>» IMPORTS</h3>
			<div class="stat-value">{total}</div>
			<p style="color: var(--text-muted); font-size: 0.9rem; margin-top: 0.25rem">COMPLETED JOBS</p>
		</div>
		<div class="card">
			<h3>» LATEST</h3>
			{#if latestJob}
				<div class="stat-value" style="font-size: 0.7rem; margin-bottom: 0.5rem">
					<span class="badge {latestJob.status}">{latestJob.status}</span>
				</div>
				<p style="color: var(--text-muted); font-size: 0.9rem">
					{latestJob.records_fetched.toLocaleString()} FETCHED
				</p>
				<p style="color: var(--text-muted); font-size: 0.85rem">
					{new Date(latestJob.started_at).toLocaleDateString()}
				</p>
			{:else}
				<div class="stat-value" style="font-size: 0.6rem; color: var(--text-muted)">AWAITING</div>
			{/if}
		</div>
	</div>
{/if}

{#if showImportForm}
	<!-- Batch Presets -->
	<div class="card" style="margin-bottom: 1rem">
		<h3>» QUICK BATCH IMPORT</h3>
		<div style="display: flex; gap: 0.75rem; align-items: center; margin-top: 0.75rem; flex-wrap: wrap">
			<select bind:value={batchSeason} style="width: 100px">
				{#each SEASONS as year}
					<option value={year}>{year}</option>
				{/each}
			</select>
			{#each IMPORT_PRESETS as preset, i}
				<button
					class="primary"
					onclick={() => triggerBatchPreset(i)}
					disabled={batchImporting}
					title={preset.desc}
				>
					⚡ {preset.label}
				</button>
			{/each}
		</div>
		{#if batchMessage}
			<p style="font-family: var(--font-pixel); font-size: 0.4rem; color: var(--success); margin-top: 0.5rem">
				{batchMessage}
			</p>
		{/if}
		<p style="color: var(--text-muted); font-size: 0.85rem; margin-top: 0.5rem">
			{IMPORT_PRESETS.map(p => p.label + ': ' + p.desc).join(' · ')}
		</p>
	</div>

	<!-- Single Import Form -->
	<div class="card" style="margin-bottom: 1.5rem; max-width: 500px">
		<h3>» CONFIGURE IMPORT</h3>
		<div style="display: flex; flex-direction: column; gap: 1rem; margin-top: 0.75rem">
			<div>
				<label style="font-family: var(--font-pixel); font-size: 0.4rem; color: var(--text-muted); display: block; margin-bottom: 0.4rem">
					DATA TYPE
				</label>
				<select bind:value={selectedCollector} style="width: 100%">
					{#each COLLECTOR_TYPES as ct}
						<option value={ct.value}>{ct.label}</option>
					{/each}
				</select>
			</div>
			<div>
				<label style="font-family: var(--font-pixel); font-size: 0.4rem; color: var(--text-muted); display: block; margin-bottom: 0.4rem">
					SEASON
				</label>
				<select bind:value={selectedSeason} style="width: 100%">
					{#each SEASONS as year}
						<option value={year}>{year}</option>
					{/each}
				</select>
			</div>
			{#if selectedCollector === 'nflreadpy_stats'}
				<div>
					<label style="font-family: var(--font-pixel); font-size: 0.4rem; color: var(--text-muted); display: block; margin-bottom: 0.4rem">
						SUMMARY LEVEL
					</label>
					<select bind:value={selectedSummaryLevel} style="width: 100%">
						{#each SUMMARY_LEVELS as sl}
							<option value={sl.value}>{sl.label}</option>
						{/each}
					</select>
				</div>
			{/if}
			{#if selectedCollector === 'nflreadpy_ff_rankings'}
				<div>
					<label style="font-family: var(--font-pixel); font-size: 0.4rem; color: var(--text-muted); display: block; margin-bottom: 0.4rem">
						RANKING TYPE
					</label>
					<select bind:value={selectedRankType} style="width: 100%">
						{#each RANK_TYPES as rt}
							<option value={rt.value}>{rt.label}</option>
						{/each}
					</select>
				</div>
			{/if}
			<div>
				<label style="font-family: var(--font-pixel); font-size: 0.4rem; color: var(--text-muted); display: block; margin-bottom: 0.4rem">
					STRATEGY
				</label>
				<select bind:value={selectedStrategy} style="width: 100%">
					{#each strategies as s}
						<option value={s.value}>{s.label} — {s.desc}</option>
					{/each}
				</select>
			</div>
			<button class="primary" onclick={triggerImport} disabled={importing} style="align-self: flex-start">
				{importing ? 'DISPATCHING...' : 'LAUNCH IMPORT'}
			</button>
		</div>
	</div>
{/if}

{#if loading}
	<div class="card" style="text-align: center; padding: 2rem">
		<p style="font-family: var(--font-pixel); font-size: 0.6rem; color: var(--accent)">
			QUERYING LOGS...
		</p>
	</div>
{:else}
	<div class="card" style="padding: 0; overflow: hidden">
		<table>
			<thead>
				<tr>
					<th>ID</th>
					<th>TYPE</th>
					<th>STATUS</th>
					<th>FETCHED</th>
					<th>INSERTED</th>
					<th>UPDATED</th>
					<th>DURATION</th>
					<th>STARTED</th>
				</tr>
			</thead>
			<tbody>
				{#each jobs as job}
					<tr class:job-active={job.status === 'running' || job.status === 'pending' || job.status === 'PENDING' || job.status === 'STARTED'}>
						<td style="color: var(--accent)">#{job.id}</td>
						<td>{job.collector_type}</td>
						<td>
							<span class="badge {job.status}">{job.status}</span>
						</td>
						<td>{job.records_fetched.toLocaleString()}</td>
						<td>{job.records_inserted.toLocaleString()}</td>
						<td>{job.records_updated.toLocaleString()}</td>
						<td>{formatDuration(job.started_at, job.finished_at)}</td>
						<td>{new Date(job.started_at).toLocaleString()}</td>
					</tr>
				{:else}
					<tr>
						<td colspan="8" style="text-align: center; color: var(--text-muted); padding: 2rem; font-family: var(--font-pixel); font-size: 0.55rem">
							NO JOBS LOGGED. CLICK "+ NEW IMPORT" TO BEGIN.
						</td>
					</tr>
				{/each}
			</tbody>
		</table>
	</div>

	<div class="pagination">
		<span>{total} JOB{total === 1 ? '' : 'S'} TOTAL</span>
		<div style="display: flex; gap: 0.5rem">
			<button onclick={prevPage} disabled={offset === 0}>◄ PREV</button>
			<button onclick={nextPage} disabled={offset + limit >= total}>NEXT ►</button>
		</div>
	</div>
{/if}

<style>
	.stats-grid {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
		gap: 1rem;
		margin-bottom: 1.5rem;
	}
	.stat-value {
		font-family: var(--font-pixel);
		font-size: 1.2rem;
		color: var(--accent);
		margin-top: 0.5rem;
	}
	.pulse-dot {
		display: inline-block;
		width: 8px;
		height: 8px;
		border-radius: 50%;
		background: var(--warning, #f0ad4e);
		animation: pulse 1.2s ease-in-out infinite;
	}
	@keyframes pulse {
		0%, 100% { opacity: 1; transform: scale(1); }
		50% { opacity: 0.4; transform: scale(0.7); }
	}
	tr.job-active {
		background: rgba(240, 173, 78, 0.08);
	}
	tr.job-active td {
		border-left-color: var(--warning, #f0ad4e);
	}
</style>
