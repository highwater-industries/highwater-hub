import { sveltekit } from '@sveltejs/kit/vite';
import tailwindcss from '@tailwindcss/vite';
import { defineConfig } from 'vite';

export default defineConfig({
	plugins: [tailwindcss(), sveltekit()],
	server: {
		proxy: {
			// During development, proxy /api requests to the Go server
			'/api': {
				target: 'http://localhost:3141',
				changeOrigin: true
			}
		}
	}
});
