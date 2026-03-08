import { readdir, readFile } from 'node:fs/promises';
import { fileURLToPath } from 'node:url';

import type { EpicRecord } from './types';

const epicsDir = fileURLToPath(new URL('../../../../registry/epics/', import.meta.url));

async function readJson<T>(path: string) {
	const raw = await readFile(path, 'utf-8');
	try {
		return JSON.parse(raw) as T;
	} catch (error) {
		throw new Error(`failed to parse JSON at ${path}`, { cause: error });
	}
}

export async function getEpics(): Promise<EpicRecord[]> {
	const files = (await readdir(epicsDir))
		.filter((entry: string) => entry.endsWith('.json'))
		.sort((left: string, right: string) => left.localeCompare(right));

	const entries = await Promise.all(
		files.map((entry: string) => readJson<EpicRecord>(`${epicsDir}${entry}`)),
	);

	return entries.sort((left: EpicRecord, right: EpicRecord) => {
		if (left.featured && !right.featured) return -1;
		if (!left.featured && right.featured) return 1;
		return left.title.localeCompare(right.title);
	});
}

export function getEpicSourcePath(epic: EpicRecord) {
	return `${epic.source.repo}${epic.source.path ? `/${epic.source.path}` : ''}`;
}

export function buildInstallerCommand(epic: EpicRecord, platform: 'unix' | 'windows') {
	const sourcePath = getEpicSourcePath(epic);
	if (platform === 'windows') {
		return `powershell -NoProfile -Command "& ([scriptblock]::Create((irm https://epics.sh/install.ps1))) -Epic '${sourcePath}'"`;
	}

	return `curl -fsSL https://epics.sh/install.sh | sh -s -- ${sourcePath}`;
}
