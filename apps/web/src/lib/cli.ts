import { readFile } from 'node:fs/promises';
import { fileURLToPath } from 'node:url';

import type { CliReleaseIndex } from './types';

const releasesPath = fileURLToPath(
	new URL('../../../../registry/cli/releases.json', import.meta.url),
);

export async function getReleaseIndex() {
	const raw = await readFile(releasesPath, 'utf-8');
	return JSON.parse(raw) as CliReleaseIndex;
}
