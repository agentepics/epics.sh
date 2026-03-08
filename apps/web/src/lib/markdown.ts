function escapeHtml(value: string) {
	return value
		.replaceAll('&', '&amp;')
		.replaceAll('<', '&lt;')
		.replaceAll('>', '&gt;')
		.replaceAll('"', '&quot;')
		.replaceAll("'", '&#39;');
}

function renderInline(value: string) {
	const escaped = escapeHtml(value);
	return escaped
		.replace(/\[([^\]]+)\]\(([^)]+)\)/g, '<a href="$2">$1</a>')
		.replace(/`([^`]+)`/g, '<code>$1</code>');
}

export function renderMarkdown(markdown: string) {
	const lines = markdown.replaceAll('\r\n', '\n').split('\n');
	const html: string[] = [];
	const paragraph: string[] = [];
	let bulletItems: string[] = [];
	let orderedItems: string[] = [];
	let codeLines: string[] = [];
	let inCodeBlock = false;

	const flushParagraph = () => {
		if (paragraph.length === 0) return;
		html.push(`<p>${renderInline(paragraph.join(' '))}</p>`);
		paragraph.length = 0;
	};

	const flushBullets = () => {
		if (bulletItems.length === 0) return;
		html.push(`<ul>${bulletItems.map((item) => `<li>${renderInline(item)}</li>`).join('')}</ul>`);
		bulletItems = [];
	};

	const flushOrdered = () => {
		if (orderedItems.length === 0) return;
		html.push(`<ol>${orderedItems.map((item) => `<li>${renderInline(item)}</li>`).join('')}</ol>`);
		orderedItems = [];
	};

	const flushCodeBlock = () => {
		if (!inCodeBlock) return;
		html.push(`<pre><code>${escapeHtml(codeLines.join('\n'))}</code></pre>`);
		codeLines = [];
		inCodeBlock = false;
	};

	for (const line of lines) {
		if (line.startsWith('```')) {
			flushParagraph();
			flushBullets();
			flushOrdered();
			if (inCodeBlock) {
				flushCodeBlock();
			} else {
				inCodeBlock = true;
			}
			continue;
		}

		if (inCodeBlock) {
			codeLines.push(line);
			continue;
		}

		if (line.trim() === '') {
			flushParagraph();
			flushBullets();
			flushOrdered();
			continue;
		}

		const headingMatch = line.match(/^(#{1,3})\s+(.*)$/);
		if (headingMatch) {
			flushParagraph();
			flushBullets();
			flushOrdered();
			const level = headingMatch[1].length;
			html.push(`<h${level}>${renderInline(headingMatch[2])}</h${level}>`);
			continue;
		}

		const bulletMatch = line.match(/^- (.*)$/);
		if (bulletMatch) {
			flushParagraph();
			flushOrdered();
			bulletItems.push(bulletMatch[1]);
			continue;
		}

		const orderedMatch = line.match(/^\d+\.\s+(.*)$/);
		if (orderedMatch) {
			flushParagraph();
			flushBullets();
			orderedItems.push(orderedMatch[1]);
			continue;
		}

		paragraph.push(line.trim());
	}

	flushParagraph();
	flushBullets();
	flushOrdered();
	flushCodeBlock();

	return html.join('\n');
}
