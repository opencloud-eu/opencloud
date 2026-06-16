# Kosmos Image: Deploy-Dokumentation

## Ziel

Custom OpenCloud-Image ("kosmos") basierend auf **v7.1.0 Tag** mit Immutable/Container-Permissions Feature.

## Status: DEPLOYED auf cloud.brandis.eu

Login ✅, Spaces ✅, Immutable-Permissions aktiv ✅

## Architektur

```
OpenCloud v7.1.0 (Tag)
  + "kosmos" Edition (version.go)
  + Labels-API Fix (follow.go: provider → labels Import)
  + Reva v7.1.0-Pin (0e975e5456eb)
      + go-cs3apis Update (cs3apis#272)
      + Immutable Feature (DeleteContainer, MoveContainer, SetImmutable*)
  + Web v7.1.0
  + Warmup-Patch (Issue #547, nur im Dockerfile)
```

Storage-Driver: **posix** (Wrapper um decomposedfs, Default seit 6.x)

## Kritischer Build-Hinweis

**`make release-linux-docker-amd64`** verwenden, NICHT `make build`!

`make build` setzt keine `DOCKER_LDFLAGS` → falsche Pfadauflösung:
- `BaseDataPathType=homedir` statt `path`
- `BaseConfigPathType=homedir` statt `path`
- → Override-Config `/etc/opencloud/` wird ignoriert
- → Spaces, mount_id, Passwörter aus falscher Config

Die `DOCKER_LDFLAGS` setzen:
```
-X "pkg/config/defaults.BaseDataPathType=path"
-X "pkg/config/defaults.BaseDataPathValue=/var/lib/opencloud"
-X "pkg/config/defaults.BaseConfigPathType=path"
-X "pkg/config/defaults.BaseConfigPathValue=/etc/opencloud"
```

## Build

### Voraussetzungen

- Podman
- Branches: `build/kosmos-v7.1.0` in opencloud + reva

### Befehle

```bash
cd /data/source/gitapps/opencloud
git checkout build/kosmos-v7.1.0

cd /data/source/gitapps/reva
git checkout build/kosmos-v7.1.0

cd /data/source/gitapps/opencloud
rm -rf reva-src && cp -a /data/source/gitapps/reva reva-src
podman build -f Dockerfile.test -t opencloud-immutable:test .

podman tag opencloud-immutable:test docker.io/flash7777pods/opencloud-kosmos:latest
podman push docker.io/flash7777pods/opencloud-kosmos:latest
```

### Dockerfile.test (3-Stage Build)

1. **Node Stage**: IDP-Assets generieren + Web v7.1.0 Assets herunterladen
2. **Go Stage**: Reva-Replace + `go mod vendor` + Warmup-Patch (`git apply`) + `make release-linux-docker-amd64`
3. **Runtime Stage**: Alpine 3.23 (identisch zum offiziellen Image)

## Upgrade-Pfad: 5.1.0 → Kosmos (v7.1.0-basiert)

### Voraussetzungen

- `yq` installiert: `curl -sL https://github.com/mikefarah/yq/releases/latest/download/yq_linux_amd64 -o /usr/local/bin/yq && chmod +x /usr/local/bin/yq`
- Backup existiert
- **NIEMALS** `sed` oder `python yaml.dump` auf Config-Dateien! Immer `yq`!

### Schritt 1: sharing service_account hinzufügen

```bash
SA_ID=$(yq ".proxy.service_account.service_account_id" /data/config/opencloud.yaml)
SA_SEC=$(yq ".proxy.service_account.service_account_secret" /data/config/opencloud.yaml)

for cfg in /data/config/opencloud.yaml /data/data/.opencloud/config/opencloud.yaml; do
  yq -i ".sharing.service_account.service_account_id = \"$SA_ID\"" "$cfg"
  yq -i ".sharing.service_account.service_account_secret = \"$SA_SEC\"" "$cfg"
done
```

### Schritt 2: banned-password-list.txt kopieren

```bash
cp /data/config/banned-password-list.txt /data/data/.opencloud/config/banned-password-list.txt
# Falls nicht vorhanden:
touch /data/data/.opencloud/config/banned-password-list.txt
```

