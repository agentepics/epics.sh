import type { APIRoute } from 'astro';

import { getReleaseIndex } from '../lib/cli';
import { formatDate } from '../lib/site';

export const GET: APIRoute = async () => {
	const releases = await getReleaseIndex();
	if (releases.releases.length === 0) {
		return new Response('No CLI releases found.\n', {
			headers: { 'Content-Type': 'text/markdown; charset=utf-8' },
			status: 500,
		});
	}

	const current = releases.releases[0];
	const installSection = [
		'## Install',
		'',
		'### macOS/Linux',
		'',
		'```bash',
		'curl -fsSL https://epics.sh/install.sh | sh',
		'```',
		'',
		'### Windows',
		'',
		'```powershell',
		'iwr https://epics.sh/install.ps1 -useb | iex',
		'```',
	].join('\n');

	const downloadSection = [
		'## Downloads',
		'',
		...current.downloads.map((download) => [
			`### ${download.label}`,
			'',
			`- Target: \`${download.target}\``,
			`- Artifact: ${download.url}`,
			`- Checksum: \`${download.checksum}\``,
		].join('\n')),
	].join('\n\n');

	const changelogSection = [
		'## Changelog',
		'',
		...releases.releases.map((release) => [
			`### ${release.version} (${formatDate(release.publishedAt)})`,
			'',
			...release.highlights.map((item) => `- ${item}`),
		].join('\n')),
	].join('\n\n');

	const content = [
		'---',
		'title: epics CLI',
		'description: Install the epics CLI, download platform builds, and read the changelog.',
		'---',
		'',
		'# epics CLI',
		'',
		'The website reflects the real CLI, not a separate web-only workflow. Install the binary, then use `epics install`, `epics validate`, `epics resume`, and host setup commands from the same model the directory renders.',
		'',
		installSection,
		'',
		downloadSection,
		'',
		changelogSection,
	].join('\n');

	return new Response(content, {
		headers: { 'Content-Type': 'text/markdown; charset=utf-8' },
	});
};
