---
title: 'Extension system'
---



## Concepts and Building Blocks

OpenCloud Web can be extended through various entry points with custom **apps** and **extensions**.

### Distinction between Apps and Extensions

An Application in the context of OpenCloud Web is an artifact which can be installed in an OpenCloud instance.
It serves two main purposes:

1. It makes the full app viewport (everything below the top bar) available to the application developer for any custom
   application code. This includes the ability to define views with routes, navigation items for the left sidebar, and more.
2. Through the `extensions` key in the application interface you can register extensions of any extension type. Those extensions
   are then available in standardized extension points. Additionally, they can be queried from the extension registry for
   your own purposes.

Both parts are optional. This means that an application can be a file editor without any custom extensions, or even contain
no custom application code at all and only host extensions to be registered in the extension registry, or a combination of both.

### Examples

You can find open source examples for apps and extensions in our [curated list of OpenCloud apps and extensions](https://github.com/opencloud-eu/awesome-apps).
Feel free to contribute or just be inspired for your own apps or extensions.

### Apps

To get started, define a `src/index.ts`. Below is the most basic example of its content:

```typescript
// Install '@opencloud-eu/web-pkg' as a devDependency first (only relevant for types and autocompletion, dependency is already provided by OpenCloud Web at runtime).
import {
  AppWrapperRoute,
  ApplicationFileExtension,
  defineWebApplication
} from '@opencloud-eu/web-pkg'


export default defineWebApplication({
  setup({ applicationConfig }) {
    // Here, you have access to the full injection context, meaning you can use all composables that we provide via web-pkg

    // Needs to be unique within all installed applications in any OpenCloud web instance
    // Should be short, unique and expressive as it is used as prefix on all routes within your application
    const appId = 'your-extension'

    // See extensions section below
    const extensions = [
        ...
    ]

    // See details below
    const navItems = [
      ...
    ]

    // See details below
    const routes = [
        ...
    ]

    return {
      appInfo: {
        name: $gettext('Your application name'),
        id: appId,
        icon: 'aliens', // See https://opencloud.design/#/Design%20Tokens/IconList for available options
      },
      extensions,
      navItems,
      routes
    }
  }
})
```

By defining an application via `defineWebApplication` you can provide the following:

- `appInfo` - the application metadata, which is used to make the application available via the app switcher and the app registry.
- `navItems` - the statically defined navigation items for the left sidebar. Only gets rendered when more than 1 navigation item exists at runtime.
  Additional dynamic navigation items can be registered via the extension registry.
- `routes` - the routes to the different views of your application. May be referenced within the `navItems`. Authentication requirements can be defined per item.
- `extensions` - the extensions to be registered in the extension registry. For more details see the `Extensions` section below.

If you want to learn how to implement an app for viewing and editing specific file types, please consult the [relevant documentation](./10-viewer-editor-apps.md) for detailed instructions and guidance.

To learn how to integrate an app into OpenCloud Web, please refer to the "Web Apps" section of the Web service docs ("Services" > "Web").

### Extensions

In contrast to applications, extensions usually have a rather small scope and dedicated functionality.

#### Extension Registry

The globally available extension registry provided by the OpenCloud Web runtime can be used to both register and query extensions. All extensions
which are being made available via an `app` get registered in the extension registry automatically. In your custom application code you can
then query any of the available extensions by providing an `extensionPoint` entity. Throughout the OpenCloud Web platform
and most prominently also in the `files` app we have defined some extension points which automatically use certain extensions, see the
`Extension Points` section below.

#### Extension Types

For building an extension you can choose from the types predefined by the OpenCloud Web extension system. See the full list of available extension types below.

1. `ActionExtension` (type `action`) - An extension that can register `Action` items which then get shown in various places (e.g. context menus, batch actions), depending on the
   extension points referenced in the extension respectively. Most commonly used for file and folder actions (e.g. copy, rename, delete, etc.). For details, please refer to the [action docs](./20-extension-types/10-action-extensions.md)
2. `SearchExtension` (type `search`) - An extension that can register additional search providers. For details, please refer to the [search docs](./20-extension-types/70-search-extensions.md).
3. `SidebarNavExtension` (type `sidebarNav`) - An extension that can register additional navigation items for the left sidebar. These can be scoped to specific apps, and programmatically enabled/disabled.
   For details, please refer to the [sidebar nav docs](./20-extension-types/50-left-sidebar-menu-item-extensions.md).
4. `SidebarPanelExtension`, (type `sidebarPanel`) - An extension that can register panels for the right sidebar. For details, please refer to the [sidebar panel docs](./20-extension-types/60-right-sidebar-panel-extensions.md).
5. `FolderViewExtension` (type `folderView`) - An extension that can register additional ways of displaying the content of a folder (resources like spaces, folders or files) to the user.
   For details, please refer to the [folder view docs](./20-extension-types/40-folder-view-extensions.md).
6. `CustomComponentExtension` (type `customComponent`) - An extension that can register a custom component for a render target. For details, please refer to the
   [custom component docs](./20-extension-types/30-custom-component-extensions.md)

You're free to introduce your own extension types within your application code and use the extension registry to query the available ones. However, if you have the impression
that an important extension type is missing and would be beneficial for the platform, please reach out to us by opening a [GitHub issue](https://github.com/opencloud-eu/web/issues/new/choose).

#### Extension Base Configuration

Any extension is required to define at least an `id` and a `type` in order to fulfill the generic `Extension` interface.

The `id` is supposed to be unique throughout the OpenCloud Web ecosystem. In order to keep `id`s readable for humans we didn't want to enforce uniqueness through e.g. uuids.
Instead, we chose to use dot-formatted namespaces like e.g. `com.github.opencloud-eu.web.files.search`. We'd like to encourage you to follow the same format for your own extensions.

For the `type` you can choose from the ones listed above or define a custom one.

In addition, you can also pass optional `extensionPointIds` to further limit the usage of an extension. With the right click context menu and the batch actions being
two different extension points, this could mean that a file action extension is only allowed in the context menu, but not in the batch actions.
You can find predefined extension point ids in the extension points section below.

#### Extension Points

There are standardized components and places where extensions are being used automatically. The following ones are currently provided by the OpenCloud Web runtime or
the `files` app. If you decide to develop an extension which fulfills the type and registers itself for the extensionPointId of the respective extension point,
your extension will be used automatically.

1. Left Sidebar for Navigation. ExtensionPointId `app.${appName}.navItems` (dynamically created for each app). Mounts extensions of type `sidebarNav`.
2. Global top bar
   1. Center area. ExtensionPointId `app.runtime.header.center`. Mounts extensions of type `customComponent`.
   2. Progress bar for the global loading state. ExtensionPointId `app.runtime.global-progress-bar`. Mounts a single extensions of type `customComponent`. If multiple exist, the user can choose via the account page.
3. Files app
   1. Right sidebar. ExtensionPointId `app.files.sidebar`. Mounts extensions of type `sidebarPanel`. Used in any file(s) context (files app, file viewer apps, file editor apps).
   2. Folder views for regular folders. ExtensionPointId `app.files.folder-views.folder`. Mounts extensions of type `folderView`.
   3. Folder views for the project spaces overview. ExtensionPointId `app.files.folder-views.project-spaces`. Mounts extensions of type `folderView`.
   4. Folder views for the favorites page. ExtensionPointId `app.files.folder-views.favorites`. Mounts extensions of type `folderView`.
   5. Right click context menu. ExtensionPointId `global.files.context-actions`. Mounts extensions of type `action`.
   6. Batch actions in the app bar above file lists. ExtensionPointId `global.files.batch-actions`. Mounts extensions of type `action`.
   7. Default actions (left click) on a file. ExtensionPointId `global.files.default-actions`. Mounts extensions of type `action`.
   8. Upload menu. ExtensionPointId `app.files.upload-menu`. Mounts extensions of type `action`.
   9. Quick actions. ExtensionPointId `app.files.quick-actions`. Mounts extensions of type `action`.
4. Global search providers. ExtensionPointId `app.search.providers`. Utilizes extensions of type `search` as search engines for the search input in the global top bar.
5. User preference panels. ExtensionPointId `app.runtime.preferences.panels`. Mounts extensions of type `customComponent`.

#### User Preferences for Extensions

To allow users to configure extensions, extension points can define user preferences. User preferences are defined as an object on the extension point configuration.
Whenever an extension point declares to accept user preferences, it will get listed with a dropdown on the Preferences page (reachable via top right user menu).
The user can then select one out of all the extensions which have been registered for this extension point.

### Helpful packages

We currently offer the following packages that can be integrated into your app, providing useful utilities and types.

- `web-client` - This package serves as an abstraction layer between the server APIs and an app or extension. It converts raw API data into objects with helpful types and utilities. For details, please refer to the package's [README.md](https://github.com/opencloud-eu/web/blob/main/packages/web-client/README.md).
- `web-pkg` - This package provides utilities, most importantly a variety of components and composables, that can be useful when developing apps and extensions. For details, please refer to the package's [README.md](https://github.com/opencloud-eu/web/blob/main/packages/web-pkg/README.md).
