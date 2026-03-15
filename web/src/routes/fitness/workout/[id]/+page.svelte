<script lang="ts">
	import { onMount, tick } from 'svelte';
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import PageHeader from '$lib/components/PageHeader.svelte';
	import {
		getWorkout,
		completeWorkout,
		updateWorkoutMeta,
		listExercises,
		addExerciseToWorkout,
		updateWorkoutExercise,
		removeWorkoutExercise,
		addSet,
		updateSet,
		deleteSet,
		getExerciseHistory,
		getLatestBodyweight,
		type WorkoutDetail,
		type WorkoutExerciseDetail,
		type WorkoutSet,
		type Exercise,
		type ExerciseHistoryEntry,
		type ListResponse
	} from '$lib/api';

	let workout: WorkoutDetail | null = $state(null);
	let loading = $state(true);
	let error = $state('');

	// Exercise picker
	let showPicker = $state(false);
	let exerciseSearch = $state('');
	let exerciseCategory = $state('');
	let availableExercises: Exercise[] = $state([]);
	let loadingExercises = $state(false);

	// Exercise history context
	let historyMap: Record<number, ExerciseHistoryEntry[]> = $state({});
	let expandedHistory: number | null = $state(null);

	// Completion modal
	let showComplete = $state(false);
	let completionNotes = $state('');

	// User bodyweight for effective weight calculations
	let userBodyweight: number | null = $state(null);

	$effect(() => {
		const id = Number($page.params.id);
		if (id) loadWorkout(id);
	});

	async function loadWorkout(id: number, soft = false) {
		if (!soft) {
			loading = true;
			error = '';
		}
		try {
			workout = await getWorkout(id);
			// Load bodyweight for effective weight display
			if (workout?.user_id) {
				try {
					const bw = await getLatestBodyweight(workout.user_id);
					userBodyweight = bw?.weight_lbs ?? null;
				} catch { /* non-fatal */ }
			}
		} catch (e) {
			error = 'Workout not found';
		} finally {
			loading = false;
		}
	}

	// ── Exercise Picker ──

	async function searchExercises() {
		loadingExercises = true;
		try {
			const userId = workout?.user_id;
			const res: ListResponse<Exercise> = await listExercises({
				search: exerciseSearch || undefined,
				category: exerciseCategory || undefined,
				user_id: userId,
				limit: 50
			});
			availableExercises = res.items;
		} catch (e) {
			console.error('Failed to search exercises', e);
		} finally {
			loadingExercises = false;
		}
	}

	async function handleAddExercise(exercise: Exercise) {
		if (!workout) return;
		try {
			await addExerciseToWorkout(workout.id, exercise.id);
			await loadWorkout(workout.id, true);
			showPicker = false;
			exerciseSearch = '';
			exerciseCategory = '';
			// Scroll to the new exercise (last one)
			await tick();
			const cards = document.querySelectorAll('[data-exercise-card]');
			cards[cards.length - 1]?.scrollIntoView({ behavior: 'smooth', block: 'center' });
		} catch (e) {
			console.error('Failed to add exercise', e);
		}
	}

	function openPicker() {
		showPicker = true;
		searchExercises();
	}

	// ── Sets ──

	async function handleAddSet(we: WorkoutExerciseDetail) {
		if (!workout) return;
		const isCardio = we.exercise_category === 'cardio';
		try {
			let newSet: WorkoutSet;
			if (isCardio) {
				newSet = await addSet(we.id, {});
			} else {
				// Pre-fill from last set in this exercise if available
				const lastSet = we.sets.length > 0 ? we.sets[we.sets.length - 1] : null;
				newSet = await addSet(we.id, {
					reps: lastSet?.reps,
					weight: lastSet?.weight
				});
			}
			// Optimistic local update — push the new set without full reload
			const ex = workout.exercises.find((e) => e.id === we.id);
			if (ex) {
				ex.sets = [...ex.sets, newSet];
				workout = { ...workout }; // trigger reactivity
			}
			// Scroll to the new set row
			await tick();
			const row = document.querySelector(`[data-set-id="${newSet.id}"]`);
			row?.scrollIntoView({ behavior: 'smooth', block: 'center' });
		} catch (e) {
			console.error('Failed to add set', e);
		}
	}

	async function handleUpdateSet(s: WorkoutSet, field: string, value: string) {
		if (!workout) return;
		const numVal = value === '' ? undefined : Number(value);

		// Optimistic local update — keep local state in sync immediately
		const ex = workout.exercises.find((e) => e.sets.some((st) => st.id === s.id));
		if (ex) {
			const localSet = ex.sets.find((st) => st.id === s.id);
			if (localSet) {
				(localSet as any)[field] = numVal;
				workout = { ...workout }; // trigger reactivity
			}
		}

		try {
			await updateSet(s.id, { [field]: numVal });
		} catch (e) {
			console.error('Failed to update set', e);
		}
	}

	async function handleDeleteSet(setId: number) {
		if (!workout) return;
		// Optimistic local removal
		for (const ex of workout.exercises) {
			const idx = ex.sets.findIndex((s) => s.id === setId);
			if (idx !== -1) {
				ex.sets = ex.sets.filter((s) => s.id !== setId);
				// Renumber remaining sets
				ex.sets.forEach((s, i) => (s.set_number = i + 1));
				workout = { ...workout };
				break;
			}
		}
		try {
			await deleteSet(setId);
		} catch (e) {
			console.error('Failed to delete set', e);
			// Reload to restore correct state on error
			if (workout) await loadWorkout(workout.id, true);
		}
	}

	// ── Exercise metadata ──

	async function handleUpdateExercise(
		weId: number,
		updates: { notes?: string; difficulty?: number; ready_to_progress?: boolean }
	) {
		if (!workout) return;
		try {
			await updateWorkoutExercise(weId, updates);
		} catch (e) {
			console.error('Failed to update exercise', e);
		}
	}

	// Confirm-delete state
	let confirmDeleteId: number | null = $state(null);

	function requestRemoveExercise(weId: number) {
		confirmDeleteId = weId;
	}

	function cancelRemoveExercise() {
		confirmDeleteId = null;
	}

	async function handleRemoveExercise(weId: number) {
		if (!workout) return;
		confirmDeleteId = null;
		// Optimistic local removal
		workout.exercises = workout.exercises.filter((e) => e.id !== weId);
		workout = { ...workout };
		try {
			await removeWorkoutExercise(weId);
		} catch (e) {
			console.error('Failed to remove exercise', e);
			if (workout) await loadWorkout(workout.id, true);
		}
	}

	// ── History ──

	async function toggleHistory(we: WorkoutExerciseDetail) {
		if (expandedHistory === we.id) {
			expandedHistory = null;
			return;
		}
		expandedHistory = we.id;
		if (!historyMap[we.exercise_id] && workout) {
			try {
				const entries = await getExerciseHistory(we.exercise_id, workout.user_id, 6);
				historyMap[we.exercise_id] = entries;
			} catch (e) {
				console.error('Failed to load history', e);
			}
		}
	}

	// ── Complete ──

	async function handleComplete() {
		if (!workout) return;
		try {
			await completeWorkout(workout.id, completionNotes || undefined);
			showComplete = false;
			goto('/fitness');
		} catch (e) {
			console.error('Failed to complete workout', e);
		}
	}

	// ── Helpers ──

	function formatDate(iso: string): string {
		return new Date(iso).toLocaleDateString('en-US', { weekday: 'short', month: 'short', day: 'numeric' });
	}

	function elapsed(): string {
		if (!workout) return '';
		const ms = Date.now() - new Date(workout.started_at).getTime();
		const mins = Math.floor(ms / 60000);
		if (mins < 60) return `${mins}m`;
		return `${Math.floor(mins / 60)}h ${mins % 60}m`;
	}

	const difficultyLabels = ['', 'Easy', 'Light', 'Moderate', 'Hard', 'Max'];
