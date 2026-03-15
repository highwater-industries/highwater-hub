<script lang="ts">
	import { onMount } from 'svelte';
	import PageHeader from '$lib/components/PageHeader.svelte';
	import {
		listFitnessUsers,
		getUserProgress,
		getLatestBodyweight,
		type FitnessUser,
		type ExerciseProgressCard,
		type ExerciseHistoryEntry,
		type WorkoutSet
	} from '$lib/api';

	let users: FitnessUser[] = $state([]);
	let activeUser: FitnessUser | null = $state(null);
	let cards: ExerciseProgressCard[] = $state([]);
	let loading = $state(true);
	let error = $state('');
	let userBodyweight: number | null = $state(null);
	let filterCategory = $state('all');

	// Chart instances for cleanup
	let chartInstances: Record<number, any> = {};
	let ApexChartsClass: any = null;

	const difficultyLabels: Record<number, string> = { 1: 'Easy', 2: 'Light', 3: 'Moderate', 4: 'Hard', 5: 'Max' };
	const difficultyColors: Record<number, string> = { 1: 'badge-soft badge-success', 2: 'badge-soft badge-info', 3: 'badge-soft badge-warning', 4: 'badge-soft badge-error', 5: 'badge-soft badge-error' };

	const categoryIcons: Record<string, string> = {
		strength: 'lucide--dumbbell',
		bodyweight: 'lucide--person-standing',
		cardio: 'lucide--heart-pulse'
	};

	// ── Load ──

	async function loadUsers() {
		try {
			users = await listFitnessUsers();
			const savedId = localStorage.getItem('fitness_user_id');
			if (savedId) {
				activeUser = users.find((u) => u.id === Number(savedId)) ?? null;
			}
			if (!activeUser && users.length > 0) {
				activeUser = users[0];
			}
			if (activeUser) {
				await loadBodyweight();
				await loadProgress();
			}
		} catch (e) {
			error = 'Failed to load users';
			console.error(e);
		} finally {
			loading = false;
		}
	}

	function selectUser(user: FitnessUser) {
		activeUser = user;
		localStorage.setItem('fitness_user_id', String(user.id));
		loadBodyweight();
		loadProgress();
	}

	async function loadBodyweight() {
		if (!activeUser) return;
		try {
			const entry = await getLatestBodyweight(activeUser.id);
			userBodyweight = entry?.weight_lbs ?? null;
		} catch {
			userBodyweight = null;
		}
	}

	async function loadProgress() {
		if (!activeUser) return;
		loading = true;
		error = '';
		try {
			cards = await getUserProgress(activeUser.id, 6);
		} catch (e) {
			error = 'Failed to load progress';
			console.error(e);
		} finally {
			loading = false;
		}
	}

	onMount(async () => {
		ApexChartsClass = (await import('apexcharts')).default;
		await loadUsers();
	});

	// Svelte use: action — renders chart when element mounts
	function chartAction(node: HTMLDivElement, card: ExerciseProgressCard) {
		let instance: any = null;

		function create() {
			if (!ApexChartsClass || card.sessions.length < 2) return;

			const isCardio = card.exercise_category === 'cardio';
			const reversed = [...card.sessions].reverse();

			let primaryData: number[];
			let primaryLabel: string;
			if (isCardio) {
				primaryData = reversed.map(s => {
					const fs = s.sets.length > 0 ? s.sets[s.sets.length - 1] : null;
					return fs?.distance_miles ?? 0;
				});
				primaryLabel = 'Distance (mi)';
			} else {
				primaryData = reversed.map(s => {
					const best = bestSet(s, card.exercise_category);
					return best ? effectiveWeight(best, card.exercise_category) : 0;
				});
				primaryLabel = 'Best Weight (lb)';
			}

			let secondaryData: number[];
			let secondaryLabel: string;
			if (isCardio) {
				secondaryData = reversed.map(s => {
					const fs = s.sets.length > 0 ? s.sets[s.sets.length - 1] : null;
					return fs?.duration_seconds ? Math.round(fs.duration_seconds / 60) : 0;
				});
				secondaryLabel = 'Duration (min)';
			} else {
				secondaryData = reversed.map(s => totalVolume(s, card.exercise_category));
				secondaryLabel = 'Volume (lb)';
			}

			const dates = reversed.map(s => {
				const d = new Date(s.date);
				return d.toLocaleDateString('en-US', { month: 'short', day: 'numeric' });
			});

			instance = new ApexChartsClass(node, {
				chart: {
					type: 'area',
					height: 180,
					toolbar: { show: false },
					zoom: { enabled: false },
					background: 'transparent',
					fontFamily: 'inherit',
				},
				series: [
					{ name: primaryLabel, data: primaryData },
					{ name: secondaryLabel, data: secondaryData },
				],
				xaxis: {
					categories: dates,
					labels: { style: { fontSize: '10px', colors: ['rgba(128,128,128,0.6)'] } },
					axisBorder: { show: false },
					axisTicks: { show: false },
				},
				yaxis: [
					{
						title: { text: primaryLabel, style: { color: '#167bff', fontSize: '10px', fontWeight: '600' } },
						labels: { style: { fontSize: '10px', colors: ['#167bff'] } },
					},
					{
						opposite: true,
						title: { text: secondaryLabel, style: { color: '#a855f7', fontSize: '10px', fontWeight: '600' } },
						labels: { style: { fontSize: '10px', colors: ['#a855f7'] } },
					}
				],
				stroke: { curve: 'smooth', width: [2.5, 1.5] },
				fill: {
					type: 'gradient',
					gradient: { shadeIntensity: 1, opacityFrom: 0.4, opacityTo: 0.05, stops: [0, 100] },
				},
				colors: ['#167bff', '#a855f7'],
				grid: {
					borderColor: 'rgba(128,128,128,0.12)',
					strokeDashArray: 3,
					padding: { left: 4, right: 4, top: -8, bottom: 0 },
				},
				dataLabels: { enabled: false },
				legend: { show: true, position: 'top', fontSize: '11px', labels: { colors: ['#167bff', '#a855f7'] }, markers: { size: 4 }, itemMargin: { horizontal: 8 } },
				tooltip: {
					theme: 'dark',
					shared: true,
					y: { formatter: (val: number) => val?.toLocaleString() ?? '—' },
				},
				markers: {
					size: [4, 0],
					colors: ['#167bff'],
					strokeColors: '#fff',
					strokeWidth: 2,
					hover: { size: 6 },
				},
			});
			instance.render();
			chartInstances[card.exercise_id] = instance;
		}

		create();

		return {
			destroy() {
				if (instance) {
					instance.destroy();
					delete chartInstances[card.exercise_id];
				}
			}
		};
	}

	// ── Derived ──

	let filteredCards = $derived(
		filterCategory === 'all' ? cards : cards.filter((c) => c.exercise_category === filterCategory)
	);

	let categories = $derived([...new Set(cards.map((c) => c.exercise_category))].sort());

	// ── Helpers ──

	function finalSet(entry: ExerciseHistoryEntry): WorkoutSet | null {
		if (!entry.sets || entry.sets.length === 0) return null;
		return entry.sets[entry.sets.length - 1];
	}

	function effectiveWeight(s: WorkoutSet, category: string): number {
		const w = s.weight ?? 0;
		if (category === 'bodyweight') {
			return (userBodyweight ?? 0) + w;
		}
		return w;
	}

	function bestSet(entry: ExerciseHistoryEntry, category = 'strength'): WorkoutSet | null {
		if (!entry.sets || entry.sets.length === 0) return null;
		let best = entry.sets[0];
		for (const s of entry.sets) {
			const ew = effectiveWeight(s, category);
			const bw = effectiveWeight(best, category);
			if (ew > bw) best = s;
			else if (ew === bw && (s.reps ?? 0) > (best.reps ?? 0)) best = s;
		}
		return best;
	}

	function totalVolume(entry: ExerciseHistoryEntry, category = 'strength'): number {
		return entry.sets.reduce((sum, s) => sum + effectiveWeight(s, category) * (s.reps ?? 1), 0);
	}

	function formatSet(s: WorkoutSet | null, category = 'strength'): string {
		if (!s) return '—';
		if (s.weight !== undefined && s.reps !== undefined) {
			if (category === 'bodyweight' && userBodyweight) {
				const eff = userBodyweight + (s.weight ?? 0);
				const extra = s.weight ? ` (+${s.weight})` : '';
				return `${eff}lb${extra} × ${s.reps}`;
			}
			return `${s.weight}lb × ${s.reps}`;
		}
		if (category === 'bodyweight' && s.reps !== undefined) {
			if (userBodyweight) {
				const w = s.weight ?? 0;
				const eff = userBodyweight + w;
				const extra = w > 0 ? ` (+${w})` : '';
				return `${eff}lb${extra} × ${s.reps}`;
			}
			return `${s.reps} reps`;
		}
		if (s.duration_seconds !== undefined) {
			let txt = `${Math.round(s.duration_seconds / 60)}min`;
			if (s.distance_miles !== undefined) txt += ` / ${s.distance_miles}mi`;
			return txt;
		}
		if (s.reps !== undefined) return `${s.reps} reps`;
		return '—';
	}

	function formatDate(d: string): string {
		const date = new Date(d);
		return date.toLocaleDateString('en-US', { month: 'short', day: 'numeric' });
	}

	function trend(current: ExerciseHistoryEntry, previous: ExerciseHistoryEntry | null, category = 'strength'): { icon: string; class: string } {
		if (!previous) return { icon: '', class: '' };
		const curFinal = finalSet(current);
		const prevFinal = finalSet(previous);
		if (!curFinal || !prevFinal) return { icon: '', class: '' };

		const cw = effectiveWeight(curFinal, category);
		const pw = effectiveWeight(prevFinal, category);
		const cr = curFinal.reps ?? 0;
		const pr = prevFinal.reps ?? 0;

		if (cw > pw || (cw === pw && cr > pr)) return { icon: 'lucide--trending-up', class: 'text-success' };
		if (cw < pw || (cw === pw && cr < pr)) return { icon: 'lucide--trending-down', class: 'text-error' };
		return { icon: 'lucide--minus', class: 'text-warning' };
	}

	function isPR(card: ExerciseProgressCard, sessionIdx: number): boolean {
		const session = card.sessions[sessionIdx];
		const cat = card.exercise_category;
		const best = bestSet(session, cat);
		if (!best) return false;
		const bestEff = effectiveWeight(best, cat);
		if (bestEff === 0 && (best.reps ?? 0) === 0) return false;

		for (let i = 0; i < card.sessions.length; i++) {
			if (i === sessionIdx) continue;
			const otherBest = bestSet(card.sessions[i], cat);
			if (!otherBest) continue;
			const otherEff = effectiveWeight(otherBest, cat);
			if (otherEff > bestEff) return false;
			if (otherEff === bestEff && (otherBest.reps ?? 0) > (best.reps ?? 0)) return false;
		}
		return true;
	}

	function latestBestWeight(card: ExerciseProgressCard): string {
		if (card.sessions.length === 0) return '—';
		const best = bestSet(card.sessions[0], card.exercise_category);
		if (!best) return '—';
		if (card.exercise_category === 'cardio') {
			if (best.distance_miles) return `${best.distance_miles} mi`;
			if (best.duration_seconds) return `${Math.round(best.duration_seconds / 60)} min`;
			return '—';
		}
		return `${effectiveWeight(best, card.exercise_category)} lb`;
	}

	function latestBestReps(card: ExerciseProgressCard): string {
		if (card.sessions.length === 0) return '';
		const best = bestSet(card.sessions[0], card.exercise_category);
		if (!best || best.reps === undefined) return '';
		return `× ${best.reps}`;
	}

	function trendBadge(card: ExerciseProgressCard): { icon: string; cls: string; text: string } {
		if (card.sessions.length < 2) return { icon: '', cls: '', text: '' };
		const t = trend(card.sessions[0], card.sessions[1], card.exercise_category);
		if (!t.icon) return { icon: '', cls: '', text: '' };
		if (t.icon === 'lucide--trending-up') return { icon: t.icon, cls: 'badge-success', text: 'Up' };
		if (t.icon === 'lucide--trending-down') return { icon: t.icon, cls: 'badge-error', text: 'Down' };
		return { icon: t.icon, cls: 'badge-warning', text: 'Same' };
	}
