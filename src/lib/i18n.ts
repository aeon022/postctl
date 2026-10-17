// The site is authored bilingually (<span class="lang-en">…</span><span class="lang-de">…</span>).
// At render time every page is cut down to ONE language, so /… is English and /de/… is German,
// each a clean single-language document (own lang attribute, hreflang, canonical).

export type Lang = 'en' | 'de';

export const langOf = (pathname: string): Lang => (pathname === '/de' || pathname.startsWith('/de/') ? 'de' : 'en');

/** Remove every element (any tag, nesting-aware) that is marked for the other language. */
export function filterLang(html: string, lang: Lang): string {
	const drop = lang === 'en' ? 'de' : 'en';
	const start = new RegExp(`<([a-zA-Z][a-zA-Z0-9]*)\\b[^>]*\\bclass="[^"]*\\blang-${drop}\\b[^"]*"[^>]*>`, 'g');
	let out = '';
	let pos = 0;
	let m: RegExpExecArray | null;
	while ((m = start.exec(html))) {
		const tag = m[1];
		const tags = new RegExp(`<${tag}\\b[^>]*>|</${tag}\\s*>`, 'g');
		tags.lastIndex = m.index + m[0].length;
		let depth = 1;
		let e: RegExpExecArray | null;
		while (depth > 0 && (e = tags.exec(html))) depth += e[0].startsWith('</') ? -1 : e[0].endsWith('/>') ? 0 : 1;
		out += html.slice(pos, m.index);
		pos = tags.lastIndex;
		start.lastIndex = pos;
	}
	return out + html.slice(pos);
}

/** German pages live under /de/: prefix internal page links (not assets, not external). */
export function localizeLinks(html: string, lang: Lang): string {
	if (lang === 'en') return html;
	return html.replace(/href="\/(docs|impressum|datenschutz)?\/?(#[^"]*)?"/g, (_m, p, h) => `href="/de/${p ? p + '/' : ''}${h ?? ''}"`);
}

const norm = (p: string) => (p.endsWith('/') ? p : p + '/');

/** The same page in the other language. */
export function altPath(pathname: string): string {
	const p = norm(pathname);
	return langOf(p) === 'de' ? norm(p.replace(/^\/de/, '') || '/') : '/de' + p;
}

export function localize(html: string, lang: Lang, pathname: string): string {
	return localizeLinks(filterLang(html, lang), lang).replaceAll('__ALT_LANG_URL__', altPath(pathname));
}
