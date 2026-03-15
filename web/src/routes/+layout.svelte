<script lang="ts">
	import ConfigProvider from "$lib/contexts/ConfigProvider.svelte";
	import Footer from "$lib/components/admin-layout/Footer.svelte";
	import Rightbar from "$lib/components/admin-layout/Rightbar.svelte";
	import Sidebar from "$lib/components/admin-layout/Sidebar.svelte";
	import Topbar from "$lib/components/admin-layout/Topbar.svelte";
	import { appMenuItems } from "./menu";
	import '../app.css';

	let { children } = $props();
</script>

<svelte:head>
	<title>Highwater Hub</title>
	<meta name="description" content="Highwater Hub — Flood Family HQ" />
</svelte:head>

<ConfigProvider>
	<div class="size-full">
		<div class="flex">
			<Sidebar menuItems={appMenuItems} />
			<div class="flex h-screen min-w-0 grow flex-col overflow-auto">
				<Topbar />
				<div id="layout-content" class="p-3 md:p-6">
					{@render children()}
				</div>
				<Footer />
			</div>
		</div>
		<Rightbar />
	</div>
</ConfigProvider>
