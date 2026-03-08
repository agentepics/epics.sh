import { readdir, readFile } from 'node:fs/promises';
import { fileURLToPath } from 'node:url';

import type { EpicRecord } from './types';

const epicsDir = fileURLToPath(new URL('../../../../registry/epics/', import.meta.url));

async function readJson<T>(path: string) {
	const raw = await readFile(path, 'utf-8');
	return JSON.parse(raw) as T;
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

export async function getEpicBySlug(slug: string): Promise<EpicRecord | undefined> {
	const epics = await getEpics();
	return epics.find((entry: EpicRecord) => entry.slug === slug);
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

export function matchesQuery(epic: EpicRecord, query: string) {
	const haystack = [
		epic.title,
		epic.summary,
		epic.description,
		epic.category,
		...epic.tags,
	]
		.join(' ')
		.toLowerCase();

	return haystack.includes(query.toLowerCase());
}
