<script lang="ts">
	import { onMount, tick } from 'svelte';
	import PageHeader from '$lib/components/PageHeader.svelte';
	import {
		listFantasyLeagues,
		getFantasyLeague,
		getFantasyMatchups,
		type FantasyLeague,
		type FantasyTeam,
		type FantasyMatchup,
		type LeagueDetail
	} from '$lib/api';

	// ── State ──
	let allLeagues: FantasyLeague[] = $state([]);
	let loading = $state(true);

	// Grouped leagues: league_name → sorted list of FantasyLeague (by season desc)
	let leagueGroups: Record<string, FantasyLeague[]> = $state({});

	// Per-group: selected season league ID
	let selectedLeagueId: Record<string, number> = $state({});

	// Per-group: loaded detail + matchups
	let groupDetail: Record<string, LeagueDetail | null> = $state({});
	let groupMatchups: Record<string, FantasyMatchup[]> = $state({});
	let groupLoading: Record<string, boolean> = $state({});
	let groupChartRefs: Record<string, HTMLDivElement | undefined> = $state({});
	let groupChartInstances: Record<string, any> = $state({});
	let groupChartMode: Record<string, 'cumulative' | 'weekly'> = $state({});

	// Global season filter
	let globalSeason: number | '' = $state('');

	async function loadAllLeagues() {
		loading = true;
		try {
			const res = await listFantasyLeagues({ limit: 100 });
			allLeagues = res.items;
			buildGroups();
		} catch (e) {
			console.error('Failed to load leagues', e);
		} finally {
			loading = false;
		}
	}

	function buildGroups() {
		const groups: Record<string, FantasyLeague[]> = {};
		for (const lg of allLeagues) {
			const key = lg.league_name;
			if (!groups[key]) groups[key] = [];
			groups[key].push(lg);
		}
		// Sort each group by season descending
		for (const list of Object.values(groups)) {
			list.sort((a, b) => b.season - a.season);
		}
		leagueGroups = groups;

		// Auto-select latest season for each group
		for (const [name, list] of Object.entries(groups)) {
			if (!selectedLeagueId[name]) {
				const match = globalSeason ? list.find(l => l.season === globalSeason) : list[0];
				if (match) {
					selectedLeagueId[name] = match.id;
					loadGroupData(name, match.id);
				}
			}
		}
	}

	async function loadGroupData(groupName: string, leagueId: number) {
		groupLoading[groupName] = true;
		groupChartMode[groupName] = groupChartMode[groupName] || 'cumulative';
		try {
			const [detail, matchups] = await Promise.all([
				getFantasyLeague(leagueId),
				getFantasyMatchups(leagueId)
			]);
			groupDetail[groupName] = detail;
			groupMatchups[groupName] = matchups;
			await tick();
			renderGroupChart(groupName);
		} catch (e) {
			console.error(`Failed to load data for ${groupName}`, e);
		} finally {
			groupLoading[groupName] = false;
		}
	}

	function selectSeason(groupName: string, leagueId: number) {
		selectedLeagueId[groupName] = leagueId;
		loadGroupData(groupName, leagueId);
	}

	function setGlobalSeason(season: number | '') {
		globalSeason = season;
		for (const [name, list] of Object.entries(leagueGroups)) {
			const match = season ? list.find(l => l.season === season) : list[0];
			if (match) {
				selectedLeagueId[name] = match.id;
				loadGroupData(name, match.id);
			}
		}
	}

	function toggleChartMode(groupName: string) {
		groupChartMode[groupName] = groupChartMode[groupName] === 'cumulative' ? 'weekly' : 'cumulative';
		renderGroupChart(groupName);
	}

	function renderGroupChart(groupName: string) {
		const container = groupChartRefs[groupName];
		const matchups = groupMatchups[groupName] || [];
		if (!container || matchups.length === 0) return;

		if (groupChartInstances[groupName]) {
			groupChartInstances[groupName].destroy();
			groupChartInstances[groupName] = null;
		}

		const mode = groupChartMode[groupName] || 'cumulative';

		// Group by team
		const teamWeekly: Record<string, Record<number, number>> = {};
		for (const m of matchups) {
			if (!teamWeekly[m.team_name]) teamWeekly[m.team_name] = {};
			teamWeekly[m.team_name][m.week] = m.points;
		}

		const weeks = [...new Set(matchups.map(m => m.week))].sort((a, b) => a - b);
		const teamTotals = Object.entries(teamWeekly).map(([name, wk]) => ({
			name,
			total: Object.values(wk).reduce((a, b) => a + b, 0)
		}));
		teamTotals.sort((a, b) => b.total - a.total);

		const colors = [
			'#3b82f6', '#ef4444', '#22c55e', '#f59e0b', '#8b5cf6',
			'#ec4899', '#14b8a6', '#f97316', '#6366f1', '#06b6d4',
			'#84cc16', '#e11d48', '#0ea5e9', '#a855f7'
		];

		const series = teamTotals.map(({ name }) => {
			const weekData = teamWeekly[name];
			let cumulative = 0;
			return {
				name,
				data: weeks.map(w => {
					const pts = weekData[w] || 0;
					if (mode === 'cumulative') {
						cumulative += pts;
						return Math.round(cumulative * 10) / 10;
					}
					return Math.round(pts * 10) / 10;
				})
			};
		});

		const isDark = document.documentElement.getAttribute('data-theme')?.includes('dark') ||
			window.matchMedia('(prefers-color-scheme: dark)').matches;

		const options: any = {
			chart: {
				type: 'line',
				height: 320,
				fontFamily: 'inherit',
				background: 'transparent',
				toolbar: { show: true, tools: { download: true, zoom: true, pan: false, reset: true } },
				zoom: { enabled: true },
				animations: { enabled: true, speed: 500 }
			},
			series,
			xaxis: {
				categories: weeks.map(w => `${w}`),
				title: { text: 'Week', style: { fontWeight: '500' } },
				labels: { style: { fontSize: '10px' } }
			},
			yaxis: {
				title: { text: mode === 'cumulative' ? 'Cumulative Pts' : 'Points' },
				labels: { formatter: (v: number) => v.toFixed(0), offsetX: -5 }
			},
			stroke: {
				width: mode === 'cumulative' ? 2.5 : 2,
				curve: 'smooth'
			},
			colors: colors.slice(0, series.length),
			legend: {
				position: 'bottom',
				fontSize: '10px',
				markers: { size: 3 },
				itemMargin: { horizontal: 6, vertical: 2 }
			},
			tooltip: {
				shared: true,
				intersect: false,
				y: { formatter: (v: number) => v.toFixed(1) }
			},
			grid: {
				strokeDashArray: 3,
				borderColor: isDark ? '#374151' : '#e5e7eb'
			},
			theme: { mode: isDark ? 'dark' : 'light' },
			markers: {
				size: mode === 'weekly' ? 2 : 0,
				hover: { size: 4 }
			}
		};

		import('apexcharts').then(({ default: ApexCharts }) => {
			const inst = new ApexCharts(container, options);
			inst.render();
			groupChartInstances[groupName] = inst;
		});
	}

	function platformBadge(platform: string): string {
		switch (platform) {
			case 'yahoo': return 'badge-primary';
			case 'espn': return 'badge-error';
			default: return 'badge-ghost';
		}
	}

	function platformColor(platform: string): string {
		switch (platform) {
			case 'yahoo': return 'text-primary';
			case 'espn': return 'text-error';
			default: return '';
		}
	}

	function fmtRecord(t: FantasyTeam): string {
		let r = `${t.wins}-${t.losses}`;
		if (t.ties > 0) r += `-${t.ties}`;
		return r;
	}

	function getGroupSeasons(groupName: string): FantasyLeague[] {
		return leagueGroups[groupName] || [];
	}

	function getSelectedLeague(groupName: string): FantasyLeague | null {
		const id = selectedLeagueId[groupName];
		return allLeagues.find(l => l.id === id) || null;
	}

	// All available seasons across all groups for global filter
	let allSeasons: number[] = $derived(
		[...new Set(allLeagues.map(l => l.season))].sort((a, b) => b - a)
	);

	onMount(loadAllLeagues);