</script>

<!-- HEADER -->
<PageHeader title="Exercise Progress" breadcrumbs={[{ label: 'Fitness', href: '/fitness' }, { label: 'Progress' }]}>
	{#snippet actions()}
		<!-- User selector -->
		{#if users.length > 1}
			<select class="select select-bordered select-sm"
				onchange={(e) => {
					const u = users.find((u) => u.id === Number(e.currentTarget.value));
					if (u) selectUser(u);
				}}>
				{#each users as u}
					<option value={u.id} selected={u.id === activeUser?.id}>{u.name}</option>
				{/each}
			</select>
		{/if}

		<!-- Category filter -->
		{#if categories.length > 1}
			<select class="select select-bordered select-sm" bind:value={filterCategory}>
				<option value="all">All Categories</option>
				{#each categories as cat}
					<option value={cat}>{cat.charAt(0).toUpperCase() + cat.slice(1)}</option>
				{/each}
			</select>
		{/if}

		<a href="/fitness" class="btn btn-ghost btn-sm gap-1">
			<span class="iconify lucide--arrow-left size-4"></span>
			Workouts
		</a>
	{/snippet}
</PageHeader>

{#if loading}
	<div class="card bg-base-100 shadow-sm p-8 text-center">
		<span class="loading loading-dots loading-md text-primary"></span>
		<p class="text-sm text-base-content/60 mt-2">Loading progress...</p>
	</div>
{:else if error}
	<div class="card bg-base-100 shadow-sm p-8 text-center">
		<p class="text-error font-bold">{error}</p>
	</div>
{:else if !activeUser}
	<div class="card bg-base-100 shadow-sm p-8 text-center">
		<p class="text-base-content/60">No fitness user found. <a href="/fitness" class="link link-primary">Go to Fitness</a> to create one.</p>
	</div>
{:else if filteredCards.length === 0}
	<div class="card bg-base-100 shadow-sm p-8 text-center">
		<div class="flex justify-center mb-3">
			<div class="bg-base-200 rounded-box flex items-center p-3">
				<span class="iconify lucide--trending-up size-8 text-base-content/30"></span>
			</div>
		</div>
		<p class="text-lg font-semibold text-base-content/60">No progress data yet</p>
		<p class="text-sm text-base-content/40 mt-1">Complete a workout to start tracking exercise progress.</p>
		<a href="/fitness" class="btn btn-primary btn-sm mt-3">Go to Workouts</a>
	</div>
{:else}
	<!-- PROGRESS CARDS GRID -->
	<div class="grid grid-cols-1 lg:grid-cols-2 gap-5">
		{#each filteredCards as card (card.exercise_id)}
			{@const isCardio = card.exercise_category === 'cardio'}
			{@const tb = trendBadge(card)}
			<div class="card bg-base-100 shadow-sm transition-all hover:shadow-md overflow-hidden">
				<!-- Colored top strip based on trend -->
				<div class="h-1 {tb.cls === 'badge-success' ? 'bg-gradient-to-r from-success/60 to-success/20' : tb.cls === 'badge-error' ? 'bg-gradient-to-r from-error/60 to-error/20' : 'bg-gradient-to-r from-primary/40 to-primary/10'}"></div>
				<div class="card-body p-0">
					<!-- Card Header -->
					<div class="p-4 pb-0">
						<div class="flex items-start justify-between gap-3">
							<div class="flex items-center gap-3 min-w-0">
								<div class="bg-base-200 rounded-box flex items-center p-2">
									<span class="iconify size-5 {categoryIcons[card.exercise_category] ?? 'lucide--activity'}"></span>
								</div>
								<div class="min-w-0">
									<h3 class="font-semibold leading-tight">{card.exercise_name}</h3>
									<div class="flex items-center gap-1.5 mt-0.5">
										<span class="badge badge-soft badge-xs">{card.exercise_category}</span>
										{#if card.muscle_group}
											<span class="text-xs text-base-content/40">{card.muscle_group}</span>
										{/if}
									</div>
								</div>
							</div>
							<!-- Current best + trend -->
							<div class="text-right shrink-0">
								<div class="flex items-baseline gap-1">
									<span class="text-2xl font-bold">{latestBestWeight(card)}</span>
									{#if latestBestReps(card)}
										<span class="text-sm text-base-content/50">{latestBestReps(card)}</span>
									{/if}
								</div>
								{#if tb.icon}
									<span class="badge badge-soft badge-xs {tb.cls} gap-0.5 mt-0.5">
										<span class="iconify {tb.icon} size-3"></span>
										{tb.text}
									</span>
								{/if}
							</div>
						</div>
					</div>

					<!-- CHART — prominent area chart -->
					{#if card.sessions.length > 1}
						<div class="px-2 pt-2 mb-2" use:chartAction={card}></div>
						<!-- Latest session summary -->
						{@const latest = card.sessions[0]}
						{@const best = bestSet(latest, card.exercise_category)}
						<div class="flex items-center justify-center gap-3 px-3 pb-2 text-[11px] text-base-content/50">
							<span class="font-medium text-base-content/70">Latest</span>
							<span>{latest.sets.length} set{latest.sets.length !== 1 ? 's' : ''}</span>
							{#if best}
								<span class="text-base-content/30">·</span>
								{#if isCardio}
									<span>{best.distance_miles ? `${best.distance_miles} mi` : ''}{best.duration_seconds ? ` ${Math.round(best.duration_seconds / 60)}min` : ''}</span>
								{:else}
									<span>Best: {effectiveWeight(best, card.exercise_category)} lb × {best.reps ?? 0}</span>
								{/if}
							{/if}
							<span class="text-base-content/30">·</span>
							<span>{card.sessions.length} session{card.sessions.length !== 1 ? 's' : ''}</span>
						</div>
					{:else}
						<div class="px-4 py-6 text-center">
							<p class="text-xs text-base-content/30">Chart available after 2+ sessions</p>
						</div>
					{/if}

					<!-- Session History Table — compact -->
					<div class="overflow-x-auto px-4 pb-3 border-t border-base-200">
						<table class="table table-xs">
							<thead>
								<tr class="text-[10px] text-base-content/50 uppercase tracking-wider">
									<th>Date</th>
									{#if isCardio}
										<th class="text-right">Duration</th>
										<th class="text-right">Dist</th>
									{:else}
										<th class="text-right">Best Set</th>
										<th class="text-right">Volume</th>
									{/if}
									<th class="text-center">Diff</th>
									<th class="text-center">Δ</th>
								</tr>
							</thead>
							<tbody>
								{#each card.sessions as session, i}
									{@const prev = i < card.sessions.length - 1 ? card.sessions[i + 1] : null}
									{@const t = trend(session, prev, card.exercise_category)}
									{@const pr = !isCardio && isPR(card, i)}
									<tr class="hover">
										<td class="text-xs whitespace-nowrap font-medium">
											{formatDate(session.date)}
											{#if pr}<span class="text-warning ml-0.5" title="Personal Record">🏆</span>{/if}
										</td>
										{#if isCardio}
											{@const fs = finalSet(session)}
											<td class="text-right text-xs">
												{fs?.duration_seconds ? `${Math.round(fs.duration_seconds / 60)}min` : '—'}
											</td>
											<td class="text-right text-xs">
												{fs?.distance_miles ? `${fs.distance_miles}mi` : '—'}
											</td>
										{:else}
											<td class="text-right text-xs font-mono">{formatSet(bestSet(session, card.exercise_category), card.exercise_category)}</td>
											<td class="text-right text-xs font-mono text-base-content/50">{totalVolume(session, card.exercise_category).toLocaleString()}</td>
										{/if}
										<td class="text-center">
											{#if session.difficulty}
												<span class="badge badge-xs {difficultyColors[session.difficulty] ?? ''}" title={difficultyLabels[session.difficulty]}>
													{difficultyLabels[session.difficulty]?.[0] ?? session.difficulty}
												</span>
											{:else}
												<span class="text-xs text-base-content/30">—</span>
											{/if}
										</td>
										<td class="text-center">
											{#if t.icon}
												<span class="iconify {t.icon} size-3.5 {t.class}"></span>
											{/if}
										</td>
									</tr>
								{/each}
							</tbody>
						</table>
					</div>

					{#if card.sessions.length > 0}
						{@const latest = card.sessions[0]}
						{#if latest.notes}
							<div class="px-4 pb-3 -mt-1">
								<p class="text-xs text-base-content/40 italic truncate" title={latest.notes}>
									<span class="iconify lucide--message-square size-3 inline mr-0.5"></span>
									{latest.notes}
								</p>
							</div>
						{/if}
					{/if}
				</div>
			</div>
		{/each}
	</div>

	<div class="text-center text-xs text-base-content/40 mt-4">
		Showing {filteredCards.length} exercise{filteredCards.length !== 1 ? 's' : ''} · Last 6 sessions each
	</div>
{/if}
