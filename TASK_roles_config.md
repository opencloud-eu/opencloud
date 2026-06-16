# TASK: EditorLitePlus Rolle + Rollen-Strategie

## Status

- **EditorLitePlus**: PR [opencloud-eu/opencloud#2841](https://github.com/opencloud-eu/opencloud/pull/2841) — offen, unabhängig von Upstream
- **Container-Permissions + Immutable**: Fertig im Branch, wartet auf reva#674 Merge

## EditorLitePlus — sofort verfügbar

Neue Sharing-Rolle: Bearbeiten ohne Löschen. Nutzt nur bestehende CS3 Permissions.

| Rolle | Download | Upload | Edit | Create | Delete | Move |
|-------|----------|--------|------|--------|--------|------|
| Viewer | x | - | - | - | - | - |
| EditorLite | x | x | - | x | - | x |
| **EditorLitePlus** | **x** | **x** | **x** | **x** | **-** | **x** |
| Editor | x | x | x | x | x | x |

Explizit ausgeschlossen: `Delete`, `PurgeRecycle`, `ListRecycle`, `RestoreRecycleItem`.

## Rollen nach Container-Permissions (nach reva#674 + Feature-PR)

| Rolle | Delete (Files) | DeleteContainer | MoveContainer | SetImmutableFile | SetImmutableContainer |
|-------|:-:|:-:|:-:|:-:|:-:|
| Viewer | - | - | - | - | - |
| EditorLite | - | - | - | - | - |
| EditorLitePlus | - | - | - | - | - |
| Editor | x | **x** | **x** | - | - |
| Manager | x | **x** | **x** | **x** | **x** |

## Nächste Schritte

1. EditorLitePlus (#2841): Review abwarten — unabhängig
2. Container-Permissions: nach reva Feature-PR Merge → opencloud vendor bumpen
