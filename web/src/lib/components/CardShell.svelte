<script lang="ts">
	/**
	 * CardShell — consistent Nexus-style card wrapper.
	 * Replaces the old `card bg-base-200 shadow-md border border-base-300` pattern
	 * with the cleaner `card bg-base-100 shadow-sm` Nexus look.
	 *
	 * Supports an optional title bar with icon and right-side actions.
	 */
	import type { Snippet } from 'svelte';

	let {
		title,
		icon,
		actions,
		children,
		padding = true,
		class: extraClass = '',
	}: {
		title?: string;
		icon?: string;
		actions?: Snippet;
		children: Snippet;
		padding?: boolean;
		class?: string;
	} = $props();
</script>

<div class="card bg-base-100 shadow-sm {extraClass}">
	{#if title || actions}
		<div class="flex items-center gap-3 px-5 pt-4 pb-0">
			{#if icon}
				<span class="iconify size-4.5 {icon}"></span>
			{/if}
			{#if title}
				<span class="font-medium">{title}</span>
			{/if}
			{#if actions}
				<div class="ms-auto flex items-center gap-2">
					{@render actions()}
				</div>
			{/if}
		</div>
	{/if}
	<div class={padding ? 'card-body' : ''}>
		{@render children()}
	</div>
</div>