### Schritt 3: NATS SSE-Consumer aufräumen (optional)

```bash
podman stop opencloud_full-opencloud-1
rm -rf /data/data/.opencloud/nats/jetstream/'$G'/streams/main-queue/obs/sse-*
# NUR sse-* löschen! Reguläre Consumer behalten!
```

### Schritt 4: Warmup-ENVs in compose

In `opencloud.yml` unter `environment:`:

```yaml
STORAGE_USERS_POSIX_WARMUP_IGNORE_PERM_ERRORS: "true"
STORAGE_USERS_POSIX_WARMUP_RESPECT_OCIGNORE: "true"
STORAGE_USERS_POSIX_SCAN_FS: "true"
```

### Schritt 5: Deploy

```bash
podman pull docker.io/flash7777pods/opencloud-kosmos:latest
podman tag docker.io/flash7777pods/opencloud-kosmos:latest opencloud-patched:latest

# .env
OC_DOCKER_IMAGE=opencloud-patched
OC_DOCKER_TAG=

podman compose up -d opencloud
```

### Rollback

```bash
OC_DOCKER_IMAGE=opencloudeu/opencloud-rolling
OC_DOCKER_TAG=5.1.0
podman compose up -d opencloud
```

## Bekannte Probleme

### Dual-Config (Override vs Data)

OpenCloud liest aus zwei YAML-Dateien:
- `/etc/opencloud/opencloud.yaml` (Override, von `/data/config/`)
- `/var/lib/opencloud/.opencloud/config/opencloud.yaml` (Data, von `/data/data/`)

v7.1.0 bevorzugt konsistent die Override-Config. Wenn beide verschiedene Secrets haben (durch nachträgliches `opencloud init`), müssen sie synchronisiert werden.

**Config-Dateien NUR mit `yq` editieren** — `sed` und `python yaml.dump` zerstören Sonderzeichen in Passwörtern!

### btrfs read-only Snapshots (Issue #547)

Warmup-Patch im Dockerfile umgeht das Problem: Walk-Errors werden geskippt, .ocignore respektiert, setDirty non-fatal. Betrifft auch offizielles 7.1.0.

### main HEAD Breaking Changes (NICHT verwenden!)

| Problem | Auswirkung |
|---------|------------|
| Config-Merge-Logik geändert | Services lesen Secrets aus verschiedenen Configs |
| IDM LDAPS→LDAP | Port 9235→9236, alte BoltDB-Passwörter inkompatibel |
| warmupSpaceRootCache fatal | Service startet nicht bei NATS-Problemen |

## Neue Permissions (CS3 API)

Aktiv auf Reva-Ebene, UI-Integration folgt:

| Permission | Rollen | Zweck |
|-----------|--------|-------|
| DeleteContainer | Editor, SpaceEditor, Manager, Coowner | Ordner löschen (getrennt von Datei-Delete) |
| MoveContainer | Editor, SpaceEditor, Manager, Coowner | Ordner verschieben |
| SetImmutableFile | Manager, Coowner | Dateien einfrieren (irreversibel) |
| SetImmutableContainer | Manager, Coowner | Ordner schützen (reversibel) |

## Immutable State UI

Datenfluss: Reva xattr → `GetImmutableState()` → `oc:immutable` WebDAV Property → `resource.immutableState`

### Quick Actions (Hover-Buttons in der Dateiliste)

| Typ | State | Icon | Aktion |
|-----|-------|------|--------|
| Datei | normal | leaf (Blatt) | Klick → Freeze (mit Bestätigungsdialog, irreversibel!) |
| Datei | frozen | snowflake (Schneeflocke) | deaktiviert — permanent eingefroren |
| Datei | protected | shield-fill (volles Schild) | deaktiviert — Parent-Ordner geschützt |
| Ordner | normal | shield-line (leeres Schild) | Klick → Protect |
| Ordner | protected | shield-fill (volles Schild) | Klick → Unprotect |

### Indicators (Badges neben dem Dateinamen)

| State | Icon | Bedeutung |
|-------|------|-----------|
| frozen | snowflake | Datei ist permanent eingefroren |
| protected | shield-fill | Ordner/Datei ist geschützt |

