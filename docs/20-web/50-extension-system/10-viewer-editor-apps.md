---
title: 'Viewer and editor apps'
sidebar_position: 1
---



## Viewer and editor apps

OpenCloud Web allows developers to implement apps for viewing and editing specific file types. For instance, the built-in preview app serves as the default application for opening media files like images, videos, or audio.

This section will guide you through the process of implementing such an app within OpenCloud Web.

### Basic app structure

An app is essentially a distinct package that must be specified as an external application in the Web configuration.

The structure of an app is quite simple and straightforward. Consider, for example, the [pdf-viewer app](https://github.com/opencloud-eu/web/tree/main/packages/web-app-pdf-viewer). It consists of a `package.json` file, a `src` directory containing all the source code, and a `l10n` directory for translations. Optionally, you may also include a `tests` directory if your application requires testing.

To learn more about apps in general, please refer to the [Web app docs](./index.md).

### App setup

Inside the `src` folder you will need an `index.ts` file that sets up the app so it can be registered by the Web runtime. It follows the basic structure as described in [the apps section](./index.md), so it may look like this:

```typescript
import { AppWrapperRoute, defineWebApplication, AppMenuItemExtension } from '@opencloud-eu/web-pkg'
import translations from '../l10n/translations.json'
import { useGettext } from 'vue3-gettext'
import { computed } from 'vue'

// This is the base component of your app.
import App from './App.vue'

export default defineWebApplication({
  setup() {
    // The ID of your app.
    const appId = 'advanced-pdf-viewer'

    const { $gettext } = useGettext()

    // This creates a route under which your app can be opened.
    // Later, this route will be bound to one or more file extensions.
    const routes = [
      {
        name: 'advanced-pdf-viewer',
        path: '/:driveAliasAndItem(.*)?',
        component: AppWrapperRoute(App, {
          applicationId: appId
        }),
        meta: {
          authContext: 'hybrid',
          title: $gettext('Advanced PDF Viewer'),
          patchCleanPath: true
        }
      }
    ]

    // if you want your app to be present in the app menu on the top left.
    const menuItems = computed<AppMenuItemExtension[]>(() => [
      {
        label: () => $gettext('Advanced PDF Viewer'),
        type: 'appMenuItem',
        handler: () => {
          // do stuff...
        }
      }
    ])

    return {
      appInfo: {
        name: 'Advanced PDF Viewer',
        id: appId,
        defaultExtension: 'pdf',
        extensions: [
          // This makes sure all files with the "pdf" extension will be routed to your app when being opened.
          // See the `ApplicationFileExtension` interface down below for a list of all possible properties.
          {
            extension: 'pdf',
            routeName: 'advanced-pdf-viewer',

            // Add this if you want your app to be present in the "New" file menu.
            newFileMenu: {
              menuTitle() {
                return $gettext('PDF document')
              }
            }
          }
        ]
      },
      routes,
      translations,
      extensions: menuItems
    }
  }
})
```

Here is the interface defining the `extensions` property of the `appInfo` object.

```typescript
interface ApplicationFileExtension {
  app?: string
  extension?: string
  createFileHandler?: (arg: {
    fileName: string
    space: SpaceResource
    currentFolder: Resource
  }) => Promise<Resource>
  hasPriority?: boolean
  label?: string
  name?: string
  icon?: string
  mimeType?: string
  newFileMenu?: { menuTitle: () => string }
  routeName?: string
}
```
