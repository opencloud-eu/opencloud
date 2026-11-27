import legacy from "@vitejs/plugin-legacy";
import react from "@vitejs/plugin-react";
import type { ConfigEnv, Plugin } from "vite";
import checker from "vite-plugin-checker";
import type { ViteUserConfig } from "vitest/config";
import { defineConfig } from "vitest/config";

/*
 * The server writes a fresh nonce into the document on every request and the
 * policy admits scripts carrying it, so the scripts Vite emits need the
 * placeholder to substitute. Build only: the dev server sends no policy.
 *
 * The module entry is nonced even though the policy already admits it by origin,
 * because a module's dependency fetches inherit the nonce of the element that
 * started them. Without one, plugin-legacy's probe inside the entry chunk — an
 * import of a `data:` URL, which no origin admits — is refused, the entry never
 * evaluates and the page renders nothing at all.
 */
const addScriptCSPNoncePlaceholderPlugin = (): Plugin => {
  return {
    name: "add-script-nonce-placeholderP-plugin",
    apply: "build",
    transformIndexHtml: {
      order: "post",
      handler(htmlData: string) {
        return htmlData
          .replaceAll(/<script nomodule>/gi, `<script nomodule nonce="__CSP_NONCE__">`)
          .replaceAll(/<script type="module">/gi, `<script type="module" nonce="__CSP_NONCE__">`)
          .replaceAll(/<script type="module" crossorigin/gi, `<script type="module" crossorigin nonce="__CSP_NONCE__"`)
          .replaceAll(
            /<script nomodule crossorigin id="vite-legacy-entry"/gi,
            `<script nomodule crossorigin id="vite-legacy-entry" nonce="__CSP_NONCE__"`,
          );
      },
    },
  };
};

/*
 * A font is fetched when something paints text with it, not when the stylesheet
 * declaring it arrives, so these two waited on the route chunks executing and
 * started well over a second after the CSS had been parsed. Preloading makes the
 * request unconditional and moves them alongside the document instead.
 *
 * Only the two the first screens actually ask for: Roboto latin at 400 and 500.
 * The other weights and subsets stay demand-driven, which is what they should be
 * — a preload the page never uses is a download spent on nothing. `crossorigin`
 * is required even same-origin, because fonts are fetched in CORS mode and a
 * preload whose mode differs is fetched a second time rather than reused.
 */
const preloadFirstPaintFontsPlugin = (): Plugin => {
  const wanted = /^roboto-latin-(?:400|500)-normal-[^.]+\.woff2$/;

  return {
    name: "preload-first-paint-fonts",
    apply: "build",
    transformIndexHtml: {
      order: "post",
      handler(htmlData: string, ctx) {
        const fonts = Object.keys(ctx.bundle ?? {})
          .filter((file) => wanted.test(file.split("/").pop() ?? ""))
          .sort();

        // Names carry a content hash, so they are read off the bundle rather
        // than written down. Nothing to preload means the font package changed
        // its file names and this silently stopped working, which is worth a
        // failed build rather than a slower page nobody measures.
        if (fonts.length !== 2) {
          throw new Error(
            `expected two first-paint fonts to preload, found ${fonts.length}: ` +
              `${fonts.join(", ") || "none"}. @fontsource/roboto has changed how it ` +
              `names its files, so this plugin needs updating or removing.`,
          );
        }

        const links = fonts
          .map((file) => `<link rel="preload" href="./${file}" as="font" type="font/woff2" crossorigin>`)
          .join("");

        return htmlData.replace(/(<meta charset="utf-8"\s*\/?>)/i, `$1${links}`);
      },
    },
  };
};

/*
 * Vite ships index.html as written, so the notes explaining the loading state
 * would be served to every visitor. They are for whoever edits the file, so
 * they are dropped here along with the indentation of the style block that
 * paints it.
 *
 * The one comment that stays is the placeholder the server replaces with the
 * background preload. It is a comment so that it stays invisible when nothing
 * substitutes it, which makes it the one piece of this document whose survival
 * is load-bearing, hence the check rather than trust.
 *
 * Collapsing the style's whitespace is safe only while none of its declarations
 * carry a string where spacing counts, which today means an empty `content`.
 */
const trimIndexHTMLPlugin = (): Plugin => {
  const preloadPlaceholder = "__BACKGROUND_PRELOAD__";

  return {
    name: "trim-index-html",
    apply: "build",
    transformIndexHtml: {
      order: "post",
      handler(htmlData: string) {
        const trimmed = htmlData
          .replace(/<!--[\s\S]*?-->/g, (comment) => (comment.includes(preloadPlaceholder) ? comment : ""))
          .replace(
            /<style([^>]*)>([\s\S]*?)<\/style>/g,
            (_all, attributes: string, css: string) =>
              `<style${attributes}>${css
                .replace(/\/\*[\s\S]*?\*\//g, "")
                .replace(/\s+/g, " ")
                .trim()}</style>`,
          )
          .replace(/^[ \t]*\r?\n/gm, "");

        if (!trimmed.includes(preloadPlaceholder)) {
          throw new Error(`trim-index-html: ${preloadPlaceholder} did not survive, the background preload is dead`);
        }

        return trimmed;
      },
    },
  };
};

