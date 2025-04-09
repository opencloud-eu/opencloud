---
title: 'Folder view extensions'
sidebar_position: 4
id: folder-view-extensions
---

## Folder view extension type

The folder view is one of the possible extension types. Registered folder view can be used to render multiple resources (folders, files, spaces) in the UI.

### Configuration

This is what the FolderViewExtension interface looks like:

```typescript
interface FolderViewExtension {
  id: string
  type: 'folderView'
  extensionPointIds?: string[]
  folderView: FolderView // See FolderView section below
}
```

For `id`, `type`, and `extensionPointIds`, please see [extension base section](./../index.md) in the top level docs.

#### FolderView

For the folderView object, you have the following configuration options:

- `name` - The name of the action (not displayed in the UI)
- `label` - The text to be displayed to the user when switching between different FolderView options
- `icon` - Object, expecting an icon `name` and a corresponding `IconFillType`, see https://opencloud.design/#/Design%20Tokens/IconList for available options
- `isScrollable` - Optional boolean, determines whether the user can scroll inside the component or it statically fills the viewport
- `component` - The Vue component to render the resources. It should expect a prop of type `Resource[]`
- `componentAttrs` - Optional additional configuration for the component mentioned above

### Example

The following example shows how an extension for a custom folder view could look like. Note that the extension is wrapped inside a Vue composable so it can easily be reused. All helper types and composables are being provided via the [web-pkg](https://github.com/opencloud-eu/web/tree/main/packages/web-pkg) package.

```typescript
export const useCustomFolderViewExtension = () => {
  const { $gettext } = useGettext()

  const extension = computed<FolderViewExtension>(() => ({
    id: 'com.github.opencloud-eu.web.files.folder-view.custom',
    type: 'folderView',
    scopes: ['resource', 'space', 'favorite'],
    folderView: {
      name: 'custom-table',
      label: $gettext('Switch to custom folder view'),
      icon: {
        name: 'menu-line',
        fillType: 'none'
      },
      component: YourCustomFolderViewComponent
    }
  }))

  return { extension }
}
```

The extension could then be registered in any app like so:

```typescript
export default defineWebApplication({
  setup() {
    const { extension } = useCustomFolderViewExtension()

    return {
      appInfo: {
        name: $gettext('Custom folder view app'),
        id: 'custom-folder-view-app'
      },
      extensions: computed(() => [unref(extension)])
    }
  }
})
```
