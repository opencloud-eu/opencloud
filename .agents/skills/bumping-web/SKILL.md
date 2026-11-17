---
name: bumping-web
description: Use when the user asks to bump, update, or upgrade the OpenCloud web assets/frontend to a specific version (e.g. "bump web to v7.0.0", "update web to v7.1.2"). Covers editing the version files, the single commit, and opening the PR against the right branch.
---

# Bumping Web

## Overview

Bumping web updates the pinned OpenCloud web frontend (assets + UI-test runner) to a tagged release of [opencloud-eu/web](https://github.com/opencloud-eu/web). It touches exactly two files, lands as one commit, and ships as a PR whose body is the web changelog for that version.

Template PR: https://github.com/opencloud-eu/opencloud/pull/2733

## Prerequisites

- **`gh` (GitHub CLI), authenticated** — every lookup and the PR creation go through `gh api` / `gh pr`. Verify with `gh auth status`; if it fails, ask the user to run `gh auth login` (suggest `! gh auth login` so it runs in-session). Do not proceed without it.
- `gh` needs the system keyring, so run all `gh` commands with the sandbox disabled.
- `git` and `base64` (decoding the changelog) — standard on macOS/Linux.

## What changes (exactly two files)

| File                    | Variable             | New value                                        |
| ----------------------- | -------------------- | ------------------------------------------------ |
| `services/web/Makefile` | `WEB_ASSETS_VERSION` | the version tag, e.g. `v7.0.0`                   |
| `services/web/Makefile` | `WEB_ASSETS_BRANCH`  | branch carrying the tag (`main` or `stable-X.Y`) |
| `.woodpecker.env`       | `WEB_COMMITID`       | full commit sha the tag points to                |
| `.woodpecker.env`       | `WEB_BRANCH`         | same branch as `WEB_ASSETS_BRANCH`               |

`WEB_ASSETS_BRANCH` and `WEB_BRANCH` are always the same value.

## Procedure

Let `VERSION` be the requested tag (always normalize to a leading `v`, e.g. `v7.0.0`).

### 1. Resolve the commit sha the tag points to

```bash
gh api repos/opencloud-eu/web/commits/$VERSION --jq '.sha'
```

This full sha is the new `WEB_COMMITID`.

### 2. Determine the branch (`main` vs `stable-X.Y`)

The branch is the line of development that carries the tag. Detect it:

```bash
gh api "repos/opencloud-eu/web/compare/main...$VERSION" --jq '.status'
```

- `identical` or `behind` → the tagged commit is reachable from `main` → use **`main`**.
- `ahead` or `diverged` → the tag lives on a release line → use **`stable-X.Y`** matching the version's major.minor (e.g. `v7.1.2` → `stable-7.1`).

Sanity-check that the stable branch actually exists:

```bash
gh api repos/opencloud-eu/web/branches --paginate --jq '.[].name' | grep -E 'main|stable'
```

Typically a freshly released minor (`v7.1.0`) still sits on `main`, while later patches on an older line (`v7.0.3` after `7.1` exists) sit on `stable-7.0`.

### 3. Fetch the changelog for the PR body

```bash
gh api "repos/opencloud-eu/web/contents/CHANGELOG.md?ref=$VERSION" --jq '.content' | base64 -d
```

Take **only the section for this version**. Match the template: start the PR body at the first content heading (e.g. `### 💥 Breaking changes` / `### 📈 Enhancement`) and drop the `# Changelog` title, the `## [x.y.z] - date` header, and the `### ❤️ Thanks to all contributors!` block. Stop before the next `## [...]` version header.

### 4. Apply the edits

Edit `services/web/Makefile` (`WEB_ASSETS_VERSION`, `WEB_ASSETS_BRANCH`) and `.woodpecker.env` (`WEB_COMMITID`, `WEB_BRANCH`).

### 5. Pick the target branch for the PR

- Major or minor release → target **`main`**.
- Patch release → may need to target a stable branch instead. If the user did not specify, **ask them** which branch to target before continuing.

### 6. Confirm before committing

Show the user the diff of both files and the target branch, and ask them to confirm. Do not commit until they approve.

### 7. Commit, push, open PR

- Create a branch (do not commit on `main`).
- One commit, conventional-commits format, with no body other than the `Assisted-by:` trailer that the org's [AI guideline](https://github.com/opencloud-eu/.github/blob/main/profile/AI_GUIDELINE.md) asks for. Name the agent and the model actually used, and never add a `Co-authored-by` trailer for an AI tool:
  ```
  chore: bump web to v7.0.0

  Assisted-by: Claude Code:claude-opus-5-5
  ```
- PR title: `[full-ci] chore: bump web to v7.0.0` (the commit message prefixed with `[full-ci] `).
- PR base: the branch chosen in step 5.
- PR body: the trimmed changelog from step 3.
- End the PR body with one line saying which AI did the bump, e.g. `AI assistance: Claude Code (claude-opus-5-5) did the bump.`
- Add the labels `Type:Dependencies` and `Status:Needs-Review`, and assign the PR to yourself.

```bash
gh pr create --base <target-branch> \
  --title "[full-ci] chore: bump web to $VERSION" \
  --label "Type:Dependencies,Status:Needs-Review" \
  --assignee @me \
  --body-file <changelog-file>
```

(`gh` commands need the sandbox disabled — they require the system keyring.)

## Quick reference

```bash
VERSION=v7.0.0
gh api repos/opencloud-eu/web/commits/$VERSION --jq '.sha'                       # WEB_COMMITID
gh api "repos/opencloud-eu/web/compare/main...$VERSION" --jq '.status'           # main vs stable
gh api "repos/opencloud-eu/web/contents/CHANGELOG.md?ref=$VERSION" --jq '.content' | base64 -d  # changelog
```

## Common mistakes

- Reading `WEB_COMMITID` or the changelog from web `main` instead of from the version tag (`?ref=$VERSION`). Always pin to the tag.
- Leaving `WEB_ASSETS_BRANCH`/`WEB_BRANCH` on `main` for a patch that belongs on a stable line.
- Forgetting `[full-ci] ` in the PR title, or adding a commit body beyond the `Assisted-by:` trailer.
- Committing before the user confirms the diff and target branch.
