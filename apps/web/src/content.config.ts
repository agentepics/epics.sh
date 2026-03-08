import { defineCollection, z } from 'astro:content';
import { glob } from 'astro/loaders';

const docs = defineCollection({
	loader: glob({ pattern: '**/*.{md,mdx}', base: './src/content/docs' }),
	schema: z.object({
		title: z.string(),
		description: z.string(),
		section: z.string(),
		order: z.number().default(0),
		navTitle: z.string().optional(),
	}),
});

export const collections = { docs };
