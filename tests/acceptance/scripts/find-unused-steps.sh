#!/usr/bin/env bash

# Finds Behat step definitions (in tests/acceptance/bootstrap/*.php) that are never
# used by any scenario in tests/acceptance/features/**/*.feature.
#
# Uses Behat's own RegexPatternPolicy/TurnipPatternPolicy to compile each step
# definition pattern into the exact same regex Behat itself uses for matching,
# so results are accurate (no custom regex re-implementation).
#
# This is a reporting tool, not a hard gate: some flagged steps may still be in
# use if they are called programmatically from other PHP context code rather
# than from a .feature file, which this script cannot detect.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/../../.." && pwd)"

php -d error_reporting=E_ALL -- "${ROOT_DIR}" <<'PHP'
<?php

$rootDir = $argv[1];
require $rootDir . '/vendor-bin/behat/vendor/autoload.php';

use Behat\Behat\Definition\Pattern\Policy\RegexPatternPolicy;
use Behat\Behat\Definition\Pattern\Policy\TurnipPatternPolicy;

$bootstrapDir = $rootDir . '/tests/acceptance/bootstrap';
$featuresDir = $rootDir . '/tests/acceptance/features';

$attrRegex = "/^\s*#\[(Given|When|Then)\('(.*)'\)\]\s*$/";

$definitions = [];
foreach (glob($bootstrapDir . '/*.php') as $file) {
    $lines = file($file);
    foreach ($lines as $lineNo => $line) {
        if (preg_match($attrRegex, rtrim($line, "\n"), $m)) {
            $pattern = str_replace(["\\'", '\\\\'], ["'", '\\'], $m[2]);
            $definitions[] = [
                'file' => basename($file),
                'line' => $lineNo + 1,
                'keyword' => $m[1],
                'pattern' => $pattern,
            ];
        }
    }
}

$stepLineRegex = '/^\s*(Given|When|Then|And|But)\s+(.*)$/';
$tableRowRegex = '/^\s*\|(.*)\|\s*$/';

/**
 * Splits a Gherkin table row into trimmed cell values.
 *
 * @param string $row
 *
 * @return string[]
 */
function splitTableRow(string $row): array {
    return array_map('trim', explode('|', trim($row, "| \t")));
}

$uniqueTexts = [];
$featureFiles = new RecursiveIteratorIterator(
    new RecursiveDirectoryIterator($featuresDir, FilesystemIterator::SKIP_DOTS)
);
foreach ($featureFiles as $fileInfo) {
    if ($fileInfo->getExtension() !== 'feature') {
        continue;
    }
    $lines = file($fileInfo->getPathname());

    // pendingOutlineSteps holds raw "<placeholder>" step texts belonging to the
    // Scenario Outline currently being parsed, waiting to be substituted once
    // their Examples: table(s) are found.
    $pendingOutlineSteps = [];
    $inOutline = false;

    $i = 0;
    $count = count($lines);
    while ($i < $count) {
        $line = rtrim($lines[$i], "\n");

        if (preg_match('/^\s*Scenario Outline\s*:/i', $line)) {
            $inOutline = true;
            $pendingOutlineSteps = [];
            $i++;
            continue;
        }

        if (preg_match('/^\s*(Scenario\s*:|Scenario Outline\s*:|Feature\s*:)/i', $line) && !preg_match('/^\s*Scenario Outline\s*:/i', $line)) {
            // A plain Scenario (or a new Feature) ends the current outline context.
            $inOutline = false;
            $pendingOutlineSteps = [];
        }

        if (preg_match($stepLineRegex, $line, $m)) {
            $text = rtrim($m[2]);
            if ($inOutline && strpos($text, '<') !== false) {
                $pendingOutlineSteps[] = $text;
            } else {
                $uniqueTexts[$text] = true;
            }
            $i++;
            continue;
        }

        if ($inOutline && preg_match('/^\s*Examples\s*:/i', $line)) {
            $i++;
            // skip blank/tag lines before the header row
            while ($i < $count && trim($lines[$i]) === '') {
                $i++;
            }
            if ($i >= $count || !preg_match($tableRowRegex, rtrim($lines[$i], "\n"), $hm)) {
                continue;
            }
            $header = splitTableRow($hm[1]);
            $i++;
            while ($i < $count && preg_match($tableRowRegex, rtrim($lines[$i], "\n"), $rm)) {
                $row = splitTableRow($rm[1]);
                $values = count($header) === count($row) ? array_combine($header, $row) : false;
                if ($values !== false) {
                    foreach ($pendingOutlineSteps as $template) {
                        $substituted = preg_replace_callback(
                            '/<([^>]+)>/',
                            function ($mm) use ($values) {
                                return $values[$mm[1]] ?? $mm[0];
                            },
                            $template
                        );
                        $uniqueTexts[$substituted] = true;
                    }
                }
                $i++;
            }
            continue;
        }

        $i++;
    }
}
$uniqueTexts = array_keys($uniqueTexts);

$regexPolicy = new RegexPatternPolicy();
$turnipPolicy = new TurnipPatternPolicy();

$unused = [];
foreach ($definitions as $def) {
    try {
        $regex = $regexPolicy->supportsPattern($def['pattern'])
            ? $regexPolicy->transformPatternToRegex($def['pattern'])
            : $turnipPolicy->transformPatternToRegex($def['pattern']);
    } catch (\Throwable $e) {
        fwrite(STDERR, "Skipping invalid pattern in {$def['file']}:{$def['line']}: {$e->getMessage()}\n");
        continue;
    }

    $matched = false;
    foreach ($uniqueTexts as $text) {
        if (@preg_match($regex, $text) === 1) {
            $matched = true;
            break;
        }
    }
    if (!$matched) {
        $unused[] = $def;
    }
}

echo "Checked " . count($definitions) . " step definitions against " . count($uniqueTexts) . " unique feature step texts.\n";
echo "Found " . count($unused) . " potentially unused step definitions:\n\n";
foreach ($unused as $u) {
    echo "{$u['file']}:{$u['line']} [{$u['keyword']}] {$u['pattern']}\n";
}
PHP
