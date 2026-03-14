<script lang="ts">
	// Sections available in the app — drives the mobile launcher
	const sections = [
		{ href: '/fitness', label: 'Fitness', icon: '💪', desc: 'Log workouts & track progress', color: 'text-success' },
		{ href: '/players', label: 'NFL Players', icon: '⚑', desc: 'Player roster database', color: 'text-primary' },
		{ href: '/stats', label: 'NFL Stats', icon: '📊', desc: 'Player stats & leaders', color: 'text-secondary' },
		{ href: '/games', label: 'NFL Games', icon: '🏈', desc: 'Schedules & scores', color: 'text-accent' },
		{ href: '/rankings', label: 'NFL Rankings', icon: '★', desc: 'Fantasy football rankings', color: 'text-warning' },
		{ href: '/jobs', label: 'Jobs', icon: '⚙', desc: 'Data import pipeline', color: 'text-info' },
		{ href: '/media', label: 'Media', icon: '▶', desc: 'Media servers & infrastructure', color: 'text-error' }
	];
</script>

<!-- ═══ MOBILE HOME — section launcher ═══ -->
<div class="md:hidden">
	<div class="text-center mb-6">
		<h1 class="text-2xl font-bold text-primary tracking-widest">HIGHWATER HUB</h1>
		<p class="text-xs opacity-40 tracking-widest mt-1">FLOOD FAMILY HQ</p>
	</div>

	<div class="flex flex-col gap-2">
		{#each sections as sec}
			<a
				href={sec.href}
				class="flex items-center gap-4 p-4 rounded-xl bg-base-200 border border-base-300 active:scale-[0.98] transition-all no-underline"
			>
				<span class="text-3xl w-12 text-center shrink-0">{sec.icon}</span>
				<div class="flex-1 min-w-0">
					<div class="font-bold tracking-wide {sec.color}">{sec.label}</div>
					<div class="text-xs opacity-50">{sec.desc}</div>
				</div>
				<span class="text-base-content/20 text-lg">›</span>
			</a>
		{/each}
	</div>
</div>

<!-- ═══ DESKTOP HOME — blank, will get context cards later ═══ -->
<div class="hidden md:block">
	<div class="flex justify-between items-center mb-6">
		<h1 class="text-2xl font-bold text-primary tracking-wide">// DASHBOARD</h1>
	</div>
	<p class="opacity-40 text-sm">Select a section from the sidebar to get started.</p>
</div>
