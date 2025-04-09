---
title: 'Right sidebar panel extensions'
sidebar_position: 6
id: right-sidebar-panel-extensions
---



## Extension Type SideBarPanel

The right sidebar is supposed to show information and make context specific actions available for single or multiple selected items.

It is structured in a hierarchical way:

- Panels which are defined as `root` panels get rendered as immediate members of the right sidebar.
- Panels which are defined as non-`root` panels receive a navigation item below the `root` panels so that users can navigate into the respective
  sub panel.

### Configuration

To define a right sidebar panel extension, you implement the `SidebarPanelExtension` interface.
It can be found below:

```typescript
interface SidebarPanelExtension<R extends Item, P extends Item, T extends Item> {
  id: string
  type: 'sidebarPanel'
  extensionPointIds?: string[]
  panel: SideBarPanel<R, P, T> // Please check the SideBarPanel section below
}
```

For `id`, `type`, and `extensionPointIds`, please see [extension base section](./../index.md) in the top level docs.

The `panel` object configures the actual sidebar panel. It consists of different properties and functions, where all the functions get called with a
`SideBarPanelContext` entity from the integrating extension points.

#### SideBarPanelContext

```typescript
interface SideBarPanelContext<R extends Item, P extends Item, T extends Item> {
  root?: R
  parent?: P
  items?: T[]
}
```

- `items` - The most important member of the panel context, which denotes all selected items. That can mean all selected files in a files listing,
  all selected users in a user listing, the individual current file in a file editor.
- `parent` - The immediate parent of the selected items. For example, if the user selects a file in a file listing, the parent is the parent folder,
  or if being in a root of a space, the space itself. Can be `null` for non-hierarchical contexts, e.g. a user listing.
- `root` - The uppermost parent of the selected items. For example, if the user selects a file in a file listing, the root is always the space in which
  the selected files reside. Can be `null` for non-hierarchical contexts, e.g. a user listing.

#### SideBarPanel

```typescript
interface SideBarPanel<R extends Item, P extends Item, T extends Item> {
  name: string
  icon: string
  iconFillType?: IconFillType
  title(context: SideBarPanelContext<R, P, T>): string
  isVisible(context: SideBarPanelContext<R, P, T>): boolean
  component: Component
  componentAttrs?(context: SideBarPanelContext<R, P, T>): any
  isRoot?(context: SideBarPanelContext<R, P, T>): boolean
}
```

- `name` - A human readable id for the panel.
- `icon`, `iconFillType` and `title` - Properties which are used to render the panel itself or right sidebar navigation items for navigating into that panel.
- `isVisible` - Determines if the panel is available for the given panel context.
- `component` - Provides a component that renders the actual sidebar panel.
- `componentAttrs` - Defines additional props for the component with the given panel context.
- `isRoot` - Determines if the panel is a root panel for the given panel context.

## Extension Point FileSideBar

In the context of files (e.g. file listing, text editor for a single file, etc.) we have a dedicated component `FileSideBar` which can be
toggled (shown/hidden) with a button in the top bar. The component queries all extensions of the type `sideBarPanel` from the extension
registry that also fulfill the scope `resource`. By registering an custom extension of type `sideBarPanel` and scope `resource`, your extension
will automatically become available in all environments that display the `FileSideBar` (i.e. any file viewer, file editor, file listing).

## Example

The following example shows how a sidebar panel for displaying exif data for a resource could look like. Note that the extension is wrapped inside a Vue composable so it can easily be reused. All helper types and composables are being provided via the [web-pkg](https://github.com/opencloud-eu/web/tree/main/packages/web-pkg) and the [web-client](https://github.com/opencloud-eu/web/tree/main/packages/web-client) packages.

```typescript
export const useExifDataPanelExtension = () => {
  const { $gettext } = useGettext()

  const extension = computed<SidebarPanelExtension<SpaceResource, Resource, Resource>>(() => ({
    id: 'com.github.opencloud-eu.web.files.sidebar-panel.exif-data',
    type: 'sidebarPanel',
    scopes: ['resource'],
    panel: {
      name: 'exif-data',
      icon: 'image',
      title: () => $gettext('EXIF data'),
      component: ExifDataPanelComponent,
      isRoot: () => true,
      isVisible: ({ items }) => {
        if (items?.length !== 1) {
          return false
        }

        return true
      }
    }
  }))

  return { extension }
}
```

The extension can then be registered in any app like so:

```typescript
export default defineWebApplication({
  setup() {
    const { extension } = useExifDataPanelExtension()

    return {
      appInfo: {
        name: $gettext('Exif panel app'),
        id: 'exif-panel-app'
      },
      extensions: computed(() => [unref(extension)])
    }
  }
})
```
