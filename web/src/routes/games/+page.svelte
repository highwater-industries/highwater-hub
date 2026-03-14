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

<div class="flex justify-between items-center mb-4">
	<h1 class="text-2xl font-bold text-primary tracking-wide">// GAMES</h1>
	<span class="text-sm opacity-60">{total.toLocaleString()} games</span>
</div>

<div class="flex flex-wrap gap-2 mb-4 items-center">
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
			<option value={t.abbr}>{t.abbr} — {t.name}</option>
		{/each}
	</select>
	<button class="btn btn-sm" onclick={applyFilters}>Scan</button>
	<button class="btn btn-ghost btn-sm" onclick={clearFilters}>Reset</button>
</div>

{#if loading}
	<div class="card bg-base-200 shadow-md border border-base-300 p-8 text-center">
		<span class="loading loading-dots loading-md text-primary"></span>
		<p class="text-sm opacity-60 mt-2">Loading schedule...</p>
	</div>
{:else}
	<div class="card bg-base-100 shadow-md border border-base-300 overflow-hidden" class:table-fetching={fetching}>
		<div class="table-scroll-wrap">
			<table class="table table-zebra table-pin-rows table-responsive">
				<thead>
					<tr>
						<th class="sortable" onclick={() => toggleSort('gameday')}>Date{sortIndicator('gameday')}</th>
						<th class="sortable" onclick={() => toggleSort('season')}>Szn{sortIndicator('season')}</th>
						<th class="sortable" onclick={() => toggleSort('week')}>Wk{sortIndicator('week')}</th>
						<th>Matchup</th>
						<th class="sortable text-center" onclick={() => toggleSort('home_score')}>Score{sortIndicator('home_score')}</th>
						<th>Type</th>
						<th>OT</th>
						<th>Stadium</th>
					</tr>
				</thead>
				<tbody>
					{#each games as g}
						<tr class="hover">
							<td class="whitespace-nowrap">{g.gameday ?? '—'}</td>
							<td>{g.season ?? '—'}</td>
							<td>{g.week ?? '—'}</td>
							<td>
								<span class="font-bold text-primary">{g.away_team ?? '?'}</span>
								<span class="opacity-40"> @ </span>
								<span class="font-bold">{g.home_team ?? '?'}</span>
							</td>
							<td class="text-center font-bold text-accent">{formatScore(g)}</td>
							<td>{g.game_type ?? '—'}</td>
							<td>{g.overtime ? '⚡' : ''}</td>
							<td class="opacity-60 text-sm">{g.stadium ?? '—'}</td>
						</tr>
					{:else}
						<tr>
							<td colspan="8" class="text-center opacity-50 py-8">No games found</td>
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
