import type { APIRoute, GetStaticPaths } from 'astro';
import { getCollection } from 'astro:content';

import { docSlug } from '../../lib/docs';

export const getStaticPaths: GetStaticPaths = async () => {
	const docs = await getCollection('docs');
	return docs.map((entry) => ({
		params: {
			slug: docSlug(entry.id),
		},
		props: { entry },
	}));
};

export const GET: APIRoute = ({ props }) => {
	const { entry } = props as {
		entry: Awaited<ReturnType<typeof getCollection<'docs'>>>[number];
	};

	const frontmatter = [
		'---',
		`title: ${entry.data.title}`,
		`description: ${entry.data.description}`,
		'---',
	].join('\n');

	const body = entry.body ?? '';
	const content = `${frontmatter}\n\n${body}`;

	return new Response(content, {
		headers: { 'Content-Type': 'text/markdown; charset=utf-8' },
	});
};
