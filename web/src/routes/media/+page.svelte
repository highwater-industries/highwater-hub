<script lang="ts">
	interface Service {
		name: string;
		icon: string;
		port: number;
		color: string;
		description: string;
		category: 'media' | 'infra';
	}

	const services: Service[] = [
		{ name: 'PLEX',      icon: '▶',  port: 32400, color: '#e5a00d', description: 'Media Server',      category: 'media' },
		{ name: 'OVERSEERR',  icon: '✦',  port: 5055,  color: '#7b68ee', description: 'Request Manager',   category: 'media' },
		{ name: 'SONARR',    icon: '📺', port: 8989,  color: '#35c5f4', description: 'TV Shows',           category: 'media' },
		{ name: 'RADARR',    icon: '🎬', port: 7878,  color: '#ffc230', description: 'Movies',             category: 'media' },
		{ name: 'PROWLARR',  icon: '🔍', port: 9696,  color: '#f47521', description: 'Indexer Manager',    category: 'media' },
		{ name: 'SABNZBD',   icon: '⬇',  port: 8080,  color: '#eab92d', description: 'Download Client',   category: 'media' },
		{ name: 'TAUTULLI',  icon: '📊', port: 8181,  color: '#cc7b19', description: 'Plex Analytics',     category: 'media' },
		{ name: 'UPTIME KUMA', icon: '♥', port: 3001,  color: '#5cdd8b', description: 'Status Monitor',   category: 'infra' },
		{ name: 'SCRUTINY',    icon: '💾', port: 8081, color: '#4fc3f7', description: 'Disk Health',       category: 'infra' },
		{ name: 'RABBITMQ',    icon: '🐇', port: 15672, color: '#ff6600', description: 'Message Broker',  category: 'infra' },
		{ name: 'POSTGRES',    icon: '🐘', port: 5432, color: '#336791', description: 'Database',          category: 'infra' },
	];

	function serviceUrl(port: number): string {
		const host = typeof window !== 'undefined' ? window.location.hostname : 'localhost';
		return `http://${host}:${port}`;
	}
</script>

<div class="flex justify-between items-center mb-6">
	<h1 class="text-xl md:text-2xl font-bold text-primary tracking-wide">// MEDIA & INFRA</h1>
</div>

<h4 class="text-sm font-bold opacity-50 mb-2 tracking-wide">MEDIA STACK</h4>
<div class="grid grid-cols-[repeat(auto-fill,minmax(130px,1fr))] md:grid-cols-[repeat(auto-fill,minmax(150px,1fr))] gap-3 mb-6">
	{#each services.filter(s => s.category === 'media') as svc}
		<a href={serviceUrl(svc.port)} target="_blank" rel="noopener"
			class="card bg-base-200 border-2 border-base-300 hover:shadow-lg hover:-translate-y-0.5 transition-all no-underline">
			<div class="card-body items-center text-center p-4 gap-1">
				<span class="text-2xl">{svc.icon}</span>
				<span class="font-bold text-sm tracking-wide" style="color: {svc.color}">{svc.name}</span>
				<span class="text-xs opacity-60">{svc.description}</span>
				<span class="text-xs opacity-30">:{svc.port}</span>
			</div>
		</a>
	{/each}
</div>

<h4 class="text-sm font-bold opacity-50 mb-2 tracking-wide">INFRASTRUCTURE</h4>
<div class="grid grid-cols-[repeat(auto-fill,minmax(130px,1fr))] md:grid-cols-[repeat(auto-fill,minmax(150px,1fr))] gap-3">
	{#each services.filter(s => s.category === 'infra') as svc}
		<a href={serviceUrl(svc.port)} target="_blank" rel="noopener"
			class="card bg-base-200 border-2 border-base-300 hover:shadow-lg hover:-translate-y-0.5 transition-all no-underline">
			<div class="card-body items-center text-center p-4 gap-1">
				<span class="text-2xl">{svc.icon}</span>
				<span class="font-bold text-sm tracking-wide" style="color: {svc.color}">{svc.name}</span>
				<span class="text-xs opacity-60">{svc.description}</span>
				<span class="text-xs opacity-30">:{svc.port}</span>
			</div>
		</a>
	{/each}
</div>