</script>

<PageHeader
	title="Fantasy Leagues"
	breadcrumbs={[{ label: 'Fantasy', href: '/leagues' }, { label: 'Overview' }]}
>
	{#snippet actions()}
		<div class="flex items-center gap-3">
			<span class="text-base-content/60 text-sm">{allLeagues.length} seasons</span>
			<select
				class="select select-bordered select-sm"
				value={globalSeason}
				onchange={(e) => setGlobalSeason(e.currentTarget.value ? Number(e.currentTarget.value) : '')}
			>
				<option value="">Latest Season</option>
				{#each allSeasons as y}
					<option value={y}>{y}</option>
				{/each}
			</select>
		</div>
	{/snippet}
</PageHeader>

{#if loading}
	<div class="card bg-base-100 shadow-sm p-12 text-center">
		<span class="loading loading-dots loading-lg text-primary"></span>
		<p class="text-base-content/60 text-sm mt-3">Loading leagues...</p>
	</div>
{:else if Object.keys(leagueGroups).length === 0}
	<div class="card bg-base-100 shadow-sm p-12 text-center">
		<p class="text-base-content/50">No leagues imported yet.</p>
		<p class="text-base-content/40 text-xs mt-1">Use Data Management → Import → Fantasy to get started.</p>
	</div>
{:else}
	<div class="grid grid-cols-1 gap-6 xl:grid-cols-2">
		{#each Object.entries(leagueGroups) as [groupName, groupLeagues] (groupName)}
			{@const selectedLg = getSelectedLeague(groupName)}
			{@const detail = groupDetail[groupName]}
			{@const isLoading = groupLoading[groupName]}
			{@const matchups = groupMatchups[groupName] || []}
			{@const mode = groupChartMode[groupName] || 'cumulative'}
			{@const teams = detail?.teams || []}

			<div class="card bg-base-100 shadow-sm">
				<div class="card-body p-0">
					<!-- Card Header -->
					<div class="flex items-start justify-between gap-4 px-5 pt-5">
						<div class="flex items-center gap-3">
							{#if selectedLg}
								<div class="bg-base-200 rounded-box flex items-center p-2">
									{#if selectedLg.platform === 'yahoo'}
										<span class="text-lg">🟣</span>
									{:else}
										<span class="text-lg">🔴</span>
									{/if}
								</div>
							{/if}
							<div>
								<h2 class="text-lg font-semibold">{groupName}</h2>
								{#if selectedLg}
									<div class="flex items-center gap-2 mt-0.5">
										<span class="badge {platformBadge(selectedLg.platform)} badge-xs uppercase">{selectedLg.platform}</span>
										<span class="text-base-content/60 text-xs">{groupLeagues.length} season{groupLeagues.length !== 1 ? 's' : ''}</span>
									</div>
								{/if}
							</div>
						</div>
						<select
							class="select select-bordered select-sm min-w-24"
							value={selectedLeagueId[groupName]}
							onchange={(e) => selectSeason(groupName, Number(e.currentTarget.value))}
						>
							{#each groupLeagues as lg}
								<option value={lg.id}>{lg.season}</option>
							{/each}
						</select>
					</div>

					{#if isLoading}
						<div class="flex justify-center py-16">
							<span class="loading loading-dots loading-md text-primary"></span>
						</div>
					{:else if detail && selectedLg}
						<!-- Stat Row -->
						<div class="mt-4 px-5">
							<div class="grid grid-cols-2 gap-3 sm:grid-cols-4">
								<div class="card bg-base-200/50 p-3">
									<p class="text-base-content/60 text-xs font-medium uppercase tracking-wide">Teams</p>
									<p class="mt-1 text-xl font-semibold">{teams.length}</p>
								</div>
								<div class="card bg-base-200/50 p-3">
									<p class="text-base-content/60 text-xs font-medium uppercase tracking-wide">Scoring</p>
									<p class="mt-1 text-xl font-semibold capitalize">{selectedLg.scoring_type?.replaceAll('_', ' ') || '—'}</p>
								</div>
								{#each [teams.find(t => t.standing_rank === 1)] as champion}
								<div class="card bg-base-200/50 p-3 col-span-2 sm:col-span-2">
									<p class="text-base-content/60 text-xs font-medium uppercase tracking-wide">Champion</p>
									<div class="mt-1 flex items-center gap-2">
										{#if champion?.logo_url}
											<img src={champion.logo_url} alt="" class="w-5 h-5 rounded" />
										{/if}
										<p class="text-lg font-semibold truncate">{champion?.team_name || '—'}</p>
										{#if champion}
											<span class="text-base-content/60 text-xs font-mono">{fmtRecord(champion)}</span>
										{/if}
									</div>
								</div>
								{/each}
							</div>
						</div>

						<!-- Chart -->
						<div class="mt-4 px-5">
							<div class="flex items-center justify-between mb-2">
								<span class="text-sm font-medium">Season Scoring</span>
								<div class="tabs tabs-box tabs-xs">
									<button
										class="tab px-3 {mode === 'cumulative' ? 'tab-active' : ''}"
										onclick={() => { groupChartMode[groupName] = 'cumulative'; renderGroupChart(groupName); }}
									>
										Cumulative
									</button>
									<button
										class="tab px-3 {mode === 'weekly' ? 'tab-active' : ''}"
										onclick={() => { groupChartMode[groupName] = 'weekly'; renderGroupChart(groupName); }}
									>
										Weekly
									</button>
								</div>
							</div>
							{#if matchups.length > 0}
								<div bind:this={groupChartRefs[groupName]}></div>
							{:else}
								<div class="text-center py-10 text-base-content/40">
									<p class="text-sm">No matchup data</p>
									<p class="text-xs mt-1">Re-import to collect weekly scores</p>
								</div>
							{/if}
						</div>

						<!-- Mini Standings -->
						<div class="mt-2 px-5 pb-4">
							<div class="overflow-x-auto">
								<table class="table table-sm">
									<thead>
										<tr class="text-xs">
											<th class="w-8 text-right">#</th>
											<th>Team</th>
											<th class="text-center">Record</th>
											<th class="text-right">PF</th>
											<th class="text-center">Streak</th>
										</tr>
									</thead>
									<tbody>
										{#each teams.slice(0, 5) as team, i}
											<tr class="hover">
												<td class="text-right text-xs font-bold text-accent">{team.standing_rank ?? i + 1}</td>
												<td>
													<div class="flex items-center gap-2">
														{#if team.logo_url}
															<img src={team.logo_url} alt="" class="w-5 h-5 rounded" />
														{/if}
														<span class="font-medium text-sm truncate max-w-32">{team.team_name}</span>
														{#if team.clinched_playoffs}
															<span class="badge badge-success badge-xs">✓</span>
														{/if}
													</div>
												</td>
												<td class="text-center font-mono text-xs">{fmtRecord(team)}</td>
												<td class="text-right font-mono text-xs">{team.points_for.toFixed(1)}</td>
												<td class="text-center">
													{#if team.streak_type && team.streak_value > 0}
														<span class="badge badge-sm {team.streak_type === 'win' ? 'badge-success' : 'badge-error'}">
															{team.streak_type === 'win' ? 'W' : 'L'}{team.streak_value}
														</span>
													{:else}
														<span class="text-base-content/30">—</span>
													{/if}
												</td>
											</tr>
										{/each}
									</tbody>
								</table>
							</div>
							{#if teams.length > 5}
								<div class="mt-2 text-center">
									<a href="/leagues/{selectedLeagueId[groupName]}" class="btn btn-ghost btn-xs text-primary">
										View all {teams.length} teams →
									</a>
								</div>
							{:else}
								<div class="mt-2 text-center">
									<a href="/leagues/{selectedLeagueId[groupName]}" class="btn btn-ghost btn-xs text-primary">
										Full details →
									</a>
								</div>
							{/if}
						</div>
					{/if}
				</div>
			</div>
		{/each}
	</div>

	<!-- All Seasons Grid -->
	<div class="mt-8">
		<h2 class="text-lg font-medium mb-4">All Seasons</h2>
		<div class="grid gap-3 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5">
			{#each [...allLeagues].sort((a, b) => b.season - a.season || a.league_name.localeCompare(b.league_name)) as league}
				<a href="/leagues/{league.id}" class="card bg-base-100 shadow-sm hover:shadow-md transition-all">
					<div class="card-body gap-1 p-4">
						<div class="flex items-center justify-between text-xs">
							<span class="badge {platformBadge(league.platform)} badge-xs uppercase">{league.platform}</span>
							<span class="text-base-content/60 font-mono">{league.season}</span>
						</div>
						<p class="font-medium text-sm mt-1 truncate">{league.league_name}</p>
						<div class="flex items-center gap-3 text-xs text-base-content/60 mt-0.5">
							{#if league.num_teams}
								<span>{league.num_teams} teams</span>
							{/if}
							{#if league.scoring_type}
								<span class="capitalize">{league.scoring_type.replaceAll('_', ' ')}</span>
							{/if}
						</div>
					</div>
				</a>
			{/each}
		</div>
	</div>
{/if}
