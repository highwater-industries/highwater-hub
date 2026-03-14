<script lang="ts">
	import { onMount } from 'svelte';
	import { listStats, getLeaders, type PlayerStat, type StatFilter } from '$lib/api';
	import { NFL_TEAMS, POSITIONS, SEASONS, NFL_WEEKS, LEADER_STATS, STAT_TYPES, SOURCES } from '$lib/constants';

	let stats: PlayerStat[] = $state([]);
	let total = $state(0);
	let loading = $state(true);
	let fetching = $state(false);

	// Filters
	let search = $state('');
	let team = $state('');
	let position = $state('');
	let season: number | undefined = $state(undefined);
	let week: number | undefined = $state(undefined);
	let statType = $state('');
	let source = $state('');
	let offset = $state(0);
	const limit = 25;

	// Sorting
	let sortCol = $state('');
	let sortOrder = $state('');

	function toggleSort(col: string) {
		if (sortCol === col) {
			sortOrder = sortOrder === 'asc' ? 'desc' : 'asc';
		} else {
			sortCol = col;
			sortOrder = 'desc';
		}
		offset = 0;
		loadStats();
	}

	function sortIndicator(col: string): string {
		if (sortCol !== col) return '';
		return sortOrder === 'asc' ? ' ▲' : ' ▼';
	}

	// Leaders mode
	let showLeaders = $state(false);
	let leaderStat = $state('passing_yards');
	let leaderSeason = $state(SEASONS[0]);
	let leaderWeek: number | undefined = $state(undefined);
	let leaderPosition = $state('');
	let leaders: PlayerStat[] = $state([]);
	let loadingLeaders = $state(false);

	async function loadStats() {
		if (!stats.length) loading = true;
		fetching = true;
		try {
			const filter: StatFilter = { offset, limit };
			if (search) filter.search = search;
			if (team) filter.team = team;
			if (position) filter.position = position;
			if (season !== undefined) filter.season = season;
			if (week !== undefined) filter.week = week;
			if (statType) filter.stat_type = statType;
			if (source) filter.source = source;
			if (sortCol) filter.sort = sortCol;
			if (sortOrder) filter.order = sortOrder;

			const res = await listStats(filter);
			stats = res.items;
			total = res.total;
		} catch (e) {
			console.error('Failed to load stats', e);
		} finally {
			loading = false;
			fetching = false;
		}
	}

	async function loadLeaders() {
		loadingLeaders = true;
		try {
			const res = await getLeaders(
leaderStat,
leaderSeason,
leaderWeek,
leaderPosition || undefined,
25
);
			leaders = res.items;
		} catch (e) {
			console.error('Failed to load leaders', e);
		} finally {
			loadingLeaders = false;
		}
	}

	function applyFilters() {
		offset = 0;
		loadStats();
	}

	function clearFilters() {
		search = '';
		team = '';
		position = '';
		season = undefined;
		week = undefined;
		statType = '';
		source = '';
		sortCol = '';
		sortOrder = '';
		offset = 0;
		loadStats();
	}

	function nextPage() {
		if (offset + limit < total) {
			offset += limit;
			loadStats();
		}
	}

	function prevPage() {
		if (offset > 0) {
			offset = Math.max(0, offset - limit);
			loadStats();
		}
	}

	function fmtNum(n: number | undefined | null): string {
		if (n === undefined || n === null) return '—';
		return n.toLocaleString();
	}

	onMount(loadStats);
</script>

<div class="flex justify-between items-center mb-4">
	<h1 class="text-2xl font-bold text-primary tracking-wide">// PLAYER STATS</h1>
	<div class="flex gap-2 items-center">
		<span class="text-sm opacity-60">{total.toLocaleString()} stat lines</span>
		<button
			class="btn btn-sm"
			class:btn-primary={showLeaders}
			onclick={() => { showLeaders = !showLeaders; if (showLeaders) loadLeaders(); }}
		>
			{showLeaders ? '← Browse' : '★ Leaders'}
		</button>
	</div>
</div>

