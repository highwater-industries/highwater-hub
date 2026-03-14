<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import {
		listFitnessUsers,
		createFitnessUser,
		listWorkouts,
		createWorkout,
		deleteWorkout,
		type FitnessUser,
		type WorkoutSummary
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

	async function loadUsers() {
		try {
			users = await listFitnessUsers();
			// Restore last selected user from localStorage
			const savedId = localStorage.getItem('fitness_user_id');
			if (savedId) {
				activeUser = users.find((u) => u.id === Number(savedId)) ?? null;
			}
			if (!activeUser && users.length > 0) {
				selectUser(users[0]);
			}
			if (activeUser) await loadWorkouts();
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
		if (!completed) return 'In Progress';
		const ms = new Date(completed).getTime() - new Date(started).getTime();
		const mins = Math.round(ms / 60000);
		if (mins < 60) return `${mins}m`;
		return `${Math.floor(mins / 60)}h ${mins % 60}m`;
	}

	onMount(loadUsers);
</script>

<div class="flex justify-between items-center mb-4">
	<h1 class="text-xl md:text-2xl font-bold text-primary tracking-wide">// FITNESS</h1>
	{#if activeUser}
		<span class="text-xs md:text-sm opacity-60">{totalWorkouts} WORKOUTS</span>
	{/if}
</div>

{#if loading}
	<div class="card bg-base-200 shadow-md border border-base-300 p-8 text-center">
		<span class="loading loading-dots loading-md text-primary"></span>
		<p class="text-sm opacity-60 mt-2">Loading...</p>
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
		<!-- Start Workout -->
		<div class="mb-6">
			<div class="flex flex-wrap items-center gap-2">
				<button class="btn btn-primary btn-lg flex-1 md:flex-none" onclick={handleStartWorkout}>
					💪 Start Workout
				</button>
				<button
					class="btn btn-ghost btn-sm"
					onclick={() => (showNewWorkoutOptions = !showNewWorkoutOptions)}
				>{showNewWorkoutOptions ? '▲ Less' : '▼ Options'}</button>
			</div>
			{#if showNewWorkoutOptions}
				<div class="flex flex-wrap items-center gap-3 mt-3 p-3 bg-base-200 rounded-lg border border-base-300">
					<label class="text-xs opacity-50 flex items-center gap-2">
						Date
						<input
							type="date"
							class="input input-bordered input-sm w-40"
							bind:value={workoutDate}
						/>
					</label>
					<label class="text-xs opacity-50 flex items-center gap-2 cursor-pointer">
						<input
							type="checkbox"
							class="checkbox checkbox-sm checkbox-warning"
							bind:checked={workoutIsDeload}
						/>
						Deload Week
					</label>
				</div>
			{/if}
		</div>

		<!-- Recent Workouts -->
		{#if loadingWorkouts}
			<div class="text-center py-8">
				<span class="loading loading-dots loading-md text-primary"></span>
			</div>
		{:else if workouts.length === 0}
			<div class="card bg-base-200 shadow-md border border-base-300 p-8 text-center">
				<p class="text-lg font-bold opacity-60">No workouts yet</p>
				<p class="text-sm opacity-40 mt-1">Hit the button above to start your first session</p>
			</div>
		{:else}
			<h2 class="text-sm font-bold opacity-50 mb-2 tracking-wide">RECENT WORKOUTS</h2>
			<div class="flex flex-col gap-3">
				{#each workouts as w}
					<a
						href="/fitness/workout/{w.id}"
						class="card bg-base-200 border-2 border-base-300 hover:shadow-lg hover:-translate-y-0.5 transition-all no-underline"
					>
						<div class="card-body p-4">
							<div class="flex justify-between items-start">
								<div>
									<div class="font-bold text-primary">
										{formatDate(w.started_at)}
										<span class="text-xs opacity-50 ml-2">{formatTime(w.started_at)}</span>
									</div>
									<div class="text-sm opacity-60 mt-1">
										{w.exercise_count} exercise{w.exercise_count !== 1 ? 's' : ''}
										&middot; {w.set_count} set{w.set_count !== 1 ? 's' : ''}
										&middot; {duration(w.started_at, w.completed_at)}
									</div>
									{#if w.exercise_names}
										<div class="text-xs opacity-40 mt-1">{w.exercise_names}</div>
									{/if}
								</div>
								<div class="flex items-center gap-2">
									{#if w.completed_at}
										<span class="badge badge-success badge-sm">Done</span>
									{:else}
										<span class="badge badge-warning badge-sm">Active</span>
									{/if}
									{#if w.is_deload}
										<span class="badge badge-outline badge-sm text-warning">Deload</span>
									{/if}
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
										class="btn btn-ghost btn-xs text-error"
										onclick={(e) => { e.stopPropagation(); e.preventDefault(); requestDeleteWorkout(w.id); }}
										title="Delete workout"
									>✕</button>
								{/if}
								</div>
							</div>
						</div>
					</a>
				{/each}
			</div>
		{/if}
	{/if}
{/if}
