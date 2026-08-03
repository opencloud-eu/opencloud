---
name: bumping-reva
description: Use when the user asks to bump, update, or upgrade the OpenCloud reva dependency to a specific version (e.g. "reva bump to 2.48.0", "bump reva to v2.48.0"). Covers editing go.mod, re-vendoring, bumping the OpenCloud LatestTag, the single commit, and opening the PR against main.
---

# Bumping Reva

## Overview

Bumping reva updates the pinned [opencloud-eu/reva](https://github.com/opencloud-eu/reva) dependency (`github.com/opencloud-eu/reva/v2`) to a tagged release, re-vendors the module graph, and bumps the OpenCloud dev version. It touches `go.mod`, `go.sum`, `vendor/**` and `pkg/version/version.go`, lands as one commit, and ships as a PR whose body is the reva changelog for that version.

Template PR: https://github.com/opencloud-eu/opencloud/pull/3127

## Prerequisites

- **The reva release must already be tagged.** A reva release is cut by merging its release PR (title `🎉 Release X.Y.Z`, branch `next-release/main`). The **user merges that PR themselves** — this skill starts *after* the tag `vX.Y.Z` exists. Verify the tag before doing anything (step 1); if it's missing, stop and ask the user to merge the reva release PR first.
- **`gh` (GitHub CLI), authenticated** — tag lookup and PR creation go through `gh api` / `gh pr`. Verify with `gh auth status`; if it fails, ask the user to run `gh auth login` (suggest `! gh auth login` so it runs in-session).
- `gh` needs the system keyring, so run all `gh` commands with the sandbox disabled.
- **Go toolchain + network** — `go get` / `go mod tidy` / `go mod vendor` hit the Go module proxy. Run them with the sandbox disabled (network access).
- `git` and `base64` (decoding the changelog) — standard on macOS/Linux.

## Inputs

Two versions are needed — **ask the user for whichever they did not give**:

- `REVA_VERSION` — the new reva tag, always normalized to a leading `v` (e.g. `v2.48.0`).
- `OC_VERSION` — the OpenCloud target for `LatestTag`, **without** the `+dev` suffix (e.g. `7.4.0`). This is a deliberate release target, not a mechanical `+1` — never guess it, ask if unclear.

## What changes

| File                     | What                                                                 |
| ------------------------ | ------------------------------------------------------------------- |
| `go.mod`                 | `github.com/opencloud-eu/reva/v2` → `REVA_VERSION` (+ indirect deps pulled by `go mod tidy`) |
| `go.sum`                 | updated by `go get` / `go mod tidy`                                  |
| `vendor/**`              | re-vendored by `go mod vendor` (incl. `vendor/modules.txt`)         |
| `pkg/version/version.go` | `LatestTag = "<OC_VERSION>+dev"`                                     |

## Procedure

### 1. Verify the reva tag exists

```bash
gh api repos/opencloud-eu/reva/commits/$REVA_VERSION --jq '.sha'
```

- Success (a sha) → the release is tagged, continue.
- `422`/`404` → the tag does not exist yet. The reva release PR (`🎉 Release X.Y.Z`) is probably not merged. **Stop** and tell the user to merge it first. Helpful checks:
  ```bash
  gh api repos/opencloud-eu/reva/releases/latest --jq '.tag_name'   # current latest tag
  gh pr list --repo opencloud-eu/reva --search '🎉 Release in:title' --state open --json number,title
  ```

### 2. Bump the dependency and re-vendor

```bash
go get github.com/opencloud-eu/reva/v2@$REVA_VERSION
go mod tidy
go mod vendor
```

(Sandbox disabled — these need network.) Notes:
- Right after a fresh tag the module proxy can lag; if `go get` reports the version as unknown, retry, or use `GOPROXY=direct go get github.com/opencloud-eu/reva/v2@$REVA_VERSION`.
- `go mod tidy` will also bump indirect dependencies that reva pulled in — that is expected (the template PR did the same).

### 3. Bump the OpenCloud dev version

Edit `pkg/version/version.go`:

```go
LatestTag = "<OC_VERSION>+dev"   // e.g. "7.4.0+dev"
```

### 4. Fetch the reva changelog for the PR body

```bash
gh api "repos/opencloud-eu/reva/contents/CHANGELOG.md?ref=$REVA_VERSION" --jq '.content' | base64 -d
```

Take **only the section for this version** and trim it exactly like the web bump: start at the first content heading (`### 🐛 Bug Fixes` / `### 📈 Enhancement` / `### 💥 Breaking changes`), drop the `# Changelog` title, the `## [x.y.z] - date` header and the `### ❤️ Thanks to all contributors!` block, and stop before the next `## [...]` version header.

Then **prepend two summary bullets** so the final PR body is:

```
- bump opencloud version to <OC_VERSION>
- reva bump <REVA_VERSION without the leading v>

### 🐛 Bug Fixes
...trimmed reva changelog...
```

Write this to a file for `--body-file` (e.g. in the scratchpad).

### 5. Confirm before committing

The `vendor/` diff is huge — do **not** dump it. Show the user:

```bash
git diff go.mod pkg/version/version.go        # the meaningful edits
git diff --stat | tail -1                      # vendor churn summary
```

Confirm the reva line in `go.mod` is exactly `github.com/opencloud-eu/reva/v2 REVA_VERSION`, show the target branch (`main`), and the PR body. **Do not commit until the user approves.**

### 6. Commit, push, open PR

- Create a branch (do not commit on `main`), e.g. `reva-bump-2.48.0`.
- Stage everything the bump touched: `git add go.mod go.sum vendor pkg/version/version.go`.
- One commit, conventional-commits format, **empty body**:
  ```
  chore: reva bump -2.48.0
  ```
- PR base: **`main`** (reva bumps always target main).
- PR title: `[full-ci] chore: reva bump -2.48.0` (the commit message prefixed with `[full-ci] `).
- PR body: the file from step 4.
- Add the label `Type:Maintenance`.

```bash
gh pr create --base main \
  --title "[full-ci] chore: reva bump -$REVA_VERSION_NO_V" \
  --label "Type:Maintenance" \
  --body-file <body-file>
```

(`gh` commands need the sandbox disabled — they require the system keyring.)

## Quick reference

```bash
REVA_VERSION=v2.48.0
OC_VERSION=7.4.0
gh api repos/opencloud-eu/reva/commits/$REVA_VERSION --jq '.sha'                       # verify tag exists
go get github.com/opencloud-eu/reva/v2@$REVA_VERSION && go mod tidy && go mod vendor    # bump + re-vendor
# edit pkg/version/version.go -> LatestTag = "$OC_VERSION+dev"
gh api "repos/opencloud-eu/reva/contents/CHANGELOG.md?ref=$REVA_VERSION" --jq '.content' | base64 -d  # changelog
```

## Common mistakes

- Running the bump before the reva release PR is merged — the tag won't exist and `go get` will fail. Verify the tag first (step 1).
- Forgetting to re-run `go mod vendor` after `go mod tidy`, leaving `vendor/` out of sync with `go.mod`.
- Guessing `OC_VERSION` — it is a deliberate release target (the template PR jumped 7.1.0 → 7.3.0). Ask the user.
- Reading the changelog from reva `main` instead of the tag (`?ref=$REVA_VERSION`). Always pin to the tag.
- Missing the `[full-ci] ` prefix in the PR title, the `Type:Maintenance` label, or the two summary bullets at the top of the body.
- Dumping the full `vendor/` diff at the confirmation step instead of `go.mod` + `version.go` + a `--stat` summary.
- Committing before the user confirms the diff.
