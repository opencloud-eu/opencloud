---
title: 'Left sidebar menu item extensions'
sidebar_position: 5
id: left-sidebar-menu-item-extensions
---



## Left sidebar menu item extension type

One possible extension type is left sidebar menu items. Registered left sidebar menu items get rendered in the left sidebar, as long as there is more than one available.

### Configuration

To define a left sidebar menu item, you implement the SidebarNavExtension interface.
It looks like this:

```typescript
interface SidebarNavExtension {
    id: string
    type: 'sidebarNav'
    extensionPointIds?: string[]
    navItem: AppNavigationItem // Please check the AppNavigationItem section below
    }
}
```

For `id`, `type`, and `extensionPointIds`, please see [extension base section](./../index.md) in the top level docs.

#### AppNavigationItem

The most important configuration options are:

- `icon` - The icon to be displayed, can be picked from https://opencloud.design/#/Design%20Tokens/IconList
- `name` - The text to be displayed
- `route` - The string/route to navigate to, if the nav item should be a `<router-link>` (Mutually exclusive with `handler`)
- `handler` - The action to perform upon click, if the nav item should be a `<button>` (Mutually exclusive with `route`)

Please check the [`AppNavigationItem` type](https://github.com/opencloud-eu/web/blob/f069ce44919cde5d112c68a519d433e015a4a011/packages/web-pkg/src/apps/types.ts#L14) for a full list of configuration options.

### Example

The following example shows an extension that adds a left sidebar nav item inside the files app, linking to a custom page. Note that the extension is wrapped inside a Vue composable so it can easily be reused. All helper types and composables are being provided via the [web-pkg](https://github.com/opencloud-eu/web/tree/main/packages/web-pkg) package.

```typescript
export const useCustomPageExtension = () => {
  const { $gettext } = useGettext()

  const extension = computed<SidebarNavExtension>(() => ({
    id: 'com.github.opencloud-eu.web.files.left-nav.custom-page',
    scopes: ['app.files'],
    type: 'sidebarNav',
    action: {
      name: $gettext('Custom page'),
      icon: 'world',
      priority: 100,
      isActive: () => true,
      isVisible: () => true,
      route: {
        path: '/files/custom-page'
      },
      activeFor: [{ path: '/files/custom-page' }]
    }
  }))

  return { extension }
}
```

The extension could then be registered in any app like so:

```typescript
export default defineWebApplication({
  setup() {
    const { extension } = useCustomPageExtension()

    return {
      appInfo: {
        name: $gettext('Custom page app'),
        id: 'custom-page-app'
      },
      routes: {
        path: '/files/custom-page',
        name: 'files-custom-page',
        component: CustomPageComponent,
        meta: {
          title: $gettext('Custom Page')
        }
      },
      extensions: computed(() => [unref(extension)])
    }
  }
})
```
