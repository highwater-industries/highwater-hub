<script lang="ts">
	import { onMount } from 'svelte';
	import { listPlayers, type Player, type PlayerFilter } from '$lib/api';
	import { NFL_TEAMS, POSITIONS, SOURCES } from '$lib/constants';

	let players: Player[] = $state([]);
	let total = $state(0);
	let loading = $state(true);

	// Filters
	let search = $state('');
	let team = $state('');
	let position = $state('');
	let source = $state('');
	let offset = $state(0);
	const limit = 25;

	async function loadPlayers() {
		loading = true;
		try {
			const filter: PlayerFilter = { offset, limit };
			if (search) filter.search = search;
			if (team) filter.team = team;
			if (position) filter.position = position;
			if (source) filter.source = source;

			const res = await listPlayers(filter);
			players = res.items;
			total = res.total;
		} catch (e) {
			console.error('Failed to load players', e);
		} finally {
			loading = false;
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

<div class="page-header">
	<h1>// PLAYERS</h1>
	<span style="font-family: var(--font-pixel); font-size: 0.45rem; color: var(--text-muted)">
		{total.toLocaleString()} IN ROSTER
	</span>
</div>

<div class="filters">
	<input
		type="text"
		placeholder="SEARCH NAME..."
		bind:value={search}
		onkeydown={(e) => e.key === 'Enter' && applyFilters()}
	/>
	<select bind:value={team} onchange={applyFilters}>
		<option value="">ALL TEAMS</option>
		{#each NFL_TEAMS as t}
			<option value={t.abbr}>{t.abbr} — {t.name}</option>
		{/each}
	</select>
	<select bind:value={position} onchange={applyFilters}>
		<option value="">ALL POS</option>
		{#each POSITIONS as pos}
			<option value={pos.abbr}>{pos.abbr} — {pos.name}</option>
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
		<p style="font-family: var(--font-pixel); font-size: 0.5rem; color: var(--accent)">
			SCANNING DATABASE...
		</p>
	</div>
{:else}
	<div class="card" style="padding: 0; overflow: hidden">
		<table>
			<thead>
				<tr>
					<th>NAME</th>
					<th>TEAM</th>
					<th>POS</th>
					<th>COLLEGE</th>
					<th>STATUS</th>
					<th>#</th>
				</tr>
			</thead>
			<tbody>
				{#each players as player}
					<tr>
						<td>
							<strong style="color: var(--accent)">{player.player_name}</strong>
							<div style="font-size: 0.85rem; color: var(--text-muted)">{player.player_id}</div>
						</td>
						<td>{player.team}</td>
						<td>{player.player_position}</td>
						<td>{player.metadata?.college ?? '—'}</td>
						<td>
							<span class="badge" class:success={player.metadata?.status === 'ACT'}>
								{player.metadata?.status ?? '—'}
							</span>
						</td>
						<td>{player.metadata?.jersey_number ?? '—'}</td>
					</tr>
				{:else}
					<tr>
						<td colspan="6" style="text-align: center; color: var(--text-muted); padding: 2rem; font-family: var(--font-pixel); font-size: 0.45rem">
							NO RECORDS FOUND
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
