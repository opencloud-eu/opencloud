import * as fs from 'fs'
import * as cheerio from 'cheerio'
import { stdin } from 'process'

process.on('unhandledRejection', (reason, promise) => {
  console.error('Unhandled Rejection at:', promise, 'reason:', reason);
  process.exit(1);
});

async function run() {
  try {
    const faviconFile = process.argv[2];
    if (!faviconFile) {
      throw new Error("No favicon file provided");
    }

    const favicon = fs.readFileSync(faviconFile).toString('base64');

    let html = '';
    for await (const chunk of stdin) {
      html += chunk;
    }

    if (!html) {
      throw new Error("No HTML received from stdin");
    }

    const $ = cheerio.load(html);
    $('head').append(`<link rel="icon" href="data:image/png;base64,${favicon}">`);

    process.stdout.write($.html() + "\n");
  } catch (e) {
    console.error(`Error occurred while post-processing HTML: ${e.message}`);
    process.exit(1);
  }
}

run();
