// @ts-check
import { defineConfig } from 'astro/config';
import sitemap from '@astrojs/sitemap';

// https://astro.build/config
export default defineConfig({
  site: 'https://postctl.sh',
  integrations: [sitemap({ filter: (page) => !page.endsWith('/tui/') && !page.endsWith('/404/'), i18n: { defaultLocale: 'en', locales: { en: 'en', de: 'de' } } })],
});
