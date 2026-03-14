<script lang="ts">
	import { page } from '$app/stores';
	import { getPlayerSummary, type PlayerSummary, type SeasonTotals, type PlayerStat, type FantasyRank } from '$lib/api';

	let summary: PlayerSummary | null = $state(null);
	let loading = $state(true);
	let error = $state('');

	async function loadSummary(id: number) {
		loading = true;
		error = '';
		try {
			summary = await getPlayerSummary(id);
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load player';
			console.error('Failed to load player summary', e);
		} finally {
			loading = false;
		}
	}

	$effect(() => {
		const id = Number($page.params.id);
		if (id) loadSummary(id);
	});

	// Helpers
	function fmtNum(n: number | undefined | null): string {
		if (n === undefined || n === null) return '—';
		return n.toLocaleString();
	}

	function fmtDec(n: number | undefined | null, decimals = 1): string {
		if (n === undefined || n === null) return '—';
		return n.toLocaleString(undefined, { minimumFractionDigits: decimals, maximumFractionDigits: decimals });
	}

	function fmtHeight(inches: unknown): string {
		if (!inches || typeof inches !== 'string') return '—';
		const total = parseInt(inches);
		if (isNaN(total)) return String(inches);
		const ft = Math.floor(total / 12);
		const inn = total % 12;
		return `${ft}'${inn}"`;
	}

	function meta(key: string): unknown {
		return summary?.player?.metadata?.[key];
	}

	function positionStatKeys(pos: string | undefined | null): { key: keyof SeasonTotals; label: string }[] {
		const passing = [
			{ key: 'passing_yards' as keyof SeasonTotals, label: 'Pass Yd' },
			{ key: 'passing_tds' as keyof SeasonTotals, label: 'Pass TD' },
			{ key: 'interceptions' as keyof SeasonTotals, label: 'INT' },
			{ key: 'completions' as keyof SeasonTotals, label: 'Cmp' },
			{ key: 'attempts' as keyof SeasonTotals, label: 'Att' },
		];
		const rushing = [
			{ key: 'carries' as keyof SeasonTotals, label: 'Carries' },
			{ key: 'rushing_yards' as keyof SeasonTotals, label: 'Rush Yd' },
			{ key: 'rushing_tds' as keyof SeasonTotals, label: 'Rush TD' },
		];
		const receiving = [
			{ key: 'targets' as keyof SeasonTotals, label: 'Targets' },
			{ key: 'receptions' as keyof SeasonTotals, label: 'Rec' },
			{ key: 'receiving_yards' as keyof SeasonTotals, label: 'Rec Yd' },
			{ key: 'receiving_tds' as keyof SeasonTotals, label: 'Rec TD' },
		];
		const fantasy = [
			{ key: 'fantasy_points_ppr' as keyof SeasonTotals, label: 'PPR' },
			{ key: 'fantasy_points' as keyof SeasonTotals, label: 'STD' },
		];

		switch (pos?.toUpperCase()) {
			case 'QB':
				return [...passing, ...rushing, ...fantasy];
			case 'RB':
				return [...rushing, ...receiving, ...fantasy];
			case 'WR':
			case 'TE':
				return [...receiving, ...rushing, ...fantasy];
			default:
				return [...passing, ...rushing, ...receiving, ...fantasy];
		}
	}

	// Game log stat columns based on position
	function gameStatKeys(pos: string | undefined | null): { key: keyof PlayerStat; label: string }[] {
		const passing = [
			{ key: 'passing_yards' as keyof PlayerStat, label: 'Pass Yd' },
			{ key: 'passing_tds' as keyof PlayerStat, label: 'Pass TD' },
			{ key: 'interceptions' as keyof PlayerStat, label: 'INT' },
		];
		const rushing = [
			{ key: 'rushing_yards' as keyof PlayerStat, label: 'Rush Yd' },
			{ key: 'rushing_tds' as keyof PlayerStat, label: 'Rush TD' },
		];
		const receiving = [
			{ key: 'receptions' as keyof PlayerStat, label: 'Rec' },
			{ key: 'receiving_yards' as keyof PlayerStat, label: 'Rec Yd' },
			{ key: 'receiving_tds' as keyof PlayerStat, label: 'Rec TD' },
		];
		const fantasy = [
			{ key: 'fantasy_points_ppr' as keyof PlayerStat, label: 'PPR' },
		];

		switch (pos?.toUpperCase()) {
			case 'QB':
				return [...passing, ...rushing, ...fantasy];
			case 'RB':
				return [...rushing, ...receiving, ...fantasy];
			case 'WR':
			case 'TE':
				return [...receiving, ...rushing, ...fantasy];
			default:
				return [...passing, ...rushing, ...receiving, ...fantasy];
		}
	}
