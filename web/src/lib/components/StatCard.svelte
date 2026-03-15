<script lang="ts">
	/**
	 * StatCard — Nexus-style stat card with value, label, and optional icon.
	 * Use for KPI grids, career totals, inventory counters, queue dashboards.
	 */
	let {
		label,
		value,
		icon,
		subtext,
		color = '',
		href,
	}: {
		label: string;
		value: string | number;
		icon?: string;
		subtext?: string;
		color?: string;
		href?: string;
	} = $props();

	const fmtValue = typeof value === 'number' ? value.toLocaleString() : value;
</script>

{#if href}
<a {href} class="card bg-base-100 shadow-sm transition-all hover:shadow-md no-underline">
	<div class="card-body gap-1 p-4">
		<div class="flex items-center justify-between">
			<p class="text-base-content/60 text-sm font-medium">{label}</p>
			{#if icon}
				<div class="bg-base-200 rounded-box flex items-center p-1.5">
					<span class="iconify size-4 {icon}"></span>
				</div>
			{/if}
		</div>
		<p class="text-2xl font-semibold {color}">{fmtValue}</p>
		{#if subtext}
			<p class="text-base-content/50 text-xs">{subtext}</p>
		{/if}
	</div>
</a>
{:else}
<div class="card bg-base-100 shadow-sm">
	<div class="card-body gap-1 p-4">
		<div class="flex items-center justify-between">
			<p class="text-base-content/60 text-sm font-medium">{label}</p>
			{#if icon}
				<div class="bg-base-200 rounded-box flex items-center p-1.5">
					<span class="iconify size-4 {icon}"></span>
				</div>
			{/if}
		</div>
		<p class="text-2xl font-semibold {color}">{fmtValue}</p>
		{#if subtext}
			<p class="text-base-content/50 text-xs">{subtext}</p>
		{/if}
	</div>
</div>
{/if}
