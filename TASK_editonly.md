# TASK: Granulare Container-Permissions (delete_container / move_container)

> Upstream: [cs3org/cs3apis#272](https://github.com/cs3org/cs3apis/pull/272) — **MERGED**
> Reva Update: [opencloud-eu/reva#674](https://github.com/opencloud-eu/reva/pull/674) — eingereicht
> Feature-Branch: `flash7777/reva` branch `feature/immutable-decomposedfs`

## Status

- CS3 Proto: **MERGED** — Felder 21 (delete_container) + 22 (move_container) sind Standard
- go-cs3apis: **Regeneriert** — Felder verfügbar
- Reva go-cs3apis Update: **PR #674 eingereicht** — Housekeeping, wartet auf Review
- Feature-Implementierung: **Fertig im Branch** — wartet auf #674 Merge, dann rebase + PR

## Was implementiert ist

### decomposedfs Handler-Checks
- Delete: `DeleteContainer` Check für Verzeichnisse
- Move: `MoveContainer` Check für Verzeichnisse

### Rollen
- Editor/SpaceEditor/SpaceEditorWithoutVersions: `DeleteContainer: true`, `MoveContainer: true`
- Coowner/Manager: `DeleteContainer: true`, `MoveContainer: true`
- Viewer/EditorLite/EditorLitePlus: kein DeleteContainer/MoveContainer

### ACL-Encoding
- `+dc`/`!dc` (DeleteContainer), `+mc`/`!mc` (MoveContainer)
- Substring-Collision-Fix: `!dc` enthält `!d`
- 8 Tests für Encoding/Decoding

### WebDAV
- `oc:permissions`: `D`/`NV` entfernt wenn immutable

### Graph API (opencloud)
- Neue Actions: `driveItem/container/delete`, `driveItem/container/move`
- Bidirektionales Mapping in conversion.go

## Nächste Schritte

1. PR #674 mergen lassen
2. Feature-Branch rebasen
3. Feature-PR bei opencloud-eu/reva einreichen