</script>

{#if loading}
	<div class="card bg-base-200 shadow-md border border-base-300 p-8 text-center">
		<span class="loading loading-dots loading-md text-primary"></span>
		<p class="text-sm opacity-60 mt-2">Loading player profile...</p>
	</div>
{:else if error}
	<div class="card bg-base-200 shadow-md border border-base-300 p-8 text-center">
		<p class="text-error font-bold">Error</p>
		<p class="text-sm opacity-60 mt-1">{error}</p>
		<a href="/players" class="btn btn-sm btn-ghost mt-4">← Back to Players</a>
	</div>
{:else if summary}
	{@const p = summary.player}
	{@const pos = p.player_position}
	{@const career = summary.career_totals}
	{@const seasonCols = positionStatKeys(pos)}
	{@const gameCols = gameStatKeys(pos)}

	<!-- PLAYER HEADER -->
	<div class="flex flex-col md:flex-row gap-4 md:gap-6 mb-6">
		<!-- Headshot + name -->
		<div class="flex items-center gap-4">
			{#if p.metadata?.headshot_url}
				<div class="avatar">
					<div class="w-20 h-20 md:w-24 md:h-24 rounded-full bg-base-300 ring ring-primary/20">
						<img src={String(p.metadata.headshot_url)} alt={p.player_name} />
					</div>
				</div>
			{:else}
				<div class="avatar placeholder">
					<div class="w-20 h-20 md:w-24 md:h-24 rounded-full bg-base-300 text-base-content/40 ring ring-primary/20">
						<span class="text-2xl">{p.player_name.split(' ').map((n: string) => n[0]).join('').slice(0, 2)}</span>
					</div>
				</div>
			{/if}
			<div>
				<h1 class="text-2xl md:text-3xl font-bold text-primary tracking-wide">{p.player_name}</h1>
				<div class="flex flex-wrap items-center gap-2 mt-1 text-sm opacity-70">
					{#if meta('jersey_number')}
						<span class="font-bold text-accent">#{meta('jersey_number')}</span>
						<span class="opacity-30">·</span>
					{/if}
					{#if pos}
						<span>{pos}</span>
						<span class="opacity-30">·</span>
					{/if}
					{#if p.team}
						<span class="font-semibold">{p.team}</span>
					{/if}
				</div>
				<!-- Mobile: compact meta row -->
				<div class="flex flex-wrap items-center gap-x-3 gap-y-1 mt-2 text-xs opacity-50">
					{#if meta('height')}
						<span>{fmtHeight(meta('height'))}</span>
					{/if}
					{#if meta('weight')}
						<span>{meta('weight')} lbs</span>
					{/if}
					{#if meta('college')}
						<span>{meta('college')}</span>
					{/if}
					{#if meta('years_exp') !== undefined}
						<span>{meta('years_exp')} yrs exp</span>
					{/if}
				</div>
			</div>
		</div>

		<!-- Status badges — desktop right side -->
		<div class="hidden md:flex md:ml-auto md:items-start gap-2 flex-wrap">
			{#if meta('status')}
				{@const status = String(meta('status'))}
				<span class="badge {status === 'ACT' ? 'badge-success' : 'badge-ghost'}">{status}</span>
			{/if}
			{#if meta('birth_date')}
				<span class="badge badge-ghost badge-sm">Born: {meta('birth_date')}</span>
			{/if}
			{#if p.source}
				<span class="badge badge-ghost badge-sm">{p.source}</span>
			{/if}
		</div>
	</div>

	<!-- CAREER TOTALS — stat cards row -->
	{#if career.games_played > 0}
		<div class="mb-6">
			<h2 class="text-lg font-bold text-primary tracking-wide mb-3">// CAREER TOTALS</h2>
			<div class="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-6 gap-2 md:gap-3">
				<div class="card bg-base-200 border border-base-300 p-3 text-center">
					<div class="text-xs opacity-50 uppercase tracking-wider">Games</div>
					<div class="text-xl font-bold text-accent">{fmtNum(career.games_played)}</div>
				</div>
				{#each seasonCols.slice(0, 7) as col}
					<div class="card bg-base-200 border border-base-300 p-3 text-center">
						<div class="text-xs opacity-50 uppercase tracking-wider">{col.label}</div>
						<div class="text-xl font-bold text-accent">{fmtNum(career[col.key] as number)}</div>
					</div>
				{/each}
			</div>
		</div>
	{/if}

	<!-- SEASON-BY-SEASON TABLE -->
	{#if (summary.seasons ?? []).length > 0}
		<div class="mb-6">
			<h2 class="text-lg font-bold text-primary tracking-wide mb-3">// SEASON BY SEASON</h2>
			<div class="card bg-base-100 shadow-md border border-base-300 overflow-hidden">
				<div class="table-scroll-wrap">
					<table class="table table-zebra table-pin-rows table-sm table-responsive">
						<thead>
							<tr>
								<th>Season</th>
								<th class="text-right">GP</th>
								{#each seasonCols as col}
									<th class="text-right">{col.label}</th>
								{/each}
							</tr>
						</thead>
						<tbody>
							{#each summary.seasons as season}
								<tr class="hover">
									<td class="font-bold text-primary">{season.season}</td>
									<td class="text-right">{season.games_played}</td>
									{#each seasonCols as col}
										<td class="text-right">{fmtNum(season[col.key] as number)}</td>
									{/each}
								</tr>
							{/each}
						</tbody>
					</table>
				</div>
			</div>
		</div>
	{/if}

	<!-- RECENT GAME LOG -->
	{#if (summary.recent_games ?? []).length > 0}
		<div class="mb-6">
			<h2 class="text-lg font-bold text-primary tracking-wide mb-3">// RECENT GAMES</h2>
			<div class="card bg-base-100 shadow-md border border-base-300 overflow-hidden">
				<div class="table-scroll-wrap">
					<table class="table table-zebra table-pin-rows table-sm table-responsive">
						<thead>
							<tr>
								<th>Szn</th>
								<th>Wk</th>
								<th>Opp</th>
								{#each gameCols as col}
									<th class="text-right">{col.label}</th>
								{/each}
							</tr>
						</thead>
						<tbody>
							{#each summary.recent_games as g}
								<tr class="hover">
									<td>{g.season}</td>
									<td>{g.week}</td>
									<td class="font-semibold">{g.opponent_team ?? '—'}</td>
									{#each gameCols as col}
										<td class="text-right {col.key === 'fantasy_points_ppr' ? 'font-bold text-accent' : ''}">
											{fmtNum(g[col.key] as number)}
										</td>
									{/each}
								</tr>
							{:else}
								<tr>
									<td colspan="{gameCols.length + 3}" class="text-center opacity-50 py-8">No game data</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>
			</div>
		</div>
	{/if}

	<!-- FANTASY RANKINGS -->
	{#if (summary.rankings ?? []).length > 0}
		<div class="mb-6">
			<h2 class="text-lg font-bold text-primary tracking-wide mb-3">// FANTASY RANKINGS</h2>
			<div class="card bg-base-100 shadow-md border border-base-300 overflow-hidden">
				<div class="table-scroll-wrap">
					<table class="table table-zebra table-pin-rows table-sm table-responsive">
						<thead>
							<tr>
								<th>Type</th>
								<th>Szn</th>
								<th>Wk</th>
								<th class="text-right">Rank</th>
								<th class="text-right">ECR</th>
								<th class="text-right">Best</th>
								<th class="text-right">Worst</th>
								<th class="text-right">Avg</th>
							</tr>
						</thead>
						<tbody>
							{#each summary.rankings as r}
								<tr class="hover">
									<td><span class="badge badge-ghost badge-sm">{r.rank_type ?? '—'}</span></td>
									<td>{r.season ?? '—'}</td>
									<td>{r.week ?? '—'}</td>
									<td class="text-right font-bold text-accent">{fmtNum(r.rank)}</td>
									<td class="text-right">{fmtDec(r.ecr)}</td>
									<td class="text-right">{fmtNum(r.best)}</td>
									<td class="text-right">{fmtNum(r.worst)}</td>
									<td class="text-right">{fmtDec(r.avg)}</td>
								</tr>
							{:else}
								<tr>
									<td colspan="8" class="text-center opacity-50 py-8">No ranking data</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>
			</div>
		</div>
	{/if}

	<!-- Back link -->
	<div class="mt-4">
		<a href="/players" class="btn btn-ghost btn-sm">← Back to Players</a>
	</div>
{/if}
