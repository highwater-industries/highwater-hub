<script lang="ts">
	import { onMount } from 'svelte';
	import { listPlayers, type Player, type PlayerFilter } from '$lib/api';
	import { NFL_TEAMS, POSITIONS, SOURCES } from '$lib/constants';

	let players: Player[] = $state([]);
	let total = $state(0);
	let loading = $state(true);
	let fetching = $state(false);

	// Filters
	let search = $state('');
	let team = $state('');
	let position = $state('');
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
		loadPlayers();
	}

	function sortIndicator(col: string): string {
		if (sortCol !== col) return '';
		return sortOrder === 'asc' ? ' ▲' : ' ▼';
	}

	async function loadPlayers() {
		if (!players.length) loading = true;
		fetching = true;
		try {
			const filter: PlayerFilter = { offset, limit };
			if (search) filter.search = search;
			if (team) filter.team = team;
			if (position) filter.position = position;
			if (source) filter.source = source;
			if (sortCol) filter.sort = sortCol;
			if (sortOrder) filter.order = sortOrder;

			const res = await listPlayers(filter);
			players = res.items;
			total = res.total;
		} catch (e) {
			console.error('Failed to load players', e);
		} finally {
			loading = false;
			fetching = false;
		}
	}

	function applyFilters() {
		offset = 0;
		loadPlayers();
	}

	function clearFilters() {
		search = '';
		team = '';
		position = '';
		source = '';
		sortCol = '';
		sortOrder = '';
		offset = 0;
		loadPlayers();
	}

	function nextPage() {
		if (offset + limit < total) {
			offset += limit;
			loadPlayers();
		}
	}

	function prevPage() {
		if (offset > 0) {
			offset = Math.max(0, offset - limit);
			loadPlayers();
		}
	}

	onMount(loadPlayers);
</script>

<div class="flex justify-between items-center mb-4">
	<h1 class="text-2xl font-bold text-primary tracking-wide">// PLAYERS</h1>
	<span class="text-sm opacity-60">{total.toLocaleString()} IN ROSTER</span>
</div>

<div class="flex flex-wrap gap-2 mb-4 items-center">
	<input
		type="text"
		placeholder="Search name..."
		class="input input-bordered input-sm w-48"
		bind:value={search}
		onkeydown={(e) => e.key === 'Enter' && applyFilters()}
	/>
	<select class="select select-bordered select-sm" bind:value={team} onchange={applyFilters}>
		<option value="">All Teams</option>
		{#each NFL_TEAMS as t}
			<option value={t.abbr}>{t.abbr} — {t.name}</option>
		{/each}
	</select>
	<select class="select select-bordered select-sm" bind:value={position} onchange={applyFilters}>
		<option value="">All Pos</option>
		{#each POSITIONS as pos}
			<option value={pos.abbr}>{pos.abbr} — {pos.name}</option>
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
		<p class="text-sm opacity-60 mt-2">Scanning database...</p>
	</div>
{:else}
	<div class="card bg-base-100 shadow-md border border-base-300 overflow-hidden" class:table-fetching={fetching}>
		<div class="table-scroll-wrap">
			<table class="table table-zebra table-pin-rows table-responsive">
				<thead>
					<tr>
						<th class="sortable" onclick={() => toggleSort('player_name')}>Name{sortIndicator('player_name')}</th>
						<th class="sortable" onclick={() => toggleSort('team')}>Team{sortIndicator('team')}</th>
						<th class="sortable" onclick={() => toggleSort('player_position')}>Pos{sortIndicator('player_position')}</th>
						<th>College</th>
						<th>Status</th>
						<th>#</th>
					</tr>
				</thead>
				<tbody>
					{#each players as player}
						<tr class="hover">
							<td>
								<div class="flex items-center gap-3">
									{#if player.metadata?.headshot_url}
										<div class="avatar">
											<div class="w-10 h-10 rounded-full bg-base-300">
												<img src={player.metadata.headshot_url} alt={player.player_name} loading="lazy" />
											</div>
										</div>
									{:else}
										<div class="avatar placeholder">
											<div class="w-10 h-10 rounded-full bg-base-300 text-base-content/40">
												<span class="text-xs">{player.player_name.split(' ').map((n: string) => n[0]).join('').slice(0, 2)}</span>
											</div>
										</div>
									{/if}
									<div>
										<div class="font-bold text-primary">{player.player_name}</div>
										<div class="text-xs opacity-40">{player.player_id}</div>
									</div>
								</div>
							</td>
							<td>{player.team}</td>
							<td>{player.player_position}</td>
							<td>{player.metadata?.college ?? '—'}</td>
							<td>
								{#if player.metadata?.status === 'ACT'}
									<span class="badge badge-success badge-sm">{player.metadata.status}</span>
								{:else}
									<span class="badge badge-ghost badge-sm">{player.metadata?.status ?? '—'}</span>
								{/if}
							</td>
							<td>{player.metadata?.jersey_number ?? '—'}</td>
						</tr>
					{:else}
						<tr>
							<td colspan="6" class="text-center opacity-50 py-8">No records found</td>
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
