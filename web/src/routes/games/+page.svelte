<script lang="ts">
	import { onMount } from 'svelte';
	import { listGames, type GameData, type GameFilter } from '$lib/api';
	import { NFL_TEAMS, SEASONS, NFL_WEEKS } from '$lib/constants';

	let games: GameData[] = $state([]);
	let total = $state(0);
	let loading = $state(true);
	let fetching = $state(false);

	// Filters
	let season: number | undefined = $state(undefined);
	let week: number | undefined = $state(undefined);
	let team = $state('');
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
		loadGames();
	}

	function sortIndicator(col: string): string {
		if (sortCol !== col) return '';
		return sortOrder === 'asc' ? ' ▲' : ' ▼';
	}

	async function loadGames() {
		if (!games.length) loading = true;
		fetching = true;
		try {
			const filter: GameFilter = { offset, limit };
			if (season !== undefined) filter.season = season;
			if (week !== undefined) filter.week = week;
			if (team) filter.team = team;
			if (sortCol) filter.sort = sortCol;
			if (sortOrder) filter.order = sortOrder;

			const res = await listGames(filter);
			games = res.items;
			total = res.total;
		} catch (e) {
			console.error('Failed to load games', e);
		} finally {
			loading = false;
			fetching = false;
		}
	}

	function applyFilters() {
		offset = 0;
		loadGames();
	}

	function clearFilters() {
		season = undefined;
		week = undefined;
		team = '';
		sortCol = '';
		sortOrder = '';
		offset = 0;
		loadGames();
	}

	function nextPage() {
		if (offset + limit < total) {
			offset += limit;
			loadGames();
		}
	}

	function prevPage() {
		if (offset > 0) {
			offset = Math.max(0, offset - limit);
			loadGames();
		}
	}

	function formatScore(g: GameData): string {
		if (g.away_score === undefined || g.home_score === undefined) return '—';
		return `${g.away_score} – ${g.home_score}`;
	}

	onMount(loadGames);
</script>

<div class="page-header">
	<h1>// GAMES</h1>
	<span style="font-family: var(--font-pixel); font-size: 0.55rem; color: var(--text-muted)">
		{total.toLocaleString()} GAMES
	</span>
</div>

<div class="filters">
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
			<option value={t.abbr}>{t.abbr} — {t.name}</option>
		{/each}
	</select>
	<button onclick={applyFilters}>SCAN</button>
	<button onclick={clearFilters}>RESET</button>
</div>

{#if loading}
	<div class="card" style="text-align: center; padding: 2rem">
		<p style="font-family: var(--font-pixel); font-size: 0.6rem; color: var(--accent)">LOADING SCHEDULE...</p>
	</div>
{:else}
	<div class="card" class:table-fetching={fetching} style="padding: 0; overflow-x: auto">
		<table>
			<thead>
				<tr>
					<th class="sortable" onclick={() => toggleSort('gameday')}>DATE{sortIndicator('gameday')}</th>
					<th class="sortable" onclick={() => toggleSort('season')}>SZN{sortIndicator('season')}</th>
					<th class="sortable" onclick={() => toggleSort('week')}>WK{sortIndicator('week')}</th>
					<th>MATCHUP</th>
					<th class="sortable" style="text-align: center" onclick={() => toggleSort('home_score')}>SCORE{sortIndicator('home_score')}</th>
					<th>TYPE</th>
					<th>OT</th>
					<th>STADIUM</th>
				</tr>
			</thead>
			<tbody>
				{#each games as g}
					<tr>
						<td style="white-space: nowrap">{g.gameday ?? '—'}</td>
						<td>{g.season ?? '—'}</td>
						<td>{g.week ?? '—'}</td>
						<td>
							<strong style="color: var(--accent)">{g.away_team ?? '?'}</strong>
							<span style="color: var(--text-muted)">&nbsp;@&nbsp;</span>
							<strong>{g.home_team ?? '?'}</strong>
						</td>
						<td style="text-align: center; font-family: var(--font-pixel); font-size: 0.55rem; color: var(--accent)">
							{formatScore(g)}
						</td>
						<td>{g.game_type ?? '—'}</td>
						<td>{g.overtime ? '⚡' : ''}</td>
						<td style="color: var(--text-muted); font-size: 0.9rem">{g.stadium ?? '—'}</td>
					</tr>
				{:else}
					<tr>
						<td colspan="8" style="text-align: center; color: var(--text-muted); padding: 2rem; font-family: var(--font-pixel); font-size: 0.55rem">
							NO GAMES FOUND
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
