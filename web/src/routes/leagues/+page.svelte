<script lang="ts">
	import { onMount } from 'svelte';
	import PageHeader from '$lib/components/PageHeader.svelte';
	import {
		listFantasyLeagues,
		type FantasyLeague,
		type FantasyLeagueFilter
	} from '$lib/api';

	let leagues: FantasyLeague[] = $state([]);
	let total = $state(0);
	let loading = $state(true);
	let offset = $state(0);
	const limit = 20;

	// Filters
	let filterPlatform = $state('');
	let filterSeason: number | undefined = $state(undefined);

	async function loadLeagues() {
		loading = true;
		try {
			const filter: FantasyLeagueFilter = { offset, limit };
			if (filterPlatform) filter.platform = filterPlatform;
			if (filterSeason !== undefined) filter.season = filterSeason;
			const res = await listFantasyLeagues(filter);
			leagues = res.items;
			total = res.total;
		} catch (e) {
			console.error('Failed to load leagues', e);
		} finally {
			loading = false;
		}
	}

	function applyFilters() {
		offset = 0;
		loadLeagues();
	}

	function clearFilters() {
		filterPlatform = '';
		filterSeason = undefined;
		offset = 0;
		loadLeagues();
	}

	function nextPage() {
		if (offset + limit < total) {
			offset += limit;
			loadLeagues();
		}
	}
	function prevPage() {
		if (offset > 0) {
			offset = Math.max(0, offset - limit);
			loadLeagues();
		}
	}

	function platformBadge(platform: string): string {
		switch (platform) {
			case 'yahoo':
				return 'badge-primary';
			case 'espn':
				return 'badge-error';
			default:
				return 'badge-ghost';
		}
	}

	const seasons = Array.from({ length: 10 }, (_, i) => new Date().getFullYear() - i);

	onMount(loadLeagues);
</script>

<PageHeader
	title="Fantasy Leagues"
	breadcrumbs={[{ label: 'Fantasy', href: '/leagues' }, { label: 'Leagues' }]}
>
	{#snippet actions()}
		<span class="text-sm text-base-content/60">{total} league{total !== 1 ? 's' : ''}</span>
	{/snippet}
</PageHeader>

<!-- Filters -->
<div class="flex flex-wrap gap-2 mb-4 items-center">
	<select class="select select-bordered select-sm" bind:value={filterPlatform} onchange={applyFilters}>
		<option value="">All Platforms</option>
		<option value="yahoo">Yahoo</option>
		<option value="espn">ESPN</option>
	</select>
	<select class="select select-bordered select-sm" bind:value={filterSeason} onchange={applyFilters}>
		<option value={undefined}>All Seasons</option>
		{#each seasons as y}
			<option value={y}>{y}</option>
		{/each}
	</select>
	<button class="btn btn-ghost btn-sm" onclick={clearFilters}>Reset</button>
</div>

<!-- League Cards -->
{#if loading}
	<div class="card bg-base-100 shadow-sm p-8 text-center">
		<span class="loading loading-dots loading-md text-primary"></span>
		<p class="text-sm text-base-content/60 mt-2">Loading leagues...</p>
	</div>
{:else if leagues.length === 0}
	<div class="card bg-base-100 shadow-sm p-8 text-center">
		<p class="text-base-content/50">No leagues imported yet.</p>
		<p class="text-xs text-base-content/40 mt-1">Use Data Management → Import → Fantasy to import a league.</p>
	</div>
{:else}
	<div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
		{#each leagues as league}
			<a href="/leagues/{league.id}" class="card bg-base-100 shadow-sm hover:shadow-md transition-shadow cursor-pointer">
				<div class="card-body p-4">
					<div class="flex items-center justify-between mb-1">
						<span class="badge {platformBadge(league.platform)} badge-sm uppercase">{league.platform}</span>
						<span class="text-sm font-semibold text-base-content/80">{league.season}</span>
					</div>
					<h3 class="card-title text-base">{league.league_name}</h3>
					<div class="flex gap-4 text-sm text-base-content/70 mt-1">
						{#if league.num_teams}
							<span>🏈 {league.num_teams} teams</span>
						{/if}
						{#if league.scoring_type}
							<span class="capitalize">{league.scoring_type.replaceAll('_', ' ')}</span>
						{/if}
					</div>
					<div class="text-xs text-base-content/40 mt-2">
						Updated {new Date(league.updated_at).toLocaleDateString()}
					</div>
				</div>
			</a>
		{/each}
	</div>

	<!-- Pagination -->
	{#if total > limit}
		<div class="flex justify-between items-center mt-4 text-sm opacity-70">
			<span>{offset + 1}–{Math.min(offset + limit, total)} of {total}</span>
			<div class="join">
				<button class="join-item btn btn-sm" onclick={prevPage} disabled={offset === 0}>◄ Prev</button>
				<button class="join-item btn btn-sm" onclick={nextPage} disabled={offset + limit >= total}>Next ►</button>
			</div>
		</div>
	{/if}
{/if}
