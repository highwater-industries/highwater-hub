<script lang="ts">
	import { onMount } from 'svelte';
	import {
		listFitnessUsers,
		getUserProgress,
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
	let filterCategory = $state('all');

	const difficultyLabels: Record<number, string> = { 1: 'Easy', 2: 'Light', 3: 'Moderate', 4: 'Hard', 5: 'Max' };
	const difficultyColors: Record<number, string> = { 1: 'badge-success', 2: 'badge-info', 3: 'badge-warning', 4: 'badge-error', 5: 'badge-error' };

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
			if (activeUser) await loadProgress();
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
		loadProgress();
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

	onMount(loadUsers);

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

	function bestSet(entry: ExerciseHistoryEntry): WorkoutSet | null {
		if (!entry.sets || entry.sets.length === 0) return null;
		let best = entry.sets[0];
		for (const s of entry.sets) {
			if ((s.weight ?? 0) > (best.weight ?? 0)) best = s;
			else if ((s.weight ?? 0) === (best.weight ?? 0) && (s.reps ?? 0) > (best.reps ?? 0)) best = s;
		}
		return best;
	}

	function totalVolume(entry: ExerciseHistoryEntry): number {
		return entry.sets.reduce((sum, s) => sum + (s.weight ?? 0) * (s.reps ?? 1), 0);
	}

	function formatSet(s: WorkoutSet | null): string {
		if (!s) return '—';
		if (s.weight !== undefined && s.reps !== undefined) return `${s.weight}lb × ${s.reps}`;
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

	function trend(current: ExerciseHistoryEntry, previous: ExerciseHistoryEntry | null): { icon: string; class: string } {
		if (!previous) return { icon: '', class: '' };
		const curFinal = finalSet(current);
		const prevFinal = finalSet(previous);
		if (!curFinal || !prevFinal) return { icon: '', class: '' };

		// Compare weight first, then reps
		const cw = curFinal.weight ?? 0;
		const pw = prevFinal.weight ?? 0;
		const cr = curFinal.reps ?? 0;
		const pr = prevFinal.reps ?? 0;

		if (cw > pw || (cw === pw && cr > pr)) return { icon: '↑', class: 'text-success' };
		if (cw < pw || (cw === pw && cr < pr)) return { icon: '↓', class: 'text-error' };
		return { icon: '=', class: 'text-warning' };
	}

	function isPR(card: ExerciseProgressCard, sessionIdx: number): boolean {
		const session = card.sessions[sessionIdx];
		const best = bestSet(session);
		if (!best || best.weight === undefined) return false;

		// Check all sessions (including those after, since sessions are ordered DESC)
		for (let i = 0; i < card.sessions.length; i++) {
			if (i === sessionIdx) continue;
			const otherBest = bestSet(card.sessions[i]);
			if (!otherBest) continue;
			if ((otherBest.weight ?? 0) > (best.weight ?? 0)) return false;
			if ((otherBest.weight ?? 0) === (best.weight ?? 0) && (otherBest.reps ?? 0) > (best.reps ?? 0)) return false;
		}
		return true;
	}

	function maxVolume(card: ExerciseProgressCard): number {
		return Math.max(...card.sessions.map(totalVolume), 1);
	}
</script>

<!-- HEADER -->
<div class="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-3 mb-4">
	<h1 class="text-2xl font-bold text-primary tracking-wide">// EXERCISE PROGRESS</h1>

	<div class="flex gap-2 items-center flex-wrap">
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

		<a href="/fitness" class="btn btn-ghost btn-sm">← Workouts</a>
	</div>
</div>

{#if loading}
	<div class="card bg-base-200 shadow-md border border-base-300 p-8 text-center">
		<span class="loading loading-dots loading-md text-primary"></span>
		<p class="text-sm opacity-60 mt-2">Loading progress...</p>
	</div>
{:else if error}
	<div class="card bg-base-200 shadow-md border border-base-300 p-8 text-center">
		<p class="text-error font-bold">{error}</p>
	</div>
{:else if !activeUser}
	<div class="card bg-base-200 shadow-md border border-base-300 p-8 text-center">
		<p class="opacity-60">No fitness user found. <a href="/fitness" class="link link-primary">Go to Fitness</a> to create one.</p>
	</div>
{:else if filteredCards.length === 0}
	<div class="card bg-base-200 shadow-md border border-base-300 p-8 text-center">
		<p class="opacity-60">No completed workouts yet.</p>
		<p class="text-xs opacity-40 mt-1">Complete a workout to start tracking progress.</p>
		<a href="/fitness" class="btn btn-primary btn-sm mt-3">Go to Workouts</a>
	</div>
{:else}
	<!-- PROGRESS CARDS GRID -->
	<div class="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
		{#each filteredCards as card}
			{@const isCardio = card.exercise_category === 'cardio'}
			{@const maxVol = maxVolume(card)}
			<div class="card bg-base-100 shadow-md border border-base-300 overflow-hidden">
				<div class="card-body p-4 gap-2">
					<!-- Card Header -->
					<div class="flex items-center justify-between">
						<div>
							<h3 class="font-bold text-primary text-sm tracking-wide">{card.exercise_name}</h3>
							<div class="flex gap-1 mt-0.5">
								<span class="badge badge-xs badge-outline">{card.exercise_category}</span>
								{#if card.muscle_group}
									<span class="badge badge-xs badge-ghost">{card.muscle_group}</span>
								{/if}
								{#if card.equipment}
									<span class="badge badge-xs badge-ghost">{card.equipment}</span>
								{/if}
							</div>
						</div>
						<span class="text-xs opacity-40">{card.sessions.length} sessions</span>
					</div>

					<!-- Volume Sparkline -->
					{#if !isCardio && card.sessions.length > 1}
						<div class="flex items-end gap-0.5 h-8 mt-1" title="Volume trend">
							{#each [...card.sessions].reverse() as session}
								{@const vol = totalVolume(session)}
								{@const pct = Math.max((vol / maxVol) * 100, 4)}
								<div
									class="flex-1 rounded-t transition-all"
									class:bg-success={pct > 80}
									class:bg-warning={pct > 50 && pct <= 80}
									class:bg-info={pct <= 50}
									style="height: {pct}%"
									title="{formatDate(session.date)}: {vol.toLocaleString()} vol"
								></div>
							{/each}
						</div>
					{/if}

					<!-- Session Table -->
					<div class="overflow-x-auto">
						<table class="table table-xs table-zebra">
							<thead>
								<tr class="text-xs opacity-60">
									<th>Date</th>
									{#if isCardio}
										<th class="text-right">Duration</th>
										<th class="text-right">Dist</th>
									{:else}
										<th class="text-right">Final Set</th>
										<th class="text-right">Best Set</th>
									{/if}
									<th class="text-center">Diff</th>
									<th class="text-center">RTP</th>
									<th class="text-center">Δ</th>
								</tr>
							</thead>
							<tbody>
								{#each card.sessions as session, i}
									{@const prev = i < card.sessions.length - 1 ? card.sessions[i + 1] : null}
									{@const t = trend(session, prev)}
									{@const pr = !isCardio && isPR(card, i)}
									<tr class="hover">
										<td class="font-semibold text-xs whitespace-nowrap">
											{formatDate(session.date)}
											{#if pr}<span class="text-warning" title="Personal Record">🏆</span>{/if}
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
											<td class="text-right text-xs font-mono">{formatSet(finalSet(session))}</td>
											<td class="text-right text-xs font-mono opacity-60">{formatSet(bestSet(session))}</td>
										{/if}
										<td class="text-center">
											{#if session.difficulty}
												<span class="badge badge-xs {difficultyColors[session.difficulty] ?? ''}" title={difficultyLabels[session.difficulty]}>
													{difficultyLabels[session.difficulty]?.[0] ?? session.difficulty}
												</span>
											{:else}
												<span class="text-xs opacity-30">—</span>
											{/if}
										</td>
										<td class="text-center">
											{#if session.ready_to_progress}
												<span class="text-success font-bold text-xs" title="Ready to progress">✓</span>
											{:else}
												<span class="text-xs opacity-30">—</span>
											{/if}
										</td>
										<td class="text-center">
											{#if t.icon}
												<span class="{t.class} font-bold text-sm">{t.icon}</span>
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
							<div class="text-xs opacity-40 italic mt-1 truncate" title={latest.notes}>
								💬 {latest.notes}
							</div>
						{/if}
					{/if}
				</div>
			</div>
		{/each}
	</div>

	<div class="text-center text-xs opacity-40 mt-4">
		Showing {filteredCards.length} exercise{filteredCards.length !== 1 ? 's' : ''} · Last 6 sessions each
	</div>
{/if}
