import { defineConfig } from 'i18next-cli';

export default defineConfig({
  locales: ['en'],
  extract: {
    currentLocale: 'en',
    input: ['../src/**/*.{js,jsx,ts,tsx}'],
    output: 'translation.{{language}}.json',
  },
});
