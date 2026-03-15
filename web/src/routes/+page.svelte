<script lang="ts">
	import PageHeader from '$lib/components/PageHeader.svelte';

	// Sections available in the app — drives the mobile launcher + desktop cards
	const sections = [
		{ href: '/fitness', label: 'Fitness', icon: '💪', desc: 'Log workouts & track progress', color: 'text-success' },
		{ href: '/players', label: 'NFL Players', icon: '⚑', desc: 'Player roster database', color: 'text-primary' },
		{ href: '/stats', label: 'NFL Stats', icon: '📊', desc: 'Player stats & leaders', color: 'text-secondary' },
		{ href: '/games', label: 'NFL Games', icon: '🏈', desc: 'Schedules & scores', color: 'text-accent' },
		{ href: '/rankings', label: 'NFL Rankings', icon: '★', desc: 'Fantasy football rankings', color: 'text-warning' },
		{ href: '/data', label: 'Data Management', icon: '⚙', desc: 'Data import pipeline', color: 'text-info' },
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
				class="flex items-center gap-4 p-4 rounded-xl bg-base-100 shadow-sm active:scale-[0.98] transition-all no-underline"
			>
				<span class="text-3xl w-12 text-center shrink-0">{sec.icon}</span>
				<div class="flex-1 min-w-0">
					<div class="font-bold tracking-wide {sec.color}">{sec.label}</div>
					<div class="text-xs text-base-content/50">{sec.desc}</div>
				</div>
				<span class="text-base-content/20 text-lg">›</span>
			</a>
		{/each}
	</div>
</div>

<!-- ═══ DESKTOP HOME — quick-launch cards ═══ -->
<div class="hidden md:block">
	<PageHeader title="Dashboard" />

	<div class="grid grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
		{#each sections as sec}
			<a
				href={sec.href}
				class="card bg-base-100 shadow-sm hover:shadow-md hover:-translate-y-0.5 transition-all no-underline group"
			>
				<div class="card-body items-center text-center p-5 gap-2">
					<span class="text-3xl group-hover:scale-110 transition-transform">{sec.icon}</span>
					<span class="font-semibold tracking-wide {sec.color}">{sec.label}</span>
					<span class="text-xs text-base-content/50">{sec.desc}</span>
				</div>
			</a>
		{/each}
	</div>
</div>
