export function docSlug(id: string): string {
	return id.replace(/\.mdx?$/, '').replace(/\/index$/, '');
}
