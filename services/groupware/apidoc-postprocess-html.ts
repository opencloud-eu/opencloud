import * as fs from 'fs'
import * as cheerio from 'cheerio'

const faviconFile = process.argv[2]
const favicon = fs.readFileSync(faviconFile).toString('base64')

let html = ''
process.stdin.on('data', (chunk) => {
  html += chunk.toString()
})
process.stdin.on('end', () => {
  try {
    const $ = cheerio.load(html)
    $('head').append(`<link rel="icon" href="data:image/png;base64,${favicon}">`)
    process.stdout.write($.html())
    process.stdout.write("\n")
  } catch (error) {
    if (error instanceof Error) {
      console.error(`Error occured while post-processing HTML: ${error.message}`)
    } else {
      console.error("Unknown error occurred")
    }
  }
});

