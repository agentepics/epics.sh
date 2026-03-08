export type HostId = 'claude' | 'gemini' | 'opencode';

export interface Maintainer {
	name: string;
	url?: string;
}

export interface EpicRecord {
	slug: string;
	title: string;
	summary: string;
	description: string;
	category: string;
	tags: string[];
	featured?: boolean;
	source: {
		repo: string;
		path?: string;
	};
	version: string;
	digest: string;
	updatedAt: string;
	validationStatus: 'reviewed' | 'draft';
	maintainers: Maintainer[];
	features: string[];
	skillMd: string;
	epicMd: string;
}

export interface CliDownload {
	label: string;
	target: string;
	url: string;
	checksum: string;
}

export interface CliRelease {
	version: string;
	channel: string;
	publishedAt: string;
	highlights: string[];
	downloads: CliDownload[];
}

export interface CliReleaseIndex {
	currentVersion: string;
	releases: CliRelease[];
}
