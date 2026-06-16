# Feature: Treeview Ansichtsmodus

## Ziel

Vierter Ansichtsmodus für die Dateiliste: hierarchischer Baum mit Collapse/Expand, Lazy-Loading der Kinder-Ebenen.

## Motivation

- Aktenplan-Strukturen haben tiefe Hierarchien (5-10 Ebenen)
- Protected/Shielded Status auf einen Blick über mehrere Ebenen sichtbar
- Windows-Explorer Vertrautheit für Ex-Windows-Nutzer
- Schnelle Navigation ohne ständiges Verzeichniswechseln

## Konzept

### Ansicht

```
▼ 📁 99 Archikart DMS                    🛡 protected
  ▼ 📁 Friedhofsverwaltung               🛡 shielded
    ▶ 📁 Mail                             🛡 shielded
    ▶ 📁 Verträge                         🛡 shielded
    ▼ 📁 Grabstellen                      🛡 shielded
      📄 Register.xlsx                    ❄️ frozen
      📄 Plan_2024.pdf                    🍃 normal
  ▶ 📁 Liegenschaften                     🛡 shielded
  ▶ 📁 Ordnungswidrigkeiten              🛡 shielded
▶ 📁 Aufgaben2
▶ 📁 Bibliothek
```

### Verhalten

- **Collapse/Expand**: Klick auf ▶/▼ Toggle
- **Lazy Loading**: Kinder werden erst beim Expand geladen (PROPFIND Depth:1)
- **Expand All / Collapse All**: Toolbar-Buttons
- **Selektion**: Klick auf Name → Detail-Panel, Doppelklick → Navigieren (wie bisher)
- **Context Menu**: Rechtsklick → gleiche Actions wie in anderen Views
- **Quick Actions**: Hover-Buttons (Protect/Freeze) an jeder Zeile
- **Immutable Icons**: Shield/Snowflake/Leaf direkt in der Baumzeile
- **Einrückung**: Pro Ebene ~20px Indent

### Performance

- **On-Demand**: Nur die sichtbare Ebene laden, Kinder erst bei Expand
- **Virtualisierung**: Bei großen Listen (>1000 sichtbare Nodes) Virtual Scrolling
- **Cache**: Einmal geladene Ebenen im Store behalten bis Navigation wechselt
- **Debounce**: Schnelles Expand/Collapse ohne Request-Flood

## Technische Umsetzung

### 1. Neuer View-Modus

In `packages/web-pkg/src/components/FilesList/`:
- `ResourceTree.vue` — Hauptkomponente
- `ResourceTreeNode.vue` — Einzelner Baum-Knoten (rekursiv)
- `useResourceTree.ts` — Composable für Expand-State + Lazy-Loading

### 2. View-Mode Integration

In `packages/web-app-files/`:
- `extensionPoints.ts`: neuer View-Mode `resource-tree`
- View-Selector (Toolbar): viertes Icon (Baum-Symbol)
- URL-Parameter: `view-mode=resource-tree`

### 3. Datenmodell

```typescript
interface TreeNode {
  resource: Resource
  expanded: boolean
  loading: boolean
  children: TreeNode[]  // leer bis geladen
  depth: number
}
```

### 4. API-Nutzung

- **PROPFIND Depth:1** pro expandiertem Ordner (wie ListFolder)
- Gleiche WebDAV-Properties wie in der normalen Liste (inkl. oc:immutable)
- Kein neuer Backend-Endpunkt nötig

### 5. Icons

Remix Icons verfügbar:
- `arrow-right-s-line` → Collapsed (▶)
- `arrow-down-s-line` → Expanded (▼)
- `folder-line` / `folder-open-line` → Ordner
- `folder-shield-line` → Protected Ordner (Bonus)
- `file-line` → Datei

## Abhängigkeiten

- Unabhängig von Immutable-Feature (nutzt es aber gut)
- Unabhängig von Metadata-Panel
- Braucht keine Backend-Änderungen

## Aufwand

- Frontend: ~3-5 Tage
- Keine Backend-Änderungen
- Keine Proto/API-Änderungen

## Priorität

Nach Immutable-Feature stabil. Eigenständiges Feature-Projekt.
