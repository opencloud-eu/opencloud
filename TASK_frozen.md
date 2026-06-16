# TASK: Immutable-Attribut (freeze / protect)

> Upstream: [cs3org/cs3apis#272](https://github.com/cs3org/cs3apis/pull/272) — **MERGED**
> Konzept: [immutable-overview.md](/data/source/gitapps/immutable-overview.md)
> Feature-Branch: `flash7777/reva` branch `feature/immutable-decomposedfs`

## Status

- CS3 Proto: **MERGED** — Feld 20 (immutable), Felder 23+24 (set_immutable_file/container), RPCs
- go-cs3apis: **Regeneriert** — alle Felder + RPCs verfügbar
- Reva go-cs3apis Update: **PR #674 eingereicht** — Housekeeping, wartet auf Review
- Feature-Implementierung: **Fertig im Branch** — wartet auf #674 Merge
- GRPC-Handler: SetImmutable/UnsetImmutable in storageprovider + gateway implementiert

## Konzept

Ein Attribut pro Objekt (`user.oc.immutable`). Wirkung hängt vom Typ ab:

### Datei — freeze
- Inhalt fixiert, kein Overwrite
- Kein Löschen, Umbenennen, Verschieben
- **Irreversibel**

### Verzeichnis — protect
- Keine Einträge hinzufügen, entfernen oder modifizieren
- Kein Löschen, Umbenennen, Verschieben des Verzeichnisses
- **Reversibel** durch Manager/Admin
- Propagiert **NICHT** zu Kindern

### Self vs. Parent → Effektiver State
| State | Bedeutung | Icon |
|-------|-----------|------|
| **Frozen** | Self-immutable | Shield filled |
| **Protected** | Parent-immutable | Shield outline |
| **None** | Weder self noch parent | — |

### Lock vs. Protected vs. Frozen
| | Lock | Protected | Frozen |
|---|---|---|---|
| Zweck | Kollaboratives Editing | Strukturschutz | Inhaltsschutz |
| Dauer | Temporär | Bis Manager aufhebt | **Permanent** (Files) |
| Reversibel | Ja | Ja | **Nein** (Files) |

## Was implementiert ist

### Reva decomposedfs
- xattr: `user.oc.immutable`
- `ImmutableState` Enum: None (0), Protected (1), Frozen (2)
- Node: `IsImmutable()`, `GetImmutableState()`, `FreezeFile()`, `ProtectContainer()`, `UnprotectContainer()`
- Storage-Interface: `SetImmutable()`, `UnsetImmutable()` mit Permission-Checks
- Handler-Checks: Delete, Move, CreateDir, Upload — alle abgesichert
- Stat(): `ResourceInfo.Immutable` + Opaque `immutable-state`
- GRPC: SetImmutable/UnsetImmutable in storageprovider + gateway
- `OwnerPermissions()` + `AddPermissions()`: neue Felder
- ACL: `+if`/`+ic` Encoding für SetImmutableFile/SetImmutableContainer
- Legacy-Treiber: Stubs

### WebDAV (ocdav)
- `oc:immutable`: neues Property (frozen/protected/absent)
- `oc:permissions`: D/NV entfernt wenn effektiv immutable
- Allprops + Named Property unterstützt

### Graph API (opencloud)
- Neue Actions: `driveItem/immutable/file/set`, `driveItem/immutable/container/set`
- Mapping in conversion.go

### Web Frontend
- `Resource.immutableState`: `'frozen' | 'protected' | undefined`
- `canBeDeleted()` / `canRename()`: return false wenn immutable
- FileDetails Sidebar: Shield-Icon (filled=frozen, outline=protected)

### Permissions
- `set_immutable_file` (Feld 23): Dateien einfrieren (irreversibel)
- `set_immutable_container` (Feld 24): Verzeichnisse protecten (reversibel)
- Nur Coowner + Manager haben diese Permissions

### Tests
- 6 Node-Tests (IsImmutable, GetImmutableState, Freeze, Protect, Unprotect, Parent-Regel)
- 7 Handler-Tests (Delete/Move/CreateDir/Upload frozen/protected, SetImmutable Permissions)
- 8 Grants-ACL-Tests (Encoding/Decoding + Substring-Collision)
- 2 pre-existierende Failures (UpdateGrant/DenyGrant ACL-Round-Trip — vorbestehender Bug)

## Nächste Schritte

1. PR #674 mergen lassen (go-cs3apis Update)
2. Feature-Branch auf #674 rebasen
3. Feature-PR bei opencloud-eu/reva einreichen
4. Graph API PR bei opencloud-eu/opencloud einreichen
5. Web Frontend PR bei opencloud-eu/web einreichen
