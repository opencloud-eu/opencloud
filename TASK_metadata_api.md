# Custom Metadata API (oy.* / Aktenplan)

## Ziel

Aktenplan-Metadaten (`user.oc.md.oy.*`) über Graph API und WebDAV abrufbar machen, zur Nutzung im Web UI.

## Ist-Zustand

- xattrs auf Disk: `user.oc.md.oy.subject`, `user.oc.md.oy.created`, etc. ✅
- Reva `node.AsResourceInfo()`: liest `user.oc.md.*` → `ArbitraryMetadata` map ✅
- CS3 gRPC `Stat()`: liefert `ArbitraryMetadata` ✅
- WebDAV PROPFIND: nur registrierte Namespaces (oc:, d:, libre.graph:), custom 404 ❌
- Graph API DriveItem: nur Audio/Photo/Image/Location Facetten, kein generisches Custom-Properties ❌
- Web Frontend: zeigt nur bekannte Facetten ❌

## Offene Fragen

1. Filtert Reva die `oy.*` Keys aus dem ResourceInfo?
   - `node.AsResourceInfo()` liest alle `user.oc.md.*` → sollte drin sein
   - Aber: werden sie bei `ListContainer`/`Stat` durchgereicht oder rausgefiltert?
   - Prüfen: `arbitrary_metadata_keys` Parameter — muss `*` oder `oy.*` enthalten

2. WebDAV PROPFIND Fallback-Handler (propfind.go:1729-1738):
   - Existiert für unbekannte Namespaces
   - Warum 404? Namespace-Registrierung nötig? Oder Request-Format falsch?

3. Eigene Abfrage `resourceMeta` sinnvoll?
   - Statt alles in ResourceInfo zu packen, separate RPC für Metadaten
   - Vorteil: keine Verschmutzung der Standard-Responses
   - Nachteil: extra Round-Trip

## Ansätze

### A) WebDAV erweitern

In Reva propfind.go einen neuen Property-Namespace registrieren:
```
Namespace: "http://openyard.eu/ns"
Prefix: "oy"
```

Dann: `<oy:subject>`, `<oy:created>`, etc. in PROPFIND Responses.

### B) Graph API erweitern

In libre-graph-api-go: `CustomProperties map[string]string` auf DriveItem.
OpenCloud Graph Service: `ArbitraryMetadata` → `CustomProperties` mappen.

### C) Beides (empfohlen)

WebDAV für Desktop-Clients, Graph API für Web UI.

## Beispiel-Metadaten (Aktenplan)

```
oy.subject      = "2015-11-25 Hochdruckreiniger"
oy.created      = "2016-02-17T12:35:00"
oy.creatorId    = "{95760e8e-...}"
oy.creatorName  = "Kögler, Kristin"
oy.fullPath     = "99 Archikart DMS|Facility Management|..."
oy.lastUpdated  = "2016-02-17T12:35:00"
oy.version      = "1"
oy.status       = "1"
oy.configName   = "Migration"
```

## Abhängigkeiten

- Unabhängig von Immutable-Feature
- libre-graph-api-go PR für DriveItem.CustomProperties
- Reva propfind.go für WebDAV Namespace
- Web Frontend Metadaten-Panel
