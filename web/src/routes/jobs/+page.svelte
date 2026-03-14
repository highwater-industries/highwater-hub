<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	onMount(() => goto('/data', { replaceState: true }));
</script>

<p class="text-center opacity-50 mt-8">Redirecting to Data Management...</p>