{#if showLeaders}
	<!-- Leaders Panel -->
	<div class="flex flex-wrap gap-2 mb-4 items-center">
		<select class="select select-bordered select-sm" bind:value={leaderStat} onchange={loadLeaders}>
			{#each LEADER_STATS as ls}
				<option value={ls.value}>{ls.label}</option>
			{/each}
		</select>
		<select class="select select-bordered select-sm" bind:value={leaderSeason} onchange={loadLeaders}>
			{#each SEASONS as year}
				<option value={year}>{year}</option>
			{/each}
		</select>
		<select class="select select-bordered select-sm" bind:value={leaderWeek} onchange={loadLeaders}>
			<option value={undefined}>Full Season</option>
			{#each NFL_WEEKS as w}
				<option value={w}>Week {w}</option>
			{/each}
		</select>
		<select class="select select-bordered select-sm" bind:value={leaderPosition} onchange={loadLeaders}>
			<option value="">All Pos</option>
			{#each POSITIONS as pos}
				<option value={pos.abbr}>{pos.abbr}</option>
			{/each}
		</select>
	</div>

	{#if loadingLeaders}
		<div class="card bg-base-200 shadow-md border border-base-300 p-8 text-center">
			<span class="loading loading-dots loading-md text-primary"></span>
			<p class="text-sm opacity-60 mt-2">Ranking leaders...</p>
		</div>
	{:else}
		<div class="card bg-base-100 shadow-md border border-base-300 overflow-hidden">
			<div class="table-scroll-wrap">
				<table class="table table-zebra table-pin-rows table-responsive">
					<thead>
						<tr>
							<th>#</th>
							<th>Player</th>
							<th>Team</th>
							<th>Pos</th>
							<th class="text-right">{LEADER_STATS.find(ls => ls.value === leaderStat)?.label ?? leaderStat}</th>
						</tr>
					</thead>
					<tbody>
						{#each leaders as leader, i}
							<tr class="hover">
								<td class="font-bold text-primary">{i + 1}</td>
								<td class="font-bold text-primary">{leader.player_name}</td>
								<td>{leader.team ?? '—'}</td>
								<td>{leader.position ?? '—'}</td>
								<td class="text-right font-bold text-accent">
									{fmtNum(leader[leaderStat as keyof PlayerStat] as number | undefined)}
								</td>
							</tr>
						{:else}
							<tr>
								<td colspan="5" class="text-center opacity-50 py-8">No leader data</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		</div>
	{/if}
{:else}
	<!-- Browse Panel -->
	<div class="flex flex-wrap gap-2 mb-4 items-center">
		<input
			type="text"
			placeholder="Search name..."
			class="input input-bordered input-sm w-44"
			bind:value={search}
			onkeydown={(e) => e.key === 'Enter' && applyFilters()}
		/>
		<select class="select select-bordered select-sm" bind:value={season} onchange={applyFilters}>
			<option value={undefined}>All Seasons</option>
			{#each SEASONS as year}
				<option value={year}>{year}</option>
			{/each}
		</select>
		<select class="select select-bordered select-sm" bind:value={week} onchange={applyFilters}>
			<option value={undefined}>All Weeks</option>
			{#each NFL_WEEKS as w}
				<option value={w}>Wk {w}</option>
			{/each}
		</select>
		<select class="select select-bordered select-sm" bind:value={team} onchange={applyFilters}>
			<option value="">All Teams</option>
			{#each NFL_TEAMS as t}
				<option value={t.abbr}>{t.abbr}</option>
			{/each}
		</select>
		<select class="select select-bordered select-sm" bind:value={position} onchange={applyFilters}>
			<option value="">All Pos</option>
			{#each POSITIONS as pos}
				<option value={pos.abbr}>{pos.abbr}</option>
			{/each}
		</select>
		<select class="select select-bordered select-sm" bind:value={statType} onchange={applyFilters}>
			<option value="">All Types</option>
			{#each STAT_TYPES as st}
				<option value={st.value}>{st.label}</option>
			{/each}
		</select>
		<select class="select select-bordered select-sm" bind:value={source} onchange={applyFilters}>
			<option value="">All Sources</option>
			{#each SOURCES as src}
				<option value={src}>{src}</option>
			{/each}
		</select>
		<button class="btn btn-sm" onclick={applyFilters}>Scan</button>
		<button class="btn btn-ghost btn-sm" onclick={clearFilters}>Reset</button>
	</div>

	{#if loading}
		<div class="card bg-base-200 shadow-md border border-base-300 p-8 text-center">
			<span class="loading loading-dots loading-md text-primary"></span>
			<p class="text-sm opacity-60 mt-2">Scanning stats...</p>
		</div>
	{:else}
		<div class="card bg-base-100 shadow-md border border-base-300 overflow-hidden" class:table-fetching={fetching}>
			<div class="table-scroll-wrap">
				<table class="table table-zebra table-pin-rows table-sm table-responsive">
					<thead>
						<tr>
							<th class="sortable" onclick={() => toggleSort('player_name')}>Player{sortIndicator('player_name')}</th>
							<th class="sortable" onclick={() => toggleSort('team')}>Team{sortIndicator('team')}</th>
							<th class="sortable" onclick={() => toggleSort('position')}>Pos{sortIndicator('position')}</th>
							<th class="sortable" onclick={() => toggleSort('season')}>Szn{sortIndicator('season')}</th>
							<th class="sortable" onclick={() => toggleSort('week')}>Wk{sortIndicator('week')}</th>
							<th class="sortable text-right" onclick={() => toggleSort('passing_yards')}>Pass Yd{sortIndicator('passing_yards')}</th>
							<th class="sortable text-right" onclick={() => toggleSort('passing_tds')}>Pass TD{sortIndicator('passing_tds')}</th>
							<th class="sortable text-right" onclick={() => toggleSort('rushing_yards')}>Rush Yd{sortIndicator('rushing_yards')}</th>
							<th class="sortable text-right" onclick={() => toggleSort('rushing_tds')}>Rush TD{sortIndicator('rushing_tds')}</th>
							<th class="sortable text-right" onclick={() => toggleSort('receiving_yards')}>Rec Yd{sortIndicator('receiving_yards')}</th>
							<th class="sortable text-right" onclick={() => toggleSort('receiving_tds')}>Rec TD{sortIndicator('receiving_tds')}</th>
							<th class="sortable text-right" onclick={() => toggleSort('receptions')}>Rec{sortIndicator('receptions')}</th>
							<th class="sortable text-right" onclick={() => toggleSort('fantasy_points_ppr')}>PPR{sortIndicator('fantasy_points_ppr')}</th>
						</tr>
					</thead>
					<tbody>
						{#each stats as s}
							<tr class="hover">
								<td class="font-bold text-primary">{s.player_name}</td>
								<td>{s.team ?? '—'}</td>
								<td>{s.position ?? '—'}</td>
								<td>{s.season}</td>
								<td>{s.week}</td>
								<td class="text-right">{fmtNum(s.passing_yards)}</td>
								<td class="text-right">{fmtNum(s.passing_tds)}</td>
								<td class="text-right">{fmtNum(s.rushing_yards)}</td>
								<td class="text-right">{fmtNum(s.rushing_tds)}</td>
								<td class="text-right">{fmtNum(s.receiving_yards)}</td>
								<td class="text-right">{fmtNum(s.receiving_tds)}</td>
								<td class="text-right">{fmtNum(s.receptions)}</td>
								<td class="text-right font-bold text-accent">{fmtNum(s.fantasy_points_ppr)}</td>
							</tr>
						{:else}
							<tr>
								<td colspan="13" class="text-center opacity-50 py-8">No stat records found</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		</div>

		<div class="flex justify-between items-center mt-4 text-sm opacity-70">
			<span>{offset + 1}–{Math.min(offset + limit, total)} of {total.toLocaleString()}</span>
			<div class="join">
				<button class="join-item btn btn-sm" onclick={prevPage} disabled={offset === 0}>◄ Prev</button>
				<button class="join-item btn btn-sm" onclick={nextPage} disabled={offset + limit >= total}>Next ►</button>
			</div>
		</div>
	{/if}
{/if}