### Reva GetImmutableState Logik

| Situation | State |
|-----------|-------|
| Datei mit `user.oc.immutable=1` | `frozen` |
| Ordner mit `user.oc.immutable=1` | `protected` |
| Kind eines immutable Ordners | `protected` |
| Normal | nicht gesetzt |

ServiceAccount und Owner haben automatisch alle neuen Permissions.

## Branches

| Repo | Branch | Basis | Inhalt |
|------|--------|-------|--------|
| opencloud | `build/kosmos-v7.1.0` | Tag v7.1.0 | + kosmos Edition + Labels-Fix |
| reva | `build/kosmos-v7.1.0` | `0e975e5456eb` | + go-cs3apis + Immutable |
| reva | `feature/immutable-decomposedfs` | main HEAD | PR #676 |
| reva | `fix/warmup-non-fatal` | main HEAD | PR #678 |

## PRs

- **opencloud-eu/reva#676**: Immutable + Container-Perms (wartet auf Review, Tests + follow.go Hinweis ergänzt)
- **opencloud-eu/reva#678**: warmupSpaceRootCache non-fatal (separater Fix für main)
- **cs3org/cs3apis#272**: MERGED

## Implementierungsstand

### Fertig & getestet

| Schicht | Feature | Status |
|---------|---------|--------|
| CS3 API | ResourcePermissions + Immutable Felder | ✅ cs3apis#272 MERGED |
| CS3 API | Gateway SetImmutable/UnsetImmutable | ⏳ cs3apis#275 (wartet auf Review) |
| Reva | Immutable xattr + Permission Checks | ✅ reva#676 (wartet auf Review) |
| Reva | warmupSpaceRootCache non-fatal | ✅ reva#678 |
| OpenCloud | Permission Actions (conversion.go) | ✅ kompiliert, Tests PASS |
| OpenCloud | Labels-Fix (follow.go) | ✅ kompiliert |
| OpenCloud | Metadata-Endpunkt (GET /metadata) | ✅ deployed, getestet mit oy.* Daten |
| OpenCloud | Freeze/Protect/Unprotect Endpunkte | ⏳ wartet auf cs3apis#275 |
| Web | MetadataPanel Sidebar-Tab | ✅ TypeScript kompiliert |
| Web | Immutable State via WebDAV PROPFIND | ✅ `oc:immutable` → `resource.immutableState` |
| Web | Quick Action Icons (leaf/snowflake/shield) | ✅ TypeScript kompiliert |
| Web | Indicator Badges (frozen/protected) | ✅ TypeScript kompiliert |

### Branches

| Repo | Branch | Tag | Inhalt |
|------|--------|-----|--------|
| opencloud | `build/kosmos-v7.1.0` | `working_immutable` | Deploy: Metadata + Permissions + Labels + Warmup |
| opencloud | `feature/immutable-graph-api` | - | PR-Vorbereitung: Graph API Actions + Endpunkte |
| reva | `build/kosmos-v7.1.0` | `working_immutable` | Deploy: go-cs3apis + Immutable Feature |
| reva | `feature/immutable-decomposedfs` | - | PR #676 |
| reva | `fix/warmup-non-fatal` | - | PR #678 |
| web | `feature/metadata-panel` | `working_metadata` | MetadataPanel + Immutable UI |
| cs3apis | `feat/gateway-immutable-rpc` | - | PR #275 |

### PRs

| PR | Repo | Status |
|----|------|--------|
| cs3apis#272 | cs3org/cs3apis | ✅ MERGED |
| cs3apis#275 | cs3org/cs3apis | ⏳ wartet auf Review |
| reva#676 | opencloud-eu/reva | ⏳ wartet auf Review |
| reva#678 | opencloud-eu/reva | ⏳ wartet auf Review |

### Nächste Schritte

1. cs3apis#275 + reva#676 Reviews abwarten
2. OpenCloud PR einreichen (nach Merges): go-cs3apis Bump + Graph API + freeze/protect Endpunkte
3. Web PR einreichen: MetadataPanel + Immutable UI
4. Web neu bauen und ins kosmos Image einbetten
5. Bleve-Fix als PR bei opencloud-eu/opencloud
