# Architektur-Analyse: CS3, Reva, OpenCloud, ownCloud

## Die Abhängigkeitskette

```
cs3org/cs3apis (CERN)          ← Proto-Spec, 24 Permission-Felder + Immutable
    ↓ (protoc generiert)
cs3org/go-cs3apis (CERN)       ← Go-Bindings (regeneriert nach jedem Merge)
    ↓ (go.mod import)
opencloud-eu/reva              ← Fork von cs3org/reva, HAT decomposedfs
    ↓ (vendor)
opencloud-eu/opencloud         ← Backend (Go)
    ↓ (API)
opencloud-eu/web               ← Frontend (Vue.js/TypeScript)
```

## ownCloud vs. OpenCloud — harter Fork seit Januar 2025

Getrennte Teams, keine Code-Synchronisation. Unsere Arbeit geht ausschließlich in OpenCloud.

## Upstream-PRs

| PR | Repo | Status |
|----|------|--------|
| [#272](https://github.com/cs3org/cs3apis/pull/272) | cs3org/cs3apis | **MERGED** (4. Jun 2026) |
| [#5628](https://github.com/cs3org/reva/pull/5628) | cs3org/reva | Open (ACL-Encoding) |
| [#674](https://github.com/opencloud-eu/reva/pull/674) | opencloud-eu/reva | **Eingereicht** (go-cs3apis Update) |
| [#2841](https://github.com/opencloud-eu/opencloud/pull/2841) | opencloud-eu/opencloud | Open (EditorLitePlus) |

## Implementierungs-Branches

| Repo | Branch | Zweck | Wartet auf |
|------|--------|-------|------------|
| flash7777/reva | `chore/update-go-cs3apis` | PR #674: Housekeeping | Review |
| flash7777/reva | `feature/immutable-decomposedfs` | Feature: Immutable + Container-Perms | #674 Merge |
| flash7777/opencloud | `feature/immutable-and-container-permissions` | Graph API Actions | Reva Feature-PR |
| flash7777/opencloud | `feature/editor-light-role` | PR #2841: EditorLitePlus | Review |
| flash7777/web | `feature/immutable-ui` | Frontend Icons + Resource-Modell | Unabhängig |

## Release-Strategie

```
cs3org/cs3apis#272 MERGED ✓
    ↓
cs3org/go-cs3apis regeneriert ✓ (c3fdb0aa5e9e)
    ↓
PR 1: opencloud-eu/reva#674 — go-cs3apis Update + Stubs (eingereicht)
    ↓
PR 2: opencloud-eu/reva — Immutable + Container-Permissions (nach #674)
    ↓
PR 3: opencloud-eu/opencloud — Graph API Actions (nach Reva Release)
    ↓
PR 4: opencloud-eu/web — Frontend (unabhängig)
```

## GitHub Fork-Situation

- `flash7777/reva` ist Fork von cs3org/reva (selbes Fork-Netzwerk wie opencloud-eu/reva → PRs möglich)
- `flash7777/reva-eu` ist eigenständiges Repo (keine PRs gegen opencloud-eu möglich)
- Token mit workflow Scope: `/data/source/gitapps/TOKEN_WF`
