<script lang="ts">
	import { page } from '$app/stores';
	import '../app.css';

	let { children } = $props();

	function isActive(href: string, pathname: string): boolean {
		return href === '/' ? pathname === '/' : pathname.startsWith(href);
	}

	// Breadcrumb map: path segment → display label + parent href
	const labelMap: Record<string, string> = {
		fitness: 'Fitness',
		workout: 'Workout',
		players: 'NFL Players',
		stats: 'NFL Stats',
		games: 'NFL Games',
		rankings: 'NFL Rankings',
		data: 'Data Management',
		media: 'Media'
	};

	function buildCrumbs(pathname: string): { label: string; href: string }[] {
		if (pathname === '/') return [];
		const segments = pathname.split('/').filter(Boolean);
		const crumbs: { label: string; href: string }[] = [{ label: '⌂', href: '/' }];
		let path = '';
		for (const seg of segments) {
			path += '/' + seg;
			const label = labelMap[seg] ?? (seg.match(/^\d+$/) ? '#' + seg : seg);
			crumbs.push({ label, href: path });
		}
		return crumbs;
	}
</script>

<div class="flex min-h-screen">
	<!-- Desktop sidebar — hidden on mobile -->
	<aside class="hidden md:flex w-60 bg-base-200 border-r-2 border-base-300 flex-col shrink-0">
		<div class="px-4 py-5 text-center border-b-2 border-base-300">
			<h2 class="text-lg font-bold text-primary tracking-widest leading-relaxed">HIGHWATER<br/>HUB</h2>
			<p class="text-xs opacity-50 tracking-widest mt-1">FLOOD FAMILY HQ</p>
		</div>
		<ul class="menu menu-md gap-0.5 p-2">
			<li>
				<a href="/" class="rounded-sm font-semibold tracking-wide" class:active={$page.url.pathname === '/'}>
					<span class="text-lg">⌂</span> Dashboard
				</a>
			</li>
			<li>
				<a href="/fitness" class="rounded-sm font-semibold tracking-wide" class:active={$page.url.pathname.startsWith('/fitness')}>
					<span class="text-lg">💪</span> Fitness
				</a>
			</li>
			<li class="menu-title text-xs opacity-40 tracking-widest mt-2">SPORTS</li>
			<li>
				<a href="/players" class="rounded-sm font-semibold tracking-wide" class:active={$page.url.pathname.startsWith('/players')}>
					<span class="text-lg">⚑</span> NFL Players
				</a>
			</li>
			<li>
				<a href="/stats" class="rounded-sm font-semibold tracking-wide" class:active={$page.url.pathname.startsWith('/stats')}>
					<span class="text-lg">📊</span> NFL Stats
				</a>
			</li>
			<li>
				<a href="/games" class="rounded-sm font-semibold tracking-wide" class:active={$page.url.pathname.startsWith('/games')}>
					<span class="text-lg">🏈</span> NFL Games
				</a>
			</li>
			<li>
				<a href="/rankings" class="rounded-sm font-semibold tracking-wide" class:active={$page.url.pathname.startsWith('/rankings')}>
					<span class="text-lg">★</span> NFL Rankings
				</a>
			</li>
			<li class="menu-title text-xs opacity-40 tracking-widest mt-2">SYSTEM</li>
			<li>
				<a href="/data" class="rounded-sm font-semibold tracking-wide" class:active={$page.url.pathname.startsWith('/data')}>
					<span class="text-lg">⚙</span> Data Management
				</a>
			</li>
			<li>
				<a href="/media" class="rounded-sm font-semibold tracking-wide" class:active={$page.url.pathname.startsWith('/media')}>
					<span class="text-lg">▶</span> Media
				</a>
			</li>
		</ul>
	</aside>

	<!-- Main content -->
	<main class="flex-1 p-3 md:p-6 overflow-x-auto">
		<!-- Mobile breadcrumb trail -->
		{#if buildCrumbs($page.url.pathname).length > 0}
			<nav class="md:hidden text-xs breadcrumbs mb-3 -mt-1 opacity-60">
				<ul class="flex items-center gap-1 flex-wrap">
					{#each buildCrumbs($page.url.pathname) as crumb, i}
						{#if i > 0}<li class="opacity-40">/</li>{/if}
						{#if i === buildCrumbs($page.url.pathname).length - 1}
							<li class="font-semibold opacity-80">{crumb.label}</li>
						{:else}
							<li><a href={crumb.href} class="hover:underline">{crumb.label}</a></li>
						{/if}
					{/each}
				</ul>
			</nav>
		{/if}
		{@render children()}
	</main>
</div>