</script>

{#if loading}
	<div class="card bg-base-100 shadow-sm p-8 text-center">
		<span class="loading loading-dots loading-md text-primary"></span>
	</div>
{:else if error}
	<div class="card bg-base-100 shadow-sm p-8 text-center">
		<p class="text-error font-bold">{error}</p>
		<a href="/fitness" class="btn btn-sm btn-ghost mt-4">← Back</a>
	</div>
{:else if workout}

	<!-- Header -->
	<PageHeader title="Workout" breadcrumbs={[{ label: 'Fitness', href: '/fitness' }, { label: 'Workout' }]}>
		{#snippet actions()}
			{#if workout.is_deload}
				<span class="badge badge-outline badge-sm text-warning">Deload</span>
			{/if}
			{#if !workout.completed_at}
				<button class="btn btn-success btn-sm" onclick={() => (showComplete = true)}>
					✓ Finish
				</button>
				<label class="flex items-center gap-1 cursor-pointer">
					<input
						type="checkbox"
						class="checkbox checkbox-xs checkbox-warning"
						checked={workout.is_deload}
						onchange={async (e) => {
							if (!workout) return;
							await updateWorkoutMeta(workout.id, { is_deload: e.currentTarget.checked });
							workout = { ...workout, is_deload: e.currentTarget.checked };
						}}
					/>
					<span class="text-xs text-base-content/50">Deload</span>
				</label>
			{:else}
				<span class="badge badge-success">Completed</span>
			{/if}
		{/snippet}
	</PageHeader>

	<p class="text-sm text-base-content/60 -mt-4 mb-4">
		{formatDate(workout.started_at)}
		{#if !workout.completed_at}
			&middot; {elapsed()} elapsed
		{/if}
	</p>

	<!-- Exercises -->
	{#each workout.exercises as we, idx (we.id)}
		<div class="card bg-base-100 shadow-sm mb-4" data-exercise-card={we.id}>
			<div class="card-body p-4">
				<!-- Exercise Header -->
				<div class="flex justify-between items-start">
					<div>
						<h3 class="font-bold text-primary">{we.exercise_name}</h3>
						<span class="text-xs text-base-content/50">{we.exercise_category}</span>
					</div>
					<div class="flex gap-1">
						<button
							class="btn btn-ghost btn-xs"
							onclick={() => toggleHistory(we)}
							title="View history"
						>📋</button>
						{#if !workout.completed_at}
						{#if confirmDeleteId === we.id}
							<button
								class="btn btn-error btn-xs"
								onclick={() => handleRemoveExercise(we.id)}
							>Delete</button>
							<button
								class="btn btn-ghost btn-xs"
								onclick={cancelRemoveExercise}
							>Cancel</button>
						{:else}
							<button
								class="btn btn-ghost btn-xs text-error"
								onclick={() => requestRemoveExercise(we.id)}
								title="Remove exercise"
							>✕</button>
						{/if}
						{/if}
					</div>
				</div>

				<!-- History Panel -->
				{#if expandedHistory === we.id && historyMap[we.exercise_id]}
					{@const isCardio = we.exercise_category === 'cardio'}
					<div class="bg-base-300 rounded-md p-3 mt-2 text-sm">
						<div class="flex items-center justify-between mb-2">
							<h4 class="font-bold text-xs text-base-content/60 tracking-wide">RECENT HISTORY</h4>
							<a href="/fitness/progress" class="link link-primary text-xs">Full Progress →</a>
						</div>
						{#if (historyMap[we.exercise_id] ?? []).length > 0}
						<div class="overflow-x-auto">
							<table class="table table-xs table-zebra">
								<thead>
									<tr class="text-xs text-base-content/60">
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
									</tr>
								</thead>
								<tbody>
									{#each historyMap[we.exercise_id] as entry}
										{@const sets = entry.sets ?? []}
										{@const final_s = sets.length > 0 ? sets[sets.length - 1] : null}
										{@const best_s = sets.reduce((b, s) => (s.weight ?? 0) > (b?.weight ?? 0) || ((s.weight ?? 0) === (b?.weight ?? 0) && (s.reps ?? 0) > (b?.reps ?? 0)) ? s : b, sets[0] ?? null)}
										<tr class="hover">
											<td class="font-semibold text-xs whitespace-nowrap">{formatDate(entry.date)}</td>
											{#if isCardio}
												<td class="text-right text-xs">
													{final_s?.duration_seconds ? `${Math.round(final_s.duration_seconds / 60)}min` : '—'}
												</td>
												<td class="text-right text-xs">
													{final_s?.distance_miles ? `${final_s.distance_miles}mi` : '—'}
												</td>
											{:else}
												<td class="text-right text-xs font-mono">
													{final_s?.weight !== undefined && final_s?.reps !== undefined ? `${final_s.weight}lb × ${final_s.reps}` : '—'}
												</td>
												<td class="text-right text-xs font-mono opacity-60">
													{best_s?.weight !== undefined && best_s?.reps !== undefined ? `${best_s.weight}lb × ${best_s.reps}` : '—'}
												</td>
											{/if}
											<td class="text-center">
												{#if entry.difficulty}
													<span class="badge badge-xs badge-outline">{difficultyLabels[entry.difficulty]}</span>
												{:else}
													<span class="text-xs opacity-30">—</span>
												{/if}
											</td>
											<td class="text-center">
												{#if entry.ready_to_progress}
													<span class="text-success font-bold text-xs">✓</span>
												{:else}
													<span class="text-xs opacity-30">—</span>
												{/if}
											</td>
										</tr>
									{/each}
								</tbody>
							</table>
						</div>
						{:else}
							<p class="text-xs opacity-40">No history yet</p>
						{/if}
					</div>
				{/if}

				<!-- Sets -->
				{#if we.exercise_category === 'cardio'}
					<!-- Cardio: single row with duration/distance/speed/incline -->
					{#each we.sets as s, i (s.id)}
						<div class="grid grid-cols-2 gap-2 mt-2 items-center" data-set-id={s.id}>
							<label class="text-xs text-base-content/50">
								Duration (min)
								<input
									type="number"
									inputmode="decimal"
									class="input input-bordered input-sm w-full mt-0.5"
									value={s.duration_seconds ? Math.round(s.duration_seconds / 60) : ''}
									onblur={(e) => handleUpdateSet(s, 'duration_seconds', e.currentTarget.value ? String(Number(e.currentTarget.value) * 60) : '')}
									disabled={!!workout.completed_at}
								/>
							</label>
							<label class="text-xs text-base-content/50">
								Distance (mi)
								<input
									type="number"
									inputmode="decimal"
									step="0.1"
									class="input input-bordered input-sm w-full mt-0.5"
									value={s.distance_miles ?? ''}
									onblur={(e) => handleUpdateSet(s, 'distance_miles', e.currentTarget.value)}
									disabled={!!workout.completed_at}
								/>
							</label>
							<label class="text-xs text-base-content/50">
								Top Speed
								<input
									type="number"
									inputmode="decimal"
									step="0.1"
									class="input input-bordered input-sm w-full mt-0.5"
									value={s.top_speed_mph ?? ''}
									onblur={(e) => handleUpdateSet(s, 'top_speed_mph', e.currentTarget.value)}
									disabled={!!workout.completed_at}
								/>
							</label>
							<label class="text-xs text-base-content/50">
								Incline %
								<input
									type="number"
									inputmode="decimal"
									step="0.5"
									class="input input-bordered input-sm w-full mt-0.5"
									value={s.incline_percent ?? ''}
									onblur={(e) => handleUpdateSet(s, 'incline_percent', e.currentTarget.value)}
									disabled={!!workout.completed_at}
								/>
							</label>
						</div>
						{#if !workout.completed_at}
							<div class="flex justify-end mt-1">
								<button class="btn btn-ghost btn-xs text-error" onclick={() => handleDeleteSet(s.id)}>Delete</button>
							</div>
						{/if}
					{/each}
					{#if we.sets.length === 0 && !workout.completed_at}
						<button class="btn btn-sm btn-ghost mt-2" onclick={() => handleAddSet(we)}>+ Add Entry</button>
					{/if}
				{:else}
					<!-- Strength / Bodyweight: set table -->
					{#if we.exercise_category === 'bodyweight' && userBodyweight}
						<div class="text-xs text-base-content/60 mt-2 mb-1">
							Bodyweight: {userBodyweight} lb — weight entered below is <em>added</em> weight
						</div>
					{/if}
					{#if we.sets.length > 0}
						<div class="overflow-x-auto mt-2">
							<table class="table table-sm">
								<thead>
									<tr>
										<th class="w-12">#</th>
										<th>Reps</th>
										<th>{we.exercise_category === 'bodyweight' ? 'Added Wt (lb)' : 'Weight (lb)'}</th>
										{#if we.exercise_category === 'bodyweight' && userBodyweight}<th>Effective</th>{/if}
										{#if !workout.completed_at}<th class="w-12"></th>{/if}
									</tr>
								</thead>
								<tbody>
									{#each we.sets as s, i (s.id)}
										<tr data-set-id={s.id}>
											<td class="opacity-50">{s.set_number}</td>
											<td>
												<input
													type="number"
												inputmode="numeric"
												class="input input-bordered input-sm w-16 md:w-20"
												value={s.reps ?? ''}
												onblur={(e) => handleUpdateSet(s, 'reps', e.currentTarget.value)}
												disabled={!!workout.completed_at}
											/>
										</td>
										<td>
											<input
												type="number"
												inputmode="numeric"
												class="input input-bordered input-sm w-16 md:w-20"
													value={s.weight ?? ''}
													onblur={(e) => handleUpdateSet(s, 'weight', e.currentTarget.value)}
													disabled={!!workout.completed_at}
												/>
											</td>
											{#if we.exercise_category === 'bodyweight' && userBodyweight}
												<td class="text-xs opacity-70">{(userBodyweight + (s.weight ?? 0))} lb</td>
											{/if}
											{#if !workout.completed_at}
												<td>
													<button class="btn btn-ghost btn-xs text-error" onclick={() => handleDeleteSet(s.id)}>✕</button>
												</td>
											{/if}
										</tr>
									{/each}
								</tbody>
							</table>
						</div>
					{/if}
					{#if !workout.completed_at}
						<button class="btn btn-sm btn-ghost mt-1" onclick={() => handleAddSet(we)}>+ Set</button>
					{/if}
				{/if}

				<!-- Exercise metadata: notes, difficulty, progression -->
				<div class="flex flex-col md:flex-row flex-wrap gap-3 mt-3 pt-3 border-t border-base-300">
					<label class="text-xs text-base-content/50 flex-1 min-w-0">
						Notes
						<input
							type="text"
							class="input input-bordered input-sm w-full mt-0.5"
							value={we.notes ?? ''}
							onblur={(e) => handleUpdateExercise(we.id, { notes: e.currentTarget.value || undefined })}
							placeholder="How did it feel?"
							disabled={!!workout.completed_at}
						/>
					</label>
					<label class="text-xs text-base-content/50">
						Difficulty
						<select
							class="select select-bordered select-sm mt-0.5"
							value={we.difficulty ?? ''}
							onchange={(e) => handleUpdateExercise(we.id, { difficulty: Number(e.currentTarget.value) || undefined })}
							disabled={!!workout.completed_at}
						>
							<option value="">—</option>
							{#each [1, 2, 3, 4, 5] as d}
								<option value={d}>{d} — {difficultyLabels[d]}</option>
							{/each}
						</select>
					</label>
					<label class="text-xs text-base-content/50 flex items-end gap-1 py-1">
						<input
							type="checkbox"
							class="checkbox checkbox-sm checkbox-success"
							checked={we.ready_to_progress}
							onchange={(e) => handleUpdateExercise(we.id, { ready_to_progress: e.currentTarget.checked })}
							disabled={!!workout.completed_at}
						/>
						Ready to progress
					</label>
				</div>
			</div>
		</div>
	{/each}

	<!-- Add Exercise Button -->
	{#if !workout.completed_at}
		{#if showPicker}
			<div class="card bg-base-200 border-2 border-primary border-dashed mb-4">
				<div class="card-body p-4">
					<div class="flex justify-between items-center mb-3">
						<h3 class="font-bold text-primary text-sm">ADD EXERCISE</h3>
						<button class="btn btn-ghost btn-xs" onclick={() => (showPicker = false)}>✕</button>
					</div>
					<div class="flex gap-2 mb-3 flex-col sm:flex-row">
						<input
							type="text"
							class="input input-bordered input-sm flex-1"
							placeholder="Search exercises..."
							bind:value={exerciseSearch}
							onkeydown={(e) => e.key === 'Enter' && searchExercises()}
						/>
						<div class="flex gap-2">
							<select class="select select-bordered select-sm flex-1" bind:value={exerciseCategory} onchange={searchExercises}>
								<option value="">All</option>
								<option value="strength">Strength</option>
								<option value="bodyweight">Bodyweight</option>
								<option value="cardio">Cardio</option>
							</select>
							<button class="btn btn-sm" onclick={searchExercises}>Search</button>
						</div>
					</div>
					{#if loadingExercises}
						<span class="loading loading-dots loading-sm"></span>
					{:else}
						<div class="flex flex-col gap-1 max-h-[60vh] md:max-h-60 overflow-y-auto -mx-1 px-1">
							{#each availableExercises as ex}
								<button
									class="btn btn-ghost btn-sm md:btn-sm h-auto min-h-[44px] py-2 justify-start text-left w-full"
									onclick={() => handleAddExercise(ex)}
								>
									<div class="flex flex-col items-start">
										<span class="font-semibold">{ex.name}</span>
										<span class="text-xs opacity-40">{ex.category}{ex.muscle_group ? ' · ' + ex.muscle_group : ''}</span>
									</div>
								</button>
							{:else}
								<p class="text-sm opacity-40 text-center py-4">No exercises found</p>
							{/each}
						</div>
					{/if}
				</div>
			</div>
		{:else}
			<button class="btn btn-outline btn-primary btn-block mb-4" onclick={openPicker}>
				+ Add Exercise
			</button>
		{/if}
	{/if}

	<!-- Workout Notes (after completion) -->
	{#if workout.completed_at && workout.notes}
		<div class="card bg-base-100 shadow-sm p-4 mt-2">
			<h4 class="text-xs font-medium text-base-content/50 mb-1">SESSION NOTES</h4>
			<p class="text-sm">{workout.notes}</p>
		</div>
	{/if}

	<!-- Complete Modal -->
	{#if showComplete}
		<div class="fixed inset-0 bg-black/50 z-50 flex items-center justify-center p-4">
			<div class="card bg-base-100 shadow-xl border-2 border-base-300 w-full max-w-md">
				<div class="card-body">
					<h3 class="card-title text-primary">Finish Workout?</h3>
					<p class="text-sm text-base-content/60">
						{workout.exercises.length} exercise{workout.exercises.length !== 1 ? 's' : ''},
						{workout.exercises.reduce((sum, e) => sum + e.sets.length, 0)} total sets
					</p>
					<textarea
						class="textarea textarea-bordered w-full mt-2"
						rows="3"
						placeholder="Session notes (optional)..."
						bind:value={completionNotes}
					></textarea>
					<div class="card-actions justify-end mt-4">
						<button class="btn btn-ghost btn-sm" onclick={() => (showComplete = false)}>Cancel</button>
						<button class="btn btn-success btn-sm" onclick={handleComplete}>Complete ✓</button>
					</div>
				</div>
			</div>
		</div>
	{/if}
{/if}
