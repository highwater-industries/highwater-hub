import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';

export default defineConfig({
	plugins: [sveltekit()],
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
