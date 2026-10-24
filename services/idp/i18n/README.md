# IDP Translations

## Overview

The IDP service uses i18next for internationalization with gettext (`.po`)
files as the source of truth for translations. The workflow is:

```text
Source code t() calls → extract keys → .pot template
→ merge into .po → convert to JSON → app bundles
```

## File Structure

- `i18n/*.po` — translation files per language (committed)
- `i18n/translation.pot` — extracted keys template (committed)
- `i18n/translation.en.json` — intermediate EN keys (gitignored)
- `src/locales/*/translation.json` — JSON translations at runtime (committed)
- `src/locales/locales.json` — language metadata for locale selector (committed)

## Adding New Translation Keys

### 1. Add a key in source code

```jsx
t("konnect.new.key", "Default fallback text")
```

The first argument is the i18next key path (dot notation for nested JSON). The
second argument is the default/fallback English text — this is what translators
see in `.po` files as `msgid`.

### 2. Extract new keys into the `.pot` template

```bash
make -C i18n extract
```

This runs `i18next-cli extract` to scan all `src/**/*.{js,jsx,ts,tsx}` files,
writes extracted English keys to `i18n/translation.en.json`, then converts
them into `translation.pot`.

### 3. Merge new keys into existing `.po` files

```bash
make -C i18n merge
```

This runs `msgmerge` on each `.po` file to add any new keys from the updated
`.pot` template. Translators then fill in translations for the new entries.

### 4. Convert `.po` files to JSON for the app

```bash
make -C i18n json
```

This runs `i18next-conv` on each `.po` file to produce
`src/locales/{lang}/translation.json`. This step is also run automatically by
`make build`.

## Runtime Loading

The app loads translations via dynamic webpack imports:

```typescript
// src/i18n.ts
import(`./locales/${language}/translation.json`)
```

When the user selects a language, i18next fetches the corresponding JSON file.
The locale selector reads available languages from `src/locales/locales.json`.

## Build and CI

- **`make build`** → runs `make json` to convert `.po` files to JSON before
  webpack bundling
- **CI (Woodpecker)** runs `make node-generate-prod` which calls
  `pnpm install && pnpm build` — no translation extraction happens in CI, only
  the committed JSON files are bundled

## Tools

- `i18next-cli` — extracts keys from source code (AST-based)
- `i18next-conv` — converts between `.po` and i18next JSON
- `msgmerge` / `msgcat` — standard gettext merge/collate tools
- `gettext-parser` — cleans headers in generated POT file

## Configuration

- `i18n/i18next.config.ts` — extraction config (input globs, output path)
- `src/i18n.ts` — runtime setup (detection, fallbacks, backend loader)
