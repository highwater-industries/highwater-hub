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

<div class="page-header">
	<h1>// PLAYER STATS</h1>
	<div style="display: flex; gap: 0.5rem; align-items: center">
		<span style="font-family: var(--font-pixel); font-size: 0.55rem; color: var(--text-muted)">
			{total.toLocaleString()} STAT LINES
		</span>
		<button class:primary={showLeaders} onclick={() => { showLeaders = !showLeaders; if (showLeaders) loadLeaders(); }}>
			{showLeaders ? '← BROWSE' : '★ LEADERS'}
		</button>
	</div>
</div>

{#if showLeaders}
	<!-- Leaders Panel -->
	<div class="filters">
		<select bind:value={leaderStat} onchange={loadLeaders}>
			{#each LEADER_STATS as ls}
				<option value={ls.value}>{ls.label}</option>
			{/each}
		</select>
		<select bind:value={leaderSeason} onchange={loadLeaders}>
			{#each SEASONS as year}
				<option value={year}>{year}</option>
			{/each}
		</select>
		<select bind:value={leaderWeek} onchange={loadLeaders}>
			<option value={undefined}>FULL SEASON</option>
			{#each NFL_WEEKS as w}
				<option value={w}>WEEK {w}</option>
			{/each}
		</select>
		<select bind:value={leaderPosition} onchange={loadLeaders}>
			<option value="">ALL POS</option>
			{#each POSITIONS as pos}
				<option value={pos.abbr}>{pos.abbr}</option>
			{/each}
		</select>
	</div>

	{#if loadingLeaders}
		<div class="card" style="text-align: center; padding: 2rem">
			<p style="font-family: var(--font-pixel); font-size: 0.6rem; color: var(--accent)">RANKING LEADERS...</p>
		</div>
	{:else}
		<div class="card" style="padding: 0; overflow: hidden">
			<table>
				<thead>
					<tr>
						<th>#</th>
						<th>PLAYER</th>
						<th>TEAM</th>
						<th>POS</th>
						<th style="text-align: right">{LEADER_STATS.find(ls => ls.value === leaderStat)?.label ?? leaderStat}</th>
					</tr>
				</thead>
				<tbody>
					{#each leaders as leader, i}
						<tr>
							<td style="color: var(--accent); font-family: var(--font-pixel); font-size: 0.55rem">{i + 1}</td>
							<td><strong style="color: var(--accent)">{leader.player_name}</strong></td>
							<td>{leader.team ?? '—'}</td>
							<td>{leader.position ?? '—'}</td>
							<td style="text-align: right; font-family: var(--font-pixel); font-size: 0.55rem; color: var(--accent)">
								{fmtNum(leader[leaderStat as keyof PlayerStat] as number | undefined)}
							</td>
						</tr>
					{:else}
						<tr>
							<td colspan="5" style="text-align: center; color: var(--text-muted); padding: 2rem; font-family: var(--font-pixel); font-size: 0.55rem">
								NO LEADER DATA
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	{/if}
{:else}
	<!-- Browse Panel -->
	<div class="filters">
		<input
			type="text"
			placeholder="SEARCH NAME..."
			bind:value={search}
			onkeydown={(e) => e.key === 'Enter' && applyFilters()}
		/>
		<select bind:value={season} onchange={applyFilters}>
			<option value={undefined}>ALL SEASONS</option>
			{#each SEASONS as year}
				<option value={year}>{year}</option>
			{/each}
		</select>
		<select bind:value={week} onchange={applyFilters}>
			<option value={undefined}>ALL WEEKS</option>
			{#each NFL_WEEKS as w}
				<option value={w}>WK {w}</option>
			{/each}
		</select>
		<select bind:value={team} onchange={applyFilters}>
			<option value="">ALL TEAMS</option>
			{#each NFL_TEAMS as t}
				<option value={t.abbr}>{t.abbr}</option>
			{/each}
		</select>
		<select bind:value={position} onchange={applyFilters}>
			<option value="">ALL POS</option>
			{#each POSITIONS as pos}
				<option value={pos.abbr}>{pos.abbr}</option>
			{/each}
		</select>
		<select bind:value={statType} onchange={applyFilters}>
			<option value="">ALL TYPES</option>
			{#each STAT_TYPES as st}
				<option value={st.value}>{st.label}</option>
			{/each}
		</select>
		<select bind:value={source} onchange={applyFilters}>
			<option value="">ALL SOURCES</option>
			{#each SOURCES as src}
				<option value={src}>{src}</option>
			{/each}
		</select>
		<button onclick={applyFilters}>SCAN</button>
		<button onclick={clearFilters}>RESET</button>
	</div>

	{#if loading}
		<div class="card" style="text-align: center; padding: 2rem">
			<p style="font-family: var(--font-pixel); font-size: 0.6rem; color: var(--accent)">SCANNING STATS...</p>
		</div>
	{:else}
		<div class="card" class:table-fetching={fetching} style="padding: 0; overflow-x: auto">
			<table>
				<thead>
					<tr>
						<th class="sortable" onclick={() => toggleSort('player_name')}>PLAYER{sortIndicator('player_name')}</th>
						<th class="sortable" onclick={() => toggleSort('team')}>TEAM{sortIndicator('team')}</th>
						<th class="sortable" onclick={() => toggleSort('position')}>POS{sortIndicator('position')}</th>
						<th class="sortable" onclick={() => toggleSort('season')}>SZN{sortIndicator('season')}</th>
						<th class="sortable" onclick={() => toggleSort('week')}>WK{sortIndicator('week')}</th>
						<th class="sortable" style="text-align: right" onclick={() => toggleSort('passing_yards')}>PASS YD{sortIndicator('passing_yards')}</th>
						<th class="sortable" style="text-align: right" onclick={() => toggleSort('passing_tds')}>PASS TD{sortIndicator('passing_tds')}</th>
						<th class="sortable" style="text-align: right" onclick={() => toggleSort('rushing_yards')}>RUSH YD{sortIndicator('rushing_yards')}</th>
						<th class="sortable" style="text-align: right" onclick={() => toggleSort('rushing_tds')}>RUSH TD{sortIndicator('rushing_tds')}</th>
						<th class="sortable" style="text-align: right" onclick={() => toggleSort('receiving_yards')}>REC YD{sortIndicator('receiving_yards')}</th>
						<th class="sortable" style="text-align: right" onclick={() => toggleSort('receiving_tds')}>REC TD{sortIndicator('receiving_tds')}</th>
						<th class="sortable" style="text-align: right" onclick={() => toggleSort('receptions')}>REC{sortIndicator('receptions')}</th>
						<th class="sortable" style="text-align: right" onclick={() => toggleSort('fantasy_points_ppr')}>PPR{sortIndicator('fantasy_points_ppr')}</th>
					</tr>
				</thead>
				<tbody>
					{#each stats as s}
						<tr>
							<td><strong style="color: var(--accent)">{s.player_name}</strong></td>
							<td>{s.team ?? '—'}</td>
							<td>{s.position ?? '—'}</td>
							<td>{s.season}</td>
							<td>{s.week}</td>
							<td style="text-align: right">{fmtNum(s.passing_yards)}</td>
							<td style="text-align: right">{fmtNum(s.passing_tds)}</td>
							<td style="text-align: right">{fmtNum(s.rushing_yards)}</td>
							<td style="text-align: right">{fmtNum(s.rushing_tds)}</td>
							<td style="text-align: right">{fmtNum(s.receiving_yards)}</td>
							<td style="text-align: right">{fmtNum(s.receiving_tds)}</td>
							<td style="text-align: right">{fmtNum(s.receptions)}</td>
							<td style="text-align: right; color: var(--accent)">{fmtNum(s.fantasy_points_ppr)}</td>
						</tr>
					{:else}
						<tr>
							<td colspan="13" style="text-align: center; color: var(--text-muted); padding: 2rem; font-family: var(--font-pixel); font-size: 0.55rem">
								NO STAT RECORDS FOUND
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>

		<div class="pagination">
			<span>{offset + 1}–{Math.min(offset + limit, total)} OF {total.toLocaleString()}</span>
			<div style="display: flex; gap: 0.5rem">
				<button onclick={prevPage} disabled={offset === 0}>◄ PREV</button>
				<button onclick={nextPage} disabled={offset + limit >= total}>NEXT ►</button>
			</div>
		</div>
	{/if}
{/if}
