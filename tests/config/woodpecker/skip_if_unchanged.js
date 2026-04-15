const fs = require("node:fs");
const { minimatch } = require("minimatch");

const type = process.argv[2];

function getSkipPatterns(type) {
  const base = [
    ".github/**",
    ".vscode/**",
    "docs/**",
    "deployments/**",
    "CHANGELOG.md",
    "CONTRIBUTING.md",
    "LICENSE",
    "README.md",
  ];

  const unit = ["**/*_test.go"];
  const acceptance = ["tests/acceptance/**"];

  if (
    type === "acceptance-tests" ||
    type === "e2e-tests" ||
    type === "lint"
  ) {
    return [...base, ...unit];
  }

  if (type === "unit-tests") {
    return [...base, ...acceptance];
  }

  if (
    type === "build-binary" ||
    type === "build-docker" ||
    type === "litmus"
  ) {
    return [...base, ...unit, ...acceptance];
  }

  if (type === "cache" || type === "base") {
    return base;
  }

  return [];
}

function main() {
  const skipPatterns = getSkipPatterns(type);

  if (skipPatterns.length === 0) {
    process.exit(0);
  }

  const rawFiles = process.env.CI_PIPELINE_FILES;

  // If CI_PIPELINE_FILES is not set, we assume the pipeline should run
  if (!rawFiles) {
    console.log("[INFO] CI_PIPELINE_FILES not set → run pipeline");
    process.exit(0);
  }

  let changedFiles;
  try {
    changedFiles = JSON.parse(rawFiles);
  } catch {
    console.error("[ERROR] Failed to parse CI_PIPELINE_FILES:", rawFiles);
    process.exit(1);
  }

  if (changedFiles.length === 0) {
    process.exit(0);
  }

  console.log("[INFO] Changed files:", changedFiles);

  const onlySkippable = changedFiles.every((file) =>
    skipPatterns.some((pattern) =>
      minimatch(file, pattern, { dot: true, matchBase: true })
    )
  );

  if (onlySkippable) {
    console.log("[INFO] Only skippable files changed → SKIP WORKFLOW");
    fs.appendFileSync(".woodpecker.env", "SKIP_WORKFLOW=true\n");
  } else {
    console.log("[INFO] Relevant changes detected → run pipeline");
  }
}

main();