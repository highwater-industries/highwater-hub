<script lang="ts">
	import { type ApexOptions } from 'apexcharts';
	import { onMount, onDestroy } from 'svelte';

	let { options, height = 320, class: className = '' }: {
		options: ApexOptions;
		height?: number;
		class?: string;
	} = $props();

	let chartRef: HTMLDivElement | null = $state(null);
	let chart: any = null;

	onMount(async () => {
		const ApexCharts = (await import('apexcharts')).default;
		if (chartRef) {
			const merged: ApexOptions = {
				...options,
				chart: {
					...options.chart,
					height,
					background: 'transparent',
				},
			};
			chart = new ApexCharts(chartRef, merged);
			chart.render();
		}
	});

	onDestroy(() => {
		chart?.destroy();
	});

	// Update chart when options change
	$effect(() => {
		if (chart && options) {
			chart.updateOptions({
				...options,
				chart: {
					...options.chart,
					height,
					background: 'transparent',
				},
			}, true, true);
		}
	});
</script>

<div bind:this={chartRef} class={className}></div>
