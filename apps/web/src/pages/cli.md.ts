import type { APIRoute } from 'astro';

import { getCLIMarkdown } from '../lib/cli';

export const GET: APIRoute = async () => {
	const content = await getCLIMarkdown();

	return new Response(content, {
		headers: { 'Content-Type': 'text/markdown; charset=utf-8' },
	});
};
