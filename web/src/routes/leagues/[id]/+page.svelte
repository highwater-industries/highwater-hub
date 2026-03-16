<script lang="ts">
	import { page } from '$app/stores';
	import { onMount } from 'svelte';
	import PageHeader from '$lib/components/PageHeader.svelte';
	import {
		getFantasyLeague,
		getFantasyTeam,
		type FantasyTeam,
		type FantasyRosterEntry,
		type LeagueDetail
	} from '$lib/api';

	let leagueDetail: LeagueDetail | null = $state(null);
	let loading = $state(true);
	let error = $state('');

	// Expanded team roster
	let expandedTeamId: number | null = $state(null);
	let roster: FantasyRosterEntry[] = $state([]);
	let rosterLoading = $state(false);

	let leagueId: number = $derived(Number($page.params.id));

	async function loadLeague() {
		loading = true;
		error = '';
		try {
			leagueDetail = await getFantasyLeague(leagueId);
		} catch (e: any) {
			error = e.message;
		} finally {
			loading = false;
		}
	}

	async function toggleRoster(team: FantasyTeam) {
		if (expandedTeamId === team.id) {
			expandedTeamId = null;
			roster = [];
			return;
		}
		expandedTeamId = team.id;
		rosterLoading = true;
		try {
			const detail = await getFantasyTeam(team.id);
			roster = detail.roster;
		} catch (e) {
			console.error('Failed to load roster', e);
			roster = [];
		} finally {
			rosterLoading = false;
		}
	}

	function fmtRecord(t: FantasyTeam): string {
		let record = `${t.wins}-${t.losses}`;
		if (t.ties > 0) record += `-${t.ties}`;
		return record;
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

	onMount(loadLeague);
</script>

{#if loading}
	<PageHeader title="Loading..." breadcrumbs={[{ label: 'Fantasy', href: '/leagues' }, { label: 'Leagues', href: '/leagues' }, { label: '...' }]} />
	<div class="card bg-base-100 shadow-sm p-8 text-center">
		<span class="loading loading-dots loading-md text-primary"></span>
	</div>
{:else if error}
	<PageHeader title="Error" breadcrumbs={[{ label: 'Fantasy', href: '/leagues' }, { label: 'Leagues', href: '/leagues' }]} />
	<div class="alert alert-error">{error}</div>
{:else if leagueDetail}
	{@const league = leagueDetail.league}
	{@const teams = leagueDetail.teams}

	<PageHeader
		title={league.league_name}
		breadcrumbs={[{ label: 'Fantasy', href: '/leagues' }, { label: 'Leagues', href: '/leagues' }, { label: league.league_name }]}
	>
		{#snippet actions()}
			<span class="badge {platformBadge(league.platform)} uppercase">{league.platform}</span>
			<span class="text-sm text-base-content/60">{league.season}</span>
		{/snippet}
	</PageHeader>

	<!-- League info card -->
	<div class="card bg-base-100 shadow-sm mb-6 p-4">
		<div class="flex flex-wrap gap-6 text-sm">
			<div><span class="font-semibold">Teams:</span> {league.num_teams ?? teams.length}</div>
			{#if league.scoring_type}
				<div><span class="font-semibold">Scoring:</span> <span class="capitalize">{league.scoring_type.replaceAll('_', ' ')}</span></div>
			{/if}
			<div><span class="font-semibold">External ID:</span> {league.external_league_id}</div>
			<div><span class="font-semibold">Last Sync:</span> {new Date(league.updated_at).toLocaleString()}</div>
		</div>
	</div>

	<!-- Standings table -->
	<h2 class="text-lg font-bold mb-3">Standings</h2>
	<div class="card bg-base-100 shadow-sm overflow-hidden mb-6">
		<div class="overflow-x-auto">
		<table class="table table-zebra table-pin-rows">
			<thead>
				<tr>
					<th class="w-12 text-right">#</th>
					<th>Team</th>
					<th>Owner</th>
					<th class="text-center">Record</th>
					<th class="text-right">PF</th>
					<th class="text-right">PA</th>
					<th class="text-center">Streak</th>
					<th class="w-20"></th>
				</tr>
			</thead>
			<tbody>
				{#each teams as team, i}
					<tr class="hover">
						<td class="text-right font-bold text-accent">{team.standing_rank ?? i + 1}</td>
						<td>
							<div class="flex items-center gap-2">
								{#if team.logo_url}
									<img src={team.logo_url} alt="" class="w-6 h-6 rounded" />
								{/if}
								<span class="font-bold">{team.team_name}</span>
								{#if team.clinched_playoffs}
									<span class="badge badge-success badge-xs" title="Clinched Playoffs">✓</span>
								{/if}
								{#if team.draft_grade}
									<span class="badge badge-ghost badge-xs" title="Draft Grade">{team.draft_grade}</span>
								{/if}
							</div>
						</td>
						<td class="text-base-content/70">{team.owner_name ?? '—'}</td>
						<td class="text-center font-mono">{fmtRecord(team)}</td>
						<td class="text-right font-mono">{team.points_for.toFixed(1)}</td>
						<td class="text-right font-mono">{team.points_against.toFixed(1)}</td>
						<td class="text-center">
							{#if team.streak_type && team.streak_value > 0}
								<span class="badge badge-sm {team.streak_type === 'win' ? 'badge-success' : 'badge-error'}">
									{team.streak_type === 'win' ? 'W' : 'L'}{team.streak_value}
								</span>
							{:else}
								<span class="text-base-content/30">—</span>
							{/if}
						</td>
						<td>
							<button
								class="btn btn-ghost btn-xs"
								onclick={() => toggleRoster(team)}
							>
								{expandedTeamId === team.id ? 'Hide' : 'Roster'}
							</button>
						</td>
					</tr>
					{#if expandedTeamId === team.id}
						<tr>
							<td colspan="8" class="bg-base-200 p-0">
								{#if rosterLoading}
									<div class="p-4 text-center">
										<span class="loading loading-spinner loading-sm"></span>
									</div>
								{:else if roster.length === 0}
									<div class="p-4 text-center text-base-content/50">No roster data</div>
								{:else}
									<div class="px-4 py-2">
										<table class="table table-compact table-sm w-full">
											<thead>
												<tr>
													<th>Player</th>
													<th>Pos</th>
													<th>NFL Team</th>
													<th>Slot</th>
													<th>Matched</th>
												</tr>
											</thead>
											<tbody>
												{#each roster as entry}
													<tr>
														<td class="font-medium">
															{#if entry.player_id}
																<a href="/players/{entry.player_id}" class="text-primary hover:underline">{entry.player_name}</a>
															{:else}
																{entry.player_name}
															{/if}
														</td>
														<td>{entry.player_position}</td>
														<td>{entry.nfl_team ?? '—'}</td>
														<td>{entry.roster_position ?? '—'}</td>
														<td>
															{#if entry.matched}
																<span class="badge badge-success badge-xs">✓</span>
															{:else}
																<span class="badge badge-warning badge-xs">?</span>
															{/if}
														</td>
													</tr>
												{/each}
											</tbody>
										</table>
									</div>
								{/if}
							</td>
						</tr>
					{/if}
				{/each}
			</tbody>
		</table>
		</div>
	</div>
{/if}
