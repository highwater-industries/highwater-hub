<script lang="ts">
	import { type ApexOptions } from 'apexcharts';
	import { onMount, onDestroy } from 'svelte';
	import type { SeasonTotals } from '$lib/api';

	let {
		seasons,
		position,
	}: {
		seasons: SeasonTotals[];
		position: string | undefined | null;
	} = $props();

	// ── Stat presets by position ──
	type StatDef = { key: keyof SeasonTotals; label: string; color: string };

	const passingStats: StatDef[] = [
		{ key: 'passing_yards', label: 'Pass Yards', color: '#167bff' },
		{ key: 'passing_tds', label: 'Pass TDs', color: '#22c55e' },
		{ key: 'interceptions', label: 'INTs', color: '#ef4444' },
		{ key: 'completions', label: 'Completions', color: '#a855f7' },
		{ key: 'attempts', label: 'Attempts', color: '#f59e0b' },
	];
	const rushingStats: StatDef[] = [
		{ key: 'rushing_yards', label: 'Rush Yards', color: '#167bff' },
		{ key: 'rushing_tds', label: 'Rush TDs', color: '#22c55e' },
		{ key: 'carries', label: 'Carries', color: '#f59e0b' },
	];
	const receivingStats: StatDef[] = [
		{ key: 'receiving_yards', label: 'Rec Yards', color: '#167bff' },
		{ key: 'receiving_tds', label: 'Rec TDs', color: '#22c55e' },
		{ key: 'receptions', label: 'Receptions', color: '#a855f7' },
		{ key: 'targets', label: 'Targets', color: '#f59e0b' },
	];
	const fantasyStats: StatDef[] = [
		{ key: 'fantasy_points_ppr', label: 'PPR Points', color: '#6c74f8' },
		{ key: 'fantasy_points', label: 'Standard Points', color: '#ff8b4b' },
	];

	type PresetKey = 'passing' | 'rushing' | 'receiving' | 'fantasy';

	const presets: Record<PresetKey, { label: string; stats: StatDef[] }> = {
		passing: { label: 'Passing', stats: passingStats },
		rushing: { label: 'Rushing', stats: rushingStats },
		receiving: { label: 'Receiving', stats: receivingStats },
		fantasy: { label: 'Fantasy', stats: fantasyStats },
	};

	function defaultPreset(pos: string | undefined | null): PresetKey {
		switch (pos?.toUpperCase()) {
			case 'QB': return 'passing';
			case 'RB': return 'rushing';
			case 'WR': case 'TE': return 'receiving';
			default: return 'fantasy';
		}
	}

	let activePreset: PresetKey = $state(defaultPreset(position));
	let chartType: 'line' | 'bar' = $state('bar');

	// Available presets for this position
	let availablePresets = $derived.by(() => {
		const pos = position?.toUpperCase();
		const available: PresetKey[] = [];
		if (pos === 'QB' || !pos) available.push('passing');
		available.push('rushing');
		if (pos !== 'QB' || !pos) available.push('receiving');
		available.push('fantasy');
		return available;
	});

	// Filter seasons to only "total" rows (one per year) and sort by year
	let chartSeasons = $derived.by(() => {
		return seasons
			.filter(s => s.season_type === 'total' && s.season > 0)
			.sort((a, b) => a.season - b.season);
	});

	let chartRef: HTMLDivElement | null = $state(null);
	let chart: any = null;
	let mounted = $state(false);
	let ApexChartsClass: any = null;

	function buildOptions(): ApexOptions {
		const stats = presets[activePreset].stats;
		const categories = chartSeasons.map(s => String(s.season));

		// Pick first 2 stats for dual-axis, rest as supplementary
		const primaryStats = stats.slice(0, 2);
		const series = primaryStats.map(stat => ({
			name: stat.label,
			data: chartSeasons.map(s => (s[stat.key] as number) ?? 0),
		}));

		const colors = primaryStats.map(s => s.color);

		return {
			chart: {
				type: chartType,
				height: 320,
				background: 'transparent',
				toolbar: {
					show: true,
					tools: {
						download: true,
						zoom: false,
						zoomin: false,
						zoomout: false,
						pan: false,
						reset: false,
					},
				},
				fontFamily: 'inherit',
			},
			series,
			colors,
			xaxis: {
				categories,
				axisBorder: { show: false },
				axisTicks: { show: false },
				labels: {
					style: { colors: 'var(--color-base-content)', fontWeight: '500' },
				},
			},
			yaxis: primaryStats.length === 2
				? [
						{
							title: { text: primaryStats[0].label, style: { color: colors[0], fontWeight: '600' } },
							labels: {
								formatter: (v: number) => v >= 1000 ? `${(v / 1000).toFixed(1)}k` : String(Math.round(v)),
								style: { colors: colors[0] },
							},
						},
						{
							opposite: true,
							title: { text: primaryStats[1].label, style: { color: colors[1], fontWeight: '600' } },
							labels: {
								formatter: (v: number) => String(Math.round(v)),
								style: { colors: colors[1] },
							},
						},
					]
				: {
						labels: {
							formatter: (v: number) => v >= 1000 ? `${(v / 1000).toFixed(1)}k` : String(Math.round(v)),
							style: { colors: 'var(--color-base-content)' },
						},
					},
			stroke: chartType === 'line'
				? { curve: 'smooth', width: 3 }
				: { show: true, width: 2, colors: ['transparent'] },
			markers: chartType === 'line'
				? { size: 4, strokeWidth: 2, hover: { sizeOffset: 2 } }
				: { size: 0 },
			plotOptions: chartType === 'bar'
				? {
						bar: {
							borderRadius: 4,
							columnWidth: '55%',
						},
					}
				: {},
			dataLabels: { enabled: false },
			legend: {
				show: true,
				position: 'top',
				horizontalAlign: 'left',
				labels: { colors },
			},
			grid: {
				borderColor: 'var(--color-base-content/0.08)',
				strokeDashArray: 4,
			},
			tooltip: {
				theme: 'dark',
				shared: true,
				intersect: false,
				y: {
					formatter: (v: number) => v?.toLocaleString() ?? '—',
				},
			},
			fill: { opacity: 1 },
		};
	}

	onMount(async () => {
		ApexChartsClass = (await import('apexcharts')).default;
		if (chartRef) {
			chart = new ApexChartsClass(chartRef, buildOptions());
			chart.render();
			mounted = true;
		}
	});

	onDestroy(() => {
		chart?.destroy();
	});

	function recreateChart() {
		if (!mounted || !ApexChartsClass || !chartRef) return;
		chart?.destroy();
		chart = new ApexChartsClass(chartRef, buildOptions());
		chart.render();
	}

	// Recreate chart when type changes (ApexCharts doesn't cleanly switch chart types)
	$effect(() => {
		const _type = chartType;
		recreateChart();
	});

	// Update data when preset or seasons change
	$effect(() => {
		if (mounted && chart) {
			const _preset = activePreset;
			const _seasons = chartSeasons;
			chart.updateOptions(buildOptions(), true, true);
		}
	});
