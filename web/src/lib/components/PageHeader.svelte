<script lang="ts">
	import type { Snippet } from 'svelte';

	interface Breadcrumb {
		label: string;
		href?: string;
	}

	let {
		title,
		breadcrumbs = [],
		actions,
	}: {
		title: string;
		breadcrumbs?: Breadcrumb[];
		actions?: Snippet;
	} = $props();
</script>

<div class="mb-5">
	{#if breadcrumbs.length > 0}
		<div class="breadcrumbs text-sm py-0 mb-1 min-h-0">
			<ul class="text-xs opacity-50">
				<li><a href="/">Home</a></li>
				{#each breadcrumbs as crumb}
					<li>
						{#if crumb.href}
							<a href={crumb.href}>{crumb.label}</a>
						{:else}
							{crumb.label}
						{/if}
					</li>
				{/each}
			</ul>
		</div>
	{/if}
	<div class="flex items-center justify-between gap-3 flex-wrap">
		<h1 class="text-lg font-semibold">{title}</h1>
		{#if actions}
			<div class="flex items-center gap-2 flex-wrap">
				{@render actions()}
			</div>
		{/if}
	</div>
</div>
