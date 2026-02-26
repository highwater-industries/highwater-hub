import adapter from '@sveltejs/adapter-static';

/** @type {import('@sveltejs/kit').Config} */
const config = {
	kit: {
		adapter: adapter({
			// Build output goes here — Go will embed this directory
			pages: 'build',
			assets: 'build',
			fallback: 'index.html'  // SPA mode: all routes fall back to index.html
		})
	}
};

export default config;
