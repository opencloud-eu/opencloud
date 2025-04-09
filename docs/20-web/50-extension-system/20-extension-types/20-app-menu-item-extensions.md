---
title: 'Application menu item extensions'
sidebar_position: 2
id: app-menu-item-extensions
---



## Extension Type AppMenuItem

This extension type allows apps to register links to internal or external pages within the application switcher menu on the top left.

### Configuration

The Interface for an `AppMenuItemExtension` looks like so:

```typescript
interface AppMenuItemExtension {
  id: string
  type: 'appMenuItem'
  extensionPointIds?: string[]
  label: () => string
  color?: string
  handler?: () => void
  icon?: string
  path?: string
  priority?: number
  url?: string
}
```

For `id`, `type`, and `extensionPointIds`, please see [extension base section](./../index.md) in the top level docs.

A `handler` will result in a `<button>` element. This is necessary when an action should be performed when clicking the menu item (e.g. opening a file editor).

A `path` will result in an `<a>` element that links to an internal page via the vue router. That means the given path needs to exist within the application.

A `url` will result in an `<a>` element that links to an external page. External pages always open in a new tab or window.

At least one of these properties has to be provided when registering an extension. If you define more than one, the priority order is `handler` > `path` > `url`.

`priority` specifies the order of the menu items. 50 is a good number to start with, then go up/down based on where the item should be placed. Defaults to the highest possible number, so the item will most likely end up at the bottom of the list if you don't specify a `priority`. Leave it empty if unsure what to pick.

## Example

The following example shows how an app creates an extension that registers an app menu item, linking to an internal page. All helper types and composables are being provided via the [web-pkg](https://github.com/opencloud-eu/web/tree/main/packages/web-pkg) package.

```typescript
export default defineWebApplication({
  setup() {
    const { $gettext } = useGettext()
    const appId = 'my-cool-app'

    const menuItems = computed<AppMenuItemExtension[]>(() => [
      {
        id: 'com.github.opencloud-eu.web.my-cool-app.menu-item',
        type: 'appMenuItem',
        label: () => $gettext('My cool app'),
        path: urlJoin(appId),
        icon: 'star',
        color: '#0D856F',
        priority: 60
      }
    ])

    return {
      appInfo: {
        name: $gettext('My cool app'),
        id: appId
      },
      extensions: menuItems
    }
  }
})
```
