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
		// Media
		{ name: 'PLEX',      icon: '▶',  port: 32400, color: '#e5a00d', description: 'Media Server',      category: 'media' },
		{ name: 'OVERSEERR',  icon: '✦',  port: 5055,  color: '#7b68ee', description: 'Request Manager',   category: 'media' },
		{ name: 'SONARR',    icon: '📺', port: 8989,  color: '#35c5f4', description: 'TV Shows',           category: 'media' },
		{ name: 'RADARR',    icon: '🎬', port: 7878,  color: '#ffc230', description: 'Movies',             category: 'media' },
		{ name: 'PROWLARR',  icon: '🔍', port: 9696,  color: '#f47521', description: 'Indexer Manager',    category: 'media' },
		{ name: 'SABNZBD',   icon: '⬇',  port: 8080,  color: '#eab92d', description: 'Download Client',   category: 'media' },
		{ name: 'TAUTULLI',  icon: '📊', port: 8181,  color: '#cc7b19', description: 'Plex Analytics',     category: 'media' },
		// Infra
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

<div class="page-header">
	<h1>// HOME BASE</h1>
</div>

<h4 style="font-family: var(--font-pixel); font-size: 0.4rem; color: var(--text-muted); margin-bottom: 0.5rem">
	MEDIA STACK
</h4>
<div class="services-grid">
	{#each services.filter(s => s.category === 'media') as svc}
		<a href={serviceUrl(svc.port)} target="_blank" rel="noopener" class="service-card" style="--svc-color: {svc.color}">
			<span class="service-icon">{svc.icon}</span>
			<span class="service-name">{svc.name}</span>
			<span class="service-desc">{svc.description}</span>
			<span class="service-port">:{svc.port}</span>
		</a>
	{/each}
</div>

<h4 style="font-family: var(--font-pixel); font-size: 0.4rem; color: var(--text-muted); margin: 1.25rem 0 0.5rem">
	INFRASTRUCTURE
</h4>
<div class="services-grid">
	{#each services.filter(s => s.category === 'infra') as svc}
		<a href={serviceUrl(svc.port)} target="_blank" rel="noopener" class="service-card" style="--svc-color: {svc.color}">
			<span class="service-icon">{svc.icon}</span>
			<span class="service-name">{svc.name}</span>
			<span class="service-desc">{svc.description}</span>
			<span class="service-port">:{svc.port}</span>
		</a>
	{/each}
</div>

<style>
	.services-grid {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(150px, 1fr));
		gap: 0.75rem;
	}

	.service-card {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 0.25rem;
		padding: 1rem 0.5rem;
		background: var(--surface);
		border: 2px solid color-mix(in srgb, var(--svc-color) 40%, transparent);
		text-decoration: none;
		transition: all 0.15s ease;
		position: relative;
		overflow: hidden;
	}

	.service-card::before {
		content: '';
		position: absolute;
		top: 0;
		left: 0;
		right: 0;
		height: 2px;
		background: var(--svc-color);
		opacity: 0.6;
	}

	.service-card:hover {
		border-color: var(--svc-color);
		box-shadow: 0 0 12px color-mix(in srgb, var(--svc-color) 30%, transparent);
		transform: translateY(-2px);
	}

	.service-card:hover .service-icon {
		transform: scale(1.2);
	}

	.service-icon {
		font-size: 1.5rem;
		transition: transform 0.15s ease;
	}

	.service-name {
		font-family: var(--font-pixel);
		font-size: 0.4rem;
		color: var(--svc-color);
		text-align: center;
		line-height: 1.4;
	}

	.service-desc {
		font-family: var(--font-body);
		font-size: 0.8rem;
		color: var(--text-muted);
		text-align: center;
	}

	.service-port {
		font-family: var(--font-body);
		font-size: 0.75rem;
		color: color-mix(in srgb, var(--text-muted) 60%, transparent);
	}
</style>
