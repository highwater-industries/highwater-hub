<script lang="ts">
	import { onMount } from 'svelte';
	import { listRankings, type FantasyRank, type RankingFilter } from '$lib/api';
	import { NFL_TEAMS, POSITIONS, RANK_TYPES, SEASONS, NFL_WEEKS, SOURCES } from '$lib/constants';

	let rankings: FantasyRank[] = $state([]);
	let total = $state(0);
	let loading = $state(true);
	let fetching = $state(false);

	// Filters
	let search = $state('');
	let rankType = $state('');
	let pos = $state('');
	let team = $state('');
	let season: number | undefined = $state(undefined);
	let week: number | undefined = $state(undefined);
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
			sortOrder = 'asc';
		}
		offset = 0;
		loadRankings();
	}

	function sortIndicator(col: string): string {
		if (sortCol !== col) return '';
		return sortOrder === 'asc' ? ' ▲' : ' ▼';
	}

	async function loadRankings() {
		if (!rankings.length) loading = true;
		fetching = true;
		try {
			const filter: RankingFilter = { offset, limit };
			if (search) filter.search = search;
			if (rankType) filter.rank_type = rankType;
			if (pos) filter.pos = pos;
			if (team) filter.team = team;
			if (season !== undefined) filter.season = season;
			if (week !== undefined) filter.week = week;
			if (source) filter.source = source;
			if (sortCol) filter.sort = sortCol;
			if (sortOrder) filter.order = sortOrder;

			const res = await listRankings(filter);
			rankings = res.items;
			total = res.total;
		} catch (e) {
			console.error('Failed to load rankings', e);
		} finally {
			loading = false;
			fetching = false;
		}
	}

	function applyFilters() {
		offset = 0;
		loadRankings();
	}

	function clearFilters() {
		search = '';
		rankType = '';
		pos = '';
		team = '';
		season = undefined;
		week = undefined;
		source = '';
		sortCol = '';
		sortOrder = '';
		offset = 0;
		loadRankings();
	}

	function nextPage() {
		if (offset + limit < total) {
			offset += limit;
			loadRankings();
		}
	}

	function prevPage() {
		if (offset > 0) {
			offset = Math.max(0, offset - limit);
			loadRankings();
		}
	}

	function fmtRank(n: number | undefined | null): string {
		if (n === undefined || n === null) return '—';
		return String(n);
	}

	function fmtDec(n: number | undefined | null): string {
		if (n === undefined || n === null) return '—';
		return n.toFixed(1);
	}

	onMount(loadRankings);
</script>

<div class="page-header">
	<h1>// FANTASY RANKINGS</h1>
	<span style="font-family: var(--font-pixel); font-size: 0.55rem; color: var(--text-muted)">
		{total.toLocaleString()} RANKINGS
	</span>
</div>

<div class="filters">
	<input
		type="text"
		placeholder="SEARCH NAME..."
		bind:value={search}
		onkeydown={(e) => e.key === 'Enter' && applyFilters()}
	/>
	<select bind:value={rankType} onchange={applyFilters}>
		<option value="">ALL TYPES</option>
		{#each RANK_TYPES as rt}
			<option value={rt.value}>{rt.label}</option>
		{/each}
	</select>
	<select bind:value={pos} onchange={applyFilters}>
		<option value="">ALL POS</option>
		{#each POSITIONS as p}
			<option value={p.abbr}>{p.abbr}</option>
		{/each}
	</select>
	<select bind:value={team} onchange={applyFilters}>
		<option value="">ALL TEAMS</option>
		{#each NFL_TEAMS as t}
			<option value={t.abbr}>{t.abbr}</option>
		{/each}
	</select>
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
		<p style="font-family: var(--font-pixel); font-size: 0.6rem; color: var(--accent)">LOADING RANKINGS...</p>
	</div>
{:else}
	<div class="card" class:table-fetching={fetching} style="padding: 0; overflow-x: auto">
		<table>
			<thead>
				<tr>
					<th class="sortable" style="text-align: right" onclick={() => toggleSort('rank')}>RANK{sortIndicator('rank')}</th>
					<th class="sortable" onclick={() => toggleSort('player_name')}>PLAYER{sortIndicator('player_name')}</th>
					<th class="sortable" onclick={() => toggleSort('pos')}>POS{sortIndicator('pos')}</th>
					<th class="sortable" onclick={() => toggleSort('team')}>TEAM{sortIndicator('team')}</th>
					<th class="sortable" style="text-align: right" onclick={() => toggleSort('ecr')}>ECR{sortIndicator('ecr')}</th>
					<th class="sortable" style="text-align: right" onclick={() => toggleSort('sd')}>SD{sortIndicator('sd')}</th>
					<th class="sortable" style="text-align: right" onclick={() => toggleSort('best')}>BEST{sortIndicator('best')}</th>
					<th class="sortable" style="text-align: right" onclick={() => toggleSort('worst')}>WORST{sortIndicator('worst')}</th>
					<th class="sortable" style="text-align: right" onclick={() => toggleSort('avg')}>AVG{sortIndicator('avg')}</th>
					<th>TYPE</th>
				</tr>
			</thead>
			<tbody>
				{#each rankings as r}
					<tr>
						<td style="text-align: right; font-family: var(--font-pixel); font-size: 0.55rem; color: var(--accent)">
							{fmtRank(r.rank)}
						</td>
						<td><strong style="color: var(--accent)">{r.player_name}</strong></td>
						<td>{r.pos ?? '—'}</td>
						<td>{r.team ?? '—'}</td>
						<td style="text-align: right">{fmtDec(r.ecr)}</td>
						<td style="text-align: right">{fmtDec(r.sd)}</td>
						<td style="text-align: right">{fmtRank(r.best)}</td>
						<td style="text-align: right">{fmtRank(r.worst)}</td>
						<td style="text-align: right">{fmtDec(r.avg)}</td>
						<td>
							<span class="badge">{r.rank_type ?? '—'}</span>
						</td>
					</tr>
				{:else}
					<tr>
						<td colspan="10" style="text-align: center; color: var(--text-muted); padding: 2rem; font-family: var(--font-pixel); font-size: 0.55rem">
							NO RANKING DATA
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
