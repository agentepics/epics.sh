import { readFile } from 'node:fs/promises';
import { fileURLToPath } from 'node:url';

import type { CliReleaseIndex } from './types';

const releasesPath = fileURLToPath(
	new URL('../../../../registry/cli/releases.json', import.meta.url),
);
const markdownPath = fileURLToPath(
	new URL('../../../../registry/cli/cli.md', import.meta.url),
);

export async function getReleaseIndex() {
	const raw = await readFile(releasesPath, 'utf-8');
	try {
		return JSON.parse(raw) as CliReleaseIndex;
	} catch (error) {
		throw new Error(`failed to parse JSON at ${releasesPath}`, { cause: error });
	}
}

export async function getCLIMarkdown() {
	return readFile(markdownPath, 'utf-8');
}