// Where the build writes everything it emits, named once because the file name
// templates below have to repeat it: they replace the whole path, not just the
// name, so `assetsDir` alone no longer decides where a chunk lands.
const assetsDir = "static/assets";

/*
 * A chunk is named after the module that anchors it, so it arrives spelled the
 * way that source file is, and the shared runtime arrives named after the
 * bundler that emitted it. Neither is what we want on disk: everything served
 * from here is lowercase and hyphenated.
 */
const chunkName = (name: string): string =>
  (name === "rolldown-runtime" ? "runtime" : name)
    .replace(/([a-z\d])([A-Z])/g, "$1-$2")
    .replace(/[^a-zA-Z0-9]+/g, "-")
    .toLowerCase();

export default defineConfig((env: ConfigEnv): ViteUserConfig => {
  return {
    resolve: {
      /*
       * Six packages here reach for a react-is of their own, across three major
       * versions, and this collapses them to the one at the root. Which one that
       * is matters more than the count: react-is decides what an element is by
       * the symbol React stamps on it, and that symbol changed name in React 19,
       * so a react-is from the wrong major answers no to every element this app
       * creates. prop-types hands its element check straight to react-is, which
       * is enough to make every `children` prop in the tree look invalid. The
       * version is pinned to React's own major by `resolutions` in package.json
       * and has to move when React does.
       *
       * clsx arrives three times over as well, nested under @mui/material,
       * @mui/system and @mui/utils. Nothing disagrees about the version there —
       * all three ask for the same range and every install is the same release —
       * so the copies are an install layout, and one of them will do.
       */
      dedupe: ["react-is", "clsx"],
    },
    build: {
      outDir: "build",
      assetsDir,
      sourcemap: true,
      // terser is smaller than the default minifier here, and it is already a
      // dependency because plugin-legacy minifies the legacy chunks with it.
      // The second compress pass is worth a further couple of KB and applies to
      // those chunks too.
      minify: "terser",
      terserOptions: {
        compress: {
          passes: 2,
        },
      },
      /*
       * No chunk groups on purpose. Collecting every installed package into one
       * vendor chunk, which is what the removed vendor chunk plugin did, makes
       * each route pay for every other route's dependencies: it cost 76 kB on
       * the sign-out screen and saved 1 kB on sign-in, and put a 528 kB chunk
       * past the size warning. Left alone, the splitter keeps a chunk per lazy
       * route and shares what more than one of them reaches for.
       */
      rolldownOptions: {
        output: {
          // base64 is the default and puts capitals in every hash, including
          // the ones on the font files. base36 keeps them lowercase.
          hashCharacters: "base36",
          entryFileNames: (chunk) => `${assetsDir}/${chunkName(chunk.name)}-[hash].js`,
          chunkFileNames: (chunk) => `${assetsDir}/${chunkName(chunk.name)}-[hash].js`,
        },
      },
    },
    base: "./",
    server: {
      port: 3001,
      strictPort: true,
      host: "127.0.0.1",
      hmr: {
        protocol: "ws",
        host: "127.0.0.1",
        clientPort: 3001,
      },
    },
    plugins: [
      react(),
      legacy({
        targets: ["edge 18"],
      }),
      /*
       * Types, lint, formatting and import order in the dev overlay, and a gate
       * on the production build. Skipped under test, where vitest does its own
       * transform and the suite is the gate. The command and flags match the bun
       * script and the Makefile target, so the overlay and the command line
       * agree on what fails: biome reports most rules as warnings, which are
       * errors for this project.
       */
      env.mode !== "test" &&
        checker({
          typescript: true,
          biome: {
            command: "check",
            flags: "--error-on-warnings .",
          },
        }),
      addScriptCSPNoncePlaceholderPlugin(),
      preloadFirstPaintFontsPlugin(),
      // Last, so it trims the document every plugin above has finished with.
      trimIndexHTMLPlugin(),
    ],
    test: {
      globals: true,
      environment: "jsdom",
      setupFiles: "./tests/setup.js",
      /*
       * Above the library's own wait budget, which setup.js raises, so a test
       * which is genuinely stuck fails with the DOM that testing-library prints
       * rather than with a bare timeout from here.
       */
      testTimeout: 10000,
    },
  };
});
