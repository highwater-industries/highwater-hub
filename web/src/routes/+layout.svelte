<script lang="ts">
	import { page } from '$app/stores';
	import '../app.css';

	let { children } = $props();

	const navItems = [
		{ href: '/', label: 'Home Base', icon: '⌂' },
		{ href: '/players', label: 'Players', icon: '⚑' },
		{ href: '/jobs', label: 'Jobs', icon: '⚙' }
	];
</script>

<div class="app">
	<nav class="sidebar">
		<div class="logo">
			<h2>HIGHWATER<br/>HUB</h2>
			<div class="subtitle">FLOOD FAMILY HQ</div>
		</div>
		<ul>
			{#each navItems as item}
				<li>
					<a href={item.href} class:active={$page.url.pathname === item.href}>
						<span class="nav-icon">{item.icon}</span>
						{item.label}
					</a>
				</li>
			{/each}
		</ul>
	</nav>

	<main class="content">
		{@render children()}
	</main>
</div>
