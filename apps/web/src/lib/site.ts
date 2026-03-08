import type { HostId } from './types';

export const siteTitle = 'epics.sh';
export const siteDescription =
	'Directory and CLI home for Agent Epics: stateful, long-running workflows for coding agents.';

export const navLinks = [
	{ href: '/', label: 'Home' },
	{ href: '/docs', label: 'Docs' },
	{ href: '/cli', label: 'CLI' },
];

export const hostCatalog: Record<
	HostId,
	{
		label: string;
		headline: string;
		posture: string;
		logo: string;
	}
> = {
	claude: {
		label: 'Claude Code',
		headline: 'Supported host',
		posture: 'Claude Code is offered because it can satisfy the same autonomy contract the epics CLI promises everywhere else.',
		logo: '/agents/claude-code.svg',
	},
	gemini: {
		label: 'Gemini CLI',
		headline: 'Supported host',
		posture: 'Gemini CLI is offered because it can satisfy the same autonomy contract through the epics CLI and adapter layer.',
		logo: '/agents/gemini.svg',
	},
	opencode: {
		label: 'OpenCode',
		headline: 'Supported host',
		posture: 'OpenCode is offered because it can satisfy the same autonomy contract instead of needing a downgraded compatibility story.',
		logo: '/agents/opencode.svg',
	},
};

export function formatDate(value: string) {
	return new Intl.DateTimeFormat('en', {
		month: 'short',
		day: 'numeric',
		year: 'numeric',
	}).format(new Date(value));
}