</script>

{#if chartSeasons.length >= 2}
	<div class="card bg-base-100 shadow-sm overflow-hidden">
		<div class="card-body p-4 md:p-6">
			<!-- Controls bar -->
			<div class="flex flex-wrap items-center gap-2 mb-2">
				<h2 class="text-base font-semibold mr-auto">Season Stats</h2>

				<!-- Stat preset tabs -->
				<div class="join">
					{#each availablePresets as preset}
						<button
							class="join-item btn btn-xs {activePreset === preset ? 'btn-primary' : 'btn-ghost'}"
							onclick={() => (activePreset = preset)}
						>
							{presets[preset].label}
						</button>
					{/each}
				</div>

				<!-- Chart type toggle -->
				<div class="join">
					<button
						class="join-item btn btn-xs {chartType === 'bar' ? 'btn-neutral' : 'btn-ghost'}"
						onclick={() => (chartType = 'bar')}
						title="Bar chart"
					>
						<span class="iconify lucide--bar-chart-3 size-3.5"></span>
					</button>
					<button
						class="join-item btn btn-xs {chartType === 'line' ? 'btn-neutral' : 'btn-ghost'}"
						onclick={() => (chartType = 'line')}
						title="Line chart"
					>
						<span class="iconify lucide--trending-up size-3.5"></span>
					</button>
				</div>
			</div>

			<!-- Chart -->
			<div bind:this={chartRef}></div>
		</div>
	</div>
{/if}
