<script lang="ts">
	import { onMount, tick } from 'svelte';
	import { goto } from '$app/navigation';
	import PageHeader from '$lib/components/PageHeader.svelte';
	import {
		listFitnessUsers,
		createFitnessUser,
		listWorkouts,
		createWorkout,
		deleteWorkout,
		logBodyweight,
		getLatestBodyweight,
		listBodyweightHistory,
		deleteBodyweight,
		getExerciseHistory,
		getUserProgress,
		type FitnessUser,
		type WorkoutSummary,
		type BodyweightEntry,
		type ExerciseProgressCard,
		type ExerciseHistoryEntry,
		type WorkoutSet
	} from '$lib/api';

	let users: FitnessUser[] = $state([]);
	let activeUser: FitnessUser | null = $state(null);
	let workouts: WorkoutSummary[] = $state([]);
	let totalWorkouts = $state(0);
	let loading = $state(true);
	let loadingWorkouts = $state(false);

	// New user form
	let showNewUser = $state(false);
	let newUserName = $state('');

	// Confirm-delete state
	let confirmDeleteWorkoutId: number | null = $state(null);

	// New workout options
	let workoutDate = $state(new Date().toISOString().slice(0, 10));
	let workoutIsDeload = $state(false);
	let showNewWorkoutOptions = $state(false);

	// Bodyweight state
	let latestBodyweight: BodyweightEntry | null = $state(null);
	let bodyweightHistory: BodyweightEntry[] = $state([]);
	let showBodyweightHistory = $state(false);
	let newWeight = $state('');
	let newWeightNotes = $state('');
	let loggingWeight = $state(false);

	async function loadUsers() {
		try {
			users = await listFitnessUsers();
			const savedId = localStorage.getItem('fitness_user_id');
			if (savedId) {
				activeUser = users.find((u) => u.id === Number(savedId)) ?? null;
			}
			if (!activeUser && users.length > 0) {
				selectUser(users[0]);
			}
			if (activeUser) await loadWorkouts();
			if (activeUser) await loadBodyweight();
		} catch (e) {
			console.error('Failed to load users', e);
		} finally {
			loading = false;
		}
	}

	function selectUser(user: FitnessUser) {
		activeUser = user;
		localStorage.setItem('fitness_user_id', String(user.id));
		loadWorkouts();
		loadBodyweight();
	}

	async function loadBodyweight() {
		if (!activeUser) return;
		try {
			latestBodyweight = await getLatestBodyweight(activeUser.id);
			if (showBodyweightHistory) {
				bodyweightHistory = await listBodyweightHistory(activeUser.id, 30) ?? [];
			}
		} catch (e) {
			console.error('Failed to load bodyweight', e);
		}
	}

	async function handleLogWeight() {
		const w = Number(newWeight);
		if (!activeUser || !w || isNaN(w) || w <= 0) return;
		loggingWeight = true;
		try {
			await logBodyweight(activeUser.id, w, undefined, newWeightNotes.trim() || undefined);
			newWeight = '';
			newWeightNotes = '';
			await loadBodyweight();
		} catch (e) {
			console.error('Failed to log bodyweight', e);
		} finally {
			loggingWeight = false;
		}
	}

	async function handleDeleteWeight(id: number) {
		try {
			await deleteBodyweight(id);
			await loadBodyweight();
		} catch (e) {
			console.error('Failed to delete bodyweight entry', e);
		}
	}

	async function toggleBodyweightHistory() {
		showBodyweightHistory = !showBodyweightHistory;
		if (showBodyweightHistory && activeUser) {
			bodyweightHistory = await listBodyweightHistory(activeUser.id, 30) ?? [];
		}
	}

	async function loadWorkouts() {
		if (!activeUser) return;
		loadingWorkouts = true;
		try {
			const res = await listWorkouts(activeUser.id, 0, 10);
			workouts = res.items ?? [];
			totalWorkouts = res.total;
		} catch (e) {
			console.error('Failed to load workouts', e);
		} finally {
			loadingWorkouts = false;
		}
	}

	async function handleCreateUser() {
		if (!newUserName.trim()) return;
		try {
			const user = await createFitnessUser(newUserName.trim());
			users = [...users, user];
			selectUser(user);
			newUserName = '';
			showNewUser = false;
		} catch (e) {
			console.error('Failed to create user', e);
		}
	}

	async function handleStartWorkout() {
		if (!activeUser) return;
		try {
			const today = new Date().toISOString().slice(0, 10);
			const startedAt = workoutDate !== today
				? new Date(workoutDate + 'T10:00:00Z').toISOString()
				: undefined;
			const workout = await createWorkout(activeUser.id, startedAt, workoutIsDeload);
			goto(`/fitness/workout/${workout.id}`);
		} catch (e) {
			console.error('Failed to create workout', e);
		}
	}

	function requestDeleteWorkout(id: number) {
		confirmDeleteWorkoutId = id;
	}

	function cancelDeleteWorkout() {
		confirmDeleteWorkoutId = null;
	}

	async function handleDeleteWorkout(id: number) {
		confirmDeleteWorkoutId = null;
		try {
			await deleteWorkout(id);
			workouts = workouts.filter((w) => w.id !== id);
			totalWorkouts--;
		} catch (e) {
			console.error('Failed to delete workout', e);
		}
	}

	function formatDate(iso: string): string {
		const d = new Date(iso);
		return d.toLocaleDateString('en-US', { weekday: 'short', month: 'short', day: 'numeric' });
	}

	function formatTime(iso: string): string {
		const d = new Date(iso);
		return d.toLocaleTimeString('en-US', { hour: 'numeric', minute: '2-digit' });
	}

	function duration(started: string, completed?: string): string {
		if (!completed) return '';
		const ms = new Date(completed).getTime() - new Date(started).getTime();
		const mins = Math.round(ms / 60000);
		if (mins < 60) return `${mins}m`;
		return `${Math.floor(mins / 60)}h ${mins % 60}m`;
	}

	const categoryIcons: Record<string, string> = {
		strength: 'lucide--dumbbell',
		bodyweight: 'lucide--person-standing',
		cardio: 'lucide--heart-pulse'
	};

	function exerciseCategories(names?: string): string[] {
		if (!names) return [];
		return names.split(', ').slice(0, 4);
	}

	interface ExerciseDetail {
		name: string;
		sets: number;
		total_reps: number;
		max_weight: number;
		reps_list: number[];
	}

	function parseExerciseDetails(raw?: string): ExerciseDetail[] {
		if (!raw || raw === '[]') return [];
		try {
			return JSON.parse(raw) as ExerciseDetail[];
		} catch {
			return [];
		}
	}

	/** Format reps_list like "3×8" or "4×3, 1×5" */
	function formatRepPattern(reps: number[]): string {
		if (!reps || reps.length === 0) return '0r';
		const groups: { count: number; reps: number }[] = [];
		for (const r of reps) {
			const last = groups[groups.length - 1];
			if (last && last.reps === r) {
				last.count++;
			} else {
				groups.push({ count: 1, reps: r });
			}
		}
		return groups.map(g => `${g.count}×${g.reps}`).join(', ');
	}

	// Derived: separate active and completed workouts
	let activeWorkouts = $derived(workouts.filter((w) => !w.completed_at));
	let completedWorkouts = $derived(workouts.filter((w) => w.completed_at));

	// ── Per-exercise chart on workout cards ──
	let progressCards: ExerciseProgressCard[] = $state([]);
	let selectedExerciseCharts: Record<number, string | null> = $state({});
	let exerciseChartRefs: Record<number, HTMLDivElement | undefined> = $state({});
	let exerciseChartInstances: Record<number, any> = {};
	let ApexChartsClass: any = null;
	let loadingChartId: number | null = $state(null);

	function exerciseProgressMap(): Map<string, ExerciseProgressCard> {
		const map = new Map<string, ExerciseProgressCard>();
		for (const c of progressCards) {
			map.set(c.exercise_name, c);
		}
		return map;
	}

	async function toggleExerciseChart(workoutId: number, exerciseName: string) {
		if (selectedExerciseCharts[workoutId] === exerciseName) {
			selectedExerciseCharts[workoutId] = null;
			if (exerciseChartInstances[workoutId]) {
				exerciseChartInstances[workoutId].destroy();
				delete exerciseChartInstances[workoutId];
			}
			return;
		}

		selectedExerciseCharts[workoutId] = exerciseName;
		loadingChartId = workoutId;

		try {
			// Load ApexCharts if needed
			if (!ApexChartsClass) {
				ApexChartsClass = (await import('apexcharts')).default;
			}

			// Load progress data if not yet loaded
			if (progressCards.length === 0 && activeUser) {
				progressCards = await getUserProgress(activeUser.id, 6);
			}

			await tick();
			setTimeout(() => renderExerciseChart(workoutId, exerciseName), 30);
		} catch (e) {
			console.error('Failed to load chart', e);
		} finally {
			loadingChartId = null;
		}
	}

	function renderExerciseChart(workoutId: number, exerciseName: string) {
		const ref = exerciseChartRefs[workoutId];
		if (!ApexChartsClass || !ref) return;

		if (exerciseChartInstances[workoutId]) {
			exerciseChartInstances[workoutId].destroy();
			delete exerciseChartInstances[workoutId];
		}

		const pMap = exerciseProgressMap();
		const card = pMap.get(exerciseName);
		if (!card || card.sessions.length < 2) return;

		const isCardio = card.exercise_category === 'cardio';
		const reversed = [...card.sessions].reverse();

		let primaryData: number[];
		let primaryLabel: string;
		let secondaryData: number[];
		let secondaryLabel: string;

		if (isCardio) {
			primaryData = reversed.map(s => {
				const fs = s.sets.length > 0 ? s.sets[s.sets.length - 1] : null;
				return fs?.distance_miles ?? 0;
			});
			primaryLabel = 'Distance (mi)';
			secondaryData = reversed.map(s => {
				const fs = s.sets.length > 0 ? s.sets[s.sets.length - 1] : null;
				return fs?.duration_seconds ? Math.round(fs.duration_seconds / 60) : 0;
			});
			secondaryLabel = 'Duration (min)';
		} else {
			primaryData = reversed.map(s => {
				const best = cardBestSet(s, card.exercise_category);
				return best ? cardEffWeight(best, card.exercise_category) : 0;
			});
			primaryLabel = 'Best Weight (lb)';
			secondaryData = reversed.map(s =>
				s.sets.reduce((sum: number, st: WorkoutSet) => sum + cardEffWeight(st, card.exercise_category) * (st.reps ?? 1), 0)
			);
			secondaryLabel = 'Volume (lb)';
		}

		const dates = reversed.map(s => {
			const d = new Date(s.date);
			return d.toLocaleDateString('en-US', { month: 'short', day: 'numeric' });
		});

		const instance = new ApexChartsClass(ref, {
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
		exerciseChartInstances[workoutId] = instance;
	}

	function cardEffWeight(s: WorkoutSet, category: string): number {
		const w = s.weight ?? 0;
		if (category === 'bodyweight') return (latestBodyweight?.weight_lbs ?? 0) + w;
		return w;
	}

	function cardBestSet(entry: ExerciseHistoryEntry, category: string): WorkoutSet | null {
		if (!entry.sets || entry.sets.length === 0) return null;
		let best = entry.sets[0];
		for (const s of entry.sets) {
			const ew = cardEffWeight(s, category);
			const bw = cardEffWeight(best, category);
			if (ew > bw || (ew === bw && (s.reps ?? 0) > (best.reps ?? 0))) best = s;
		}
		return best;
	}

	onMount(loadUsers);
</script>

<PageHeader title="Fitness" breadcrumbs={[{ label: 'Fitness' }]}>
	{#snippet actions()}
		<a href="/fitness/progress" class="btn btn-ghost btn-sm gap-1">
			<span class="iconify lucide--trending-up size-4"></span>
			Progress
		</a>
		{#if activeUser}
			<span class="text-sm text-base-content/60">{totalWorkouts} workouts</span>
		{/if}
	{/snippet}
</PageHeader>

{#if loading}
	<div class="card bg-base-100 shadow-sm p-8 text-center">
		<span class="loading loading-dots loading-md text-primary"></span>
		<p class="text-sm text-base-content/60 mt-2">Loading...</p>
	</div>
{:else}

	<!-- User Picker -->
	<div class="flex flex-wrap items-center gap-2 mb-6">
		{#each users as user}
			<button
				class="btn btn-sm"
				class:btn-primary={activeUser?.id === user.id}
				class:btn-outline={activeUser?.id !== user.id}
				onclick={() => selectUser(user)}
			>
				{user.name}
			</button>
		{/each}
		{#if showNewUser}
			<form class="join" onsubmit={(e) => { e.preventDefault(); handleCreateUser(); }}>
				<input
					type="text"
					class="input input-bordered input-sm join-item w-32"
					placeholder="Name..."
					bind:value={newUserName}
				/>
				<button type="submit" class="btn btn-sm btn-primary join-item">Add</button>
				<button type="button" class="btn btn-sm btn-ghost join-item" onclick={() => { showNewUser = false; newUserName = ''; }}>✕</button>
			</form>
		{:else}
			<button class="btn btn-sm btn-ghost" onclick={() => (showNewUser = true)}>+ User</button>
		{/if}
	</div>

	{#if activeUser}

		<!-- ═══════════════════════════════════════════════════════ -->
		<!-- START WORKOUT ZONE — visually separated at the top     -->
		<!-- ═══════════════════════════════════════════════════════ -->
		<div class="card bg-base-100 shadow-sm mb-6 border-l-4 border-primary">
			<div class="card-body p-5 gap-4">
				<div class="flex items-center gap-3">
					<div class="bg-primary/10 rounded-box flex items-center p-2">
						<span class="iconify lucide--dumbbell size-5 text-primary"></span>
					</div>
					<div class="flex-1">
						<h2 class="font-semibold text-lg">New Workout</h2>
						<p class="text-sm text-base-content/60">Start a fresh session to log exercises and sets</p>
					</div>
				</div>

				<!-- Options row -->
				<div class="flex flex-wrap items-end gap-4">
					<div class="space-y-1">
						<label class="text-xs font-medium text-base-content/60 tracking-wide uppercase" for="workout-date">Date</label>
						<input
							id="workout-date"
							type="date"
							class="input input-bordered input-sm w-40"
							bind:value={workoutDate}
						/>
					</div>
					<label class="flex items-center gap-2 cursor-pointer pb-1">
						<input
							type="checkbox"
							class="checkbox checkbox-sm checkbox-warning"
							bind:checked={workoutIsDeload}
						/>
						<span class="text-sm text-base-content/70">Deload</span>
					</label>
					<div class="flex-1"></div>
					<button class="btn btn-primary gap-2" onclick={handleStartWorkout}>
						<span class="iconify lucide--play size-4"></span>
						Start Workout
					</button>
				</div>
			</div>
		</div>

		<!-- ═══════════════════════════════════════════════════════ -->
		<!-- BODYWEIGHT — compact stat-style card                   -->
		<!-- ═══════════════════════════════════════════════════════ -->
		<div class="card bg-base-100 shadow-sm mb-6">
			<div class="card-body p-4 gap-3">
				<div class="flex items-center justify-between">
					<div class="flex items-center gap-3">
						<div class="bg-base-200 rounded-box flex items-center p-1.5">
							<span class="iconify lucide--scale size-4"></span>
						</div>
						<div>
							<p class="text-xs font-medium text-base-content/60 tracking-wide uppercase">Bodyweight</p>
							{#if latestBodyweight}
								<div class="flex items-baseline gap-1.5 mt-0.5">
									<span class="text-2xl font-semibold">{latestBodyweight.weight_lbs}</span>
									<span class="text-sm text-base-content/50">lbs</span>
									<span class="text-xs text-base-content/40 ml-1">
										{new Date(latestBodyweight.logged_at).toLocaleDateString('en-US', { month: 'short', day: 'numeric' })}
									</span>
								</div>
							{:else}
								<p class="text-sm text-base-content/40 mt-0.5">Not logged yet</p>
							{/if}
						</div>
					</div>
					<button class="btn btn-ghost btn-xs btn-square" onclick={toggleBodyweightHistory} title="History">
						<span class="iconify size-4 {showBodyweightHistory ? 'lucide--chevron-up' : 'lucide--history'}"></span>
					</button>
				</div>

				<!-- Quick Log Form -->
				<form class="flex items-center gap-2" onsubmit={(e) => { e.preventDefault(); handleLogWeight(); }}>
					<label class="input input-bordered input-sm w-24 flex items-center gap-1">
						<span class="iconify lucide--weight size-3.5 text-base-content/40"></span>
						<input
							type="number"
							inputmode="decimal"
							step="0.1"
							class="grow w-full"
							placeholder="lbs"
							bind:value={newWeight}
						/>
					</label>
					<input
						type="text"
						class="input input-bordered input-sm flex-1 max-w-48"
						placeholder="Notes (optional)"
						bind:value={newWeightNotes}
					/>
					<button type="submit" class="btn btn-primary btn-sm" disabled={loggingWeight || !newWeight}>
						{loggingWeight ? '...' : 'Log'}
					</button>
				</form>

				<!-- History -->
				{#if showBodyweightHistory && bodyweightHistory.length > 0}
					<div class="overflow-x-auto border-t border-base-300 pt-3">
						<table class="table table-xs">
							<thead>
								<tr class="text-xs text-base-content/60">
									<th>Date</th>
									<th>Weight</th>
									<th>Notes</th>
									<th></th>
								</tr>
							</thead>
							<tbody>
								{#each bodyweightHistory as entry}
									<tr class="hover">
										<td class="text-xs">
											{new Date(entry.logged_at).toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: '2-digit' })}
										</td>
										<td class="font-mono font-semibold text-sm">{entry.weight_lbs}</td>
										<td class="text-xs text-base-content/50">{entry.notes ?? ''}</td>
										<td>
											<button class="btn btn-ghost btn-xs text-error" onclick={() => handleDeleteWeight(entry.id)}>✕</button>
										</td>
									</tr>
								{/each}
							</tbody>
						</table>
					</div>
				{/if}
			</div>
		</div>

		<!-- ═══════════════════════════════════════════════════════ -->
		<!-- ACTIVE WORKOUTS — prominent if any exist               -->
		<!-- ═══════════════════════════════════════════════════════ -->
		{#if activeWorkouts.length > 0}
			<div class="mb-6">
				<p class="text-xs font-medium text-base-content/60 tracking-wide uppercase mb-2">Active Sessions</p>
				<div class="flex flex-col gap-3">
					{#each activeWorkouts as w}
						<a
							href="/fitness/workout/{w.id}"
							class="card bg-base-100 shadow-sm border border-warning/30 hover:shadow-md hover:-translate-y-0.5 transition-all no-underline"
						>
							<div class="card-body p-4 flex-row items-center gap-4">
								<div class="bg-warning/10 rounded-box flex items-center p-2">
									<span class="iconify lucide--timer size-5 text-warning"></span>
								</div>
								<div class="flex-1 min-w-0">
									<div class="font-semibold">{formatDate(w.started_at)}</div>
									<div class="text-sm text-base-content/60">
										{w.exercise_count} exercise{w.exercise_count !== 1 ? 's' : ''}
										· {w.set_count} set{w.set_count !== 1 ? 's' : ''}
									</div>
									{#if w.exercise_names}
										<div class="text-xs text-base-content/40 truncate mt-0.5">{w.exercise_names}</div>
									{/if}
								</div>
								<span class="badge badge-soft badge-warning badge-sm">In Progress</span>
							</div>
						</a>
					{/each}
				</div>
			</div>
		{/if}

		<!-- ═══════════════════════════════════════════════════════ -->
		<!-- COMPLETED WORKOUTS — clean timeline-style cards         -->
		<!-- ═══════════════════════════════════════════════════════ -->
		{#if loadingWorkouts}
			<div class="text-center py-8">
				<span class="loading loading-dots loading-md text-primary"></span>
			</div>
		{:else if completedWorkouts.length === 0 && activeWorkouts.length === 0}
			<div class="card bg-base-100 shadow-sm p-8 text-center">
				<div class="flex justify-center mb-3">
					<div class="bg-base-200 rounded-box flex items-center p-3">
						<span class="iconify lucide--dumbbell size-8 text-base-content/30"></span>
					</div>
				</div>
				<p class="text-lg font-semibold text-base-content/60">No workouts yet</p>
				<p class="text-sm text-base-content/40 mt-1">Start your first session above to begin tracking</p>
			</div>
		{:else if completedWorkouts.length > 0}
			<p class="text-xs font-medium text-base-content/60 tracking-wide uppercase mb-3">Completed Workouts</p>
			<div class="grid grid-cols-1 md:grid-cols-2 gap-4">
				{#each completedWorkouts as w}
					{@const dur = duration(w.started_at, w.completed_at)}
				{@const exercises = exerciseCategories(w.exercise_names)}
					{@const details = parseExerciseDetails(w.exercise_details)}
					<div class="card bg-base-100 shadow-sm hover:shadow-md transition-all overflow-hidden group">
						<!-- Colored top strip -->
						<div class="h-1 bg-gradient-to-r from-success/60 to-success/20"></div>
						<div class="card-body p-4 gap-3">
							<!-- Date + badges row -->
							<div class="flex items-center justify-between gap-2">
								<a href="/fitness/workout/{w.id}" class="flex items-center gap-2 no-underline hover:underline">
									<span class="font-semibold">{formatDate(w.started_at)}</span>
									{#if w.is_deload}
										<span class="badge badge-soft badge-warning badge-xs">Deload</span>
									{/if}
								</a>
								<div class="flex items-center gap-1">
									<a href="/fitness/workout/{w.id}" class="btn btn-ghost btn-xs" title="View workout">
										<span class="iconify lucide--eye size-3.5"></span>
									</a>
									{#if confirmDeleteWorkoutId === w.id}
										<button
											class="btn btn-error btn-xs"
											onclick={(e) => { e.stopPropagation(); e.preventDefault(); handleDeleteWorkout(w.id); }}
										>Delete</button>
										<button
											class="btn btn-ghost btn-xs"
											onclick={(e) => { e.stopPropagation(); e.preventDefault(); cancelDeleteWorkout(); }}
										>Cancel</button>
									{:else}
										<button
											class="btn btn-ghost btn-xs text-error/30 hover:text-error opacity-0 group-hover:opacity-100 transition-opacity"
											onclick={(e) => { e.stopPropagation(); e.preventDefault(); requestDeleteWorkout(w.id); }}
											title="Delete workout"
										>
											<span class="iconify lucide--trash-2 size-3.5"></span>
										</button>
									{/if}
								</div>
							</div>

							<!-- Stats grid -->
							<div class="grid grid-cols-3 gap-2">
								<div class="bg-base-200/50 rounded-lg p-2 text-center">
									<p class="text-lg font-bold text-primary">{w.exercise_count}</p>
									<p class="text-[10px] text-base-content/50 uppercase tracking-wide">Exercises</p>
								</div>
								<div class="bg-base-200/50 rounded-lg p-2 text-center">
									<p class="text-lg font-bold text-primary">{w.set_count}</p>
									<p class="text-[10px] text-base-content/50 uppercase tracking-wide">Sets</p>
								</div>
								<div class="bg-base-200/50 rounded-lg p-2 text-center">
									<p class="text-lg font-bold text-primary">{dur || '—'}</p>
									<p class="text-[10px] text-base-content/50 uppercase tracking-wide">Duration</p>
								</div>
							</div>

							<!-- Exercise pills — clickable to show chart -->
							{#if exercises.length > 0}
								<div class="flex flex-wrap gap-1.5 mt-1">
									{#each exercises as name}
										<button
											class="badge badge-sm cursor-pointer transition-colors {selectedExerciseCharts[w.id] === name ? 'badge-primary' : 'badge-soft badge-ghost hover:badge-primary/20'}"
											onclick={(e) => { e.preventDefault(); e.stopPropagation(); toggleExerciseChart(w.id, name); }}
										>
											<span class="iconify lucide--bar-chart-2 size-3 mr-0.5"></span>
											{name}
										</button>
									{/each}
									{#if w.exercise_names && w.exercise_names.split(', ').length > 4}
										{@const remaining = w.exercise_names.split(', ').slice(4)}
										{#each remaining as name}
											<button
												class="badge badge-sm cursor-pointer transition-colors {selectedExerciseCharts[w.id] === name ? 'badge-primary' : 'badge-soft badge-ghost hover:badge-primary/20'}"
												onclick={(e) => { e.preventDefault(); e.stopPropagation(); toggleExerciseChart(w.id, name); }}
											>
												<span class="iconify lucide--bar-chart-2 size-3 mr-0.5"></span>
												{name}
											</button>
										{/each}
									{/if}
								</div>
							{/if}

							<!-- Per-exercise breakdown -->
							{#if details.length > 0}
								<div class="grid gap-1 mt-1">
									{#each details as d}
										<div class="flex items-center gap-2 text-[11px] leading-tight px-0.5">
											<span class="text-base-content/70 font-medium truncate min-w-0 flex-1">{d.name}</span>
											<span class="text-base-content/50 tabular-nums whitespace-nowrap shrink-0">
												{formatRepPattern(d.reps_list)}{#if d.max_weight > 0}
													<span class="text-base-content/30 mx-0.5">@</span>{d.max_weight}lb
												{/if}
											</span>
										</div>
									{/each}
								</div>
							{/if}

							<!-- Inline exercise chart -->
							{#if selectedExerciseCharts[w.id] && (exercises.includes(selectedExerciseCharts[w.id] ?? '') || (w.exercise_names && w.exercise_names.split(', ').includes(selectedExerciseCharts[w.id] ?? '')))}
								<div class="border-t border-base-200 pt-3 mt-1 -mx-1">
									{#if loadingChartId === w.id}
										<div class="flex justify-center py-6">
											<span class="loading loading-dots loading-sm text-primary"></span>
										</div>
									{:else}
										{@const pMap = exerciseProgressMap()}
										{@const card = pMap.get(selectedExerciseCharts[w.id] ?? '')}
										{#if card && card.sessions.length >= 2}
											<div class="mb-2" bind:this={exerciseChartRefs[w.id]}></div>
										{:else if card && card.sessions.length < 2}
											<div class="text-center py-4">
												<p class="text-xs text-base-content/40">Need 2+ sessions for a chart</p>
											</div>
										{:else}
											<div class="text-center py-4">
												<p class="text-xs text-base-content/40">No progress data for this exercise yet</p>
											</div>
										{/if}
									{/if}
								</div>
							{/if}
						</div>
					</div>
				{/each}
			</div>
		{/if}
	{/if}
{/if}
