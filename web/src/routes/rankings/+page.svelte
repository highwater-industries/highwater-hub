<script lang="ts">
	import { onMount } from 'svelte';
	import PageHeader from '$lib/components/PageHeader.svelte';
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

<PageHeader title="Fantasy Rankings" breadcrumbs={[{ label: 'NFL', href: '/rankings' }, { label: 'Rankings' }]}>
	{#snippet actions()}
		<span class="text-sm text-base-content/60">{total.toLocaleString()} rankings</span>
	{/snippet}
</PageHeader>

<div class="flex flex-wrap gap-2 mb-4 items-center">
	<input
		type="text"
		placeholder="Search name..."
		class="input input-bordered input-sm w-44"
		bind:value={search}
		onkeydown={(e) => e.key === 'Enter' && applyFilters()}
	/>
	<select class="select select-bordered select-sm" bind:value={rankType} onchange={applyFilters}>
		<option value="">All Types</option>
		{#each RANK_TYPES as rt}
			<option value={rt.value}>{rt.label}</option>
		{/each}
	</select>
	<select class="select select-bordered select-sm" bind:value={pos} onchange={applyFilters}>
		<option value="">All Pos</option>
		{#each POSITIONS as p}
			<option value={p.abbr}>{p.abbr}</option>
		{/each}
	</select>
	<select class="select select-bordered select-sm" bind:value={team} onchange={applyFilters}>
		<option value="">All Teams</option>
		{#each NFL_TEAMS as t}
			<option value={t.abbr}>{t.abbr}</option>
		{/each}
	</select>
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
	<div class="card bg-base-100 shadow-sm p-8 text-center">
		<span class="loading loading-dots loading-md text-primary"></span>
		<p class="text-sm text-base-content/60 mt-2">Loading rankings...</p>
	</div>
{:else}
	<div class="card bg-base-100 shadow-sm overflow-hidden" class:table-fetching={fetching}>
		<div class="table-scroll-wrap">
			<table class="table table-zebra table-pin-rows table-responsive">
				<thead>
					<tr>
						<th class="sortable text-right" onclick={() => toggleSort('rank')}>Rank{sortIndicator('rank')}</th>
						<th class="sortable" onclick={() => toggleSort('player_name')}>Player{sortIndicator('player_name')}</th>
						<th class="sortable" onclick={() => toggleSort('pos')}>Pos{sortIndicator('pos')}</th>
						<th class="sortable" onclick={() => toggleSort('team')}>Team{sortIndicator('team')}</th>
						<th class="sortable text-right" onclick={() => toggleSort('ecr')}>ECR{sortIndicator('ecr')}</th>
						<th class="sortable text-right" onclick={() => toggleSort('sd')}>SD{sortIndicator('sd')}</th>
						<th class="sortable text-right" onclick={() => toggleSort('best')}>Best{sortIndicator('best')}</th>
						<th class="sortable text-right" onclick={() => toggleSort('worst')}>Worst{sortIndicator('worst')}</th>
						<th class="sortable text-right" onclick={() => toggleSort('avg')}>Avg{sortIndicator('avg')}</th>
						<th>Type</th>
					</tr>
				</thead>
				<tbody>
					{#each rankings as r}
						<tr class="hover">
							<td class="text-right font-bold text-accent">{fmtRank(r.rank)}</td>
							<td class="font-bold text-primary">{#if r.player_db_id}<a href="/players/{r.player_db_id}" class="hover:underline">{r.player_name}</a>{:else}{r.player_name}{/if}</td>
							<td>{r.pos ?? '—'}</td>
							<td>{r.team ?? '—'}</td>
							<td class="text-right">{fmtDec(r.ecr)}</td>
							<td class="text-right">{fmtDec(r.sd)}</td>
							<td class="text-right">{fmtRank(r.best)}</td>
							<td class="text-right">{fmtRank(r.worst)}</td>
							<td class="text-right">{fmtDec(r.avg)}</td>
							<td><span class="badge badge-ghost badge-sm">{r.rank_type ?? '—'}</span></td>
						</tr>
					{:else}
						<tr>
							<td colspan="10" class="text-center text-base-content/50 py-8">No ranking data</td>
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
