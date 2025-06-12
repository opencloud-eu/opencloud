# Web

The web service embeds and serves the static files for the [OpenCloud Web Client](https://github.com/opencloud-eu/web).
Note that clients will respond with a connection error if the web service is not available.

The web service also provides a minimal API for branding functionality like changing the logo shown.

## Custom Compiled Web Assets

If you want to use your custom compiled web client assets instead of the embedded ones,
then you can do that by setting the `WEB_ASSET_CORE_PATH` variable to point to your compiled files.
See [OpenCloud Web / Getting Started](https://docs.opencloud.eu/clients/web/getting-started/) and [OpenCloud Web / Setup with OpenCloud](https://docs.opencloud.eu/clients/web/backend-opencloud/) for more details.

## Web UI Configuration

*   Single configuration settings of the embedded web UI can be defined via `WEB_OPTION_xxx` environment variables.
*   A json based configuration file can be used via the `WEB_UI_CONFIG_FILE` environment variable.
*   If a json based configuration file is used, these configurations take precedence over single options set.

### Web UI Options

Besides theming, the behavior of the web UI can be configured via options. See the environment variables `WEB_OPTION_xxx`
for more details.

### Web UI Config File

When defined via the `WEB_UI_CONFIG_FILE` environment variable, the configuration of the web UI can be made
with a [json based](https://github.com/opencloud-eu/web/tree/master/config) file.

### Embedding Web

Web can be consumed by another application in a stripped down version called “Embed mode”.
This mode is supposed to be used in the context of selecting or sharing resources.

For more details see the developer documentation [OpenCloud Web / Embed Mode](https://docs.opencloud.eu/clients/web/embed-mode/).
See the environment variables: `WEB_OPTION_MODE` and `WEB_OPTION_EMBED_TARGET` to configure the embedded mode.

## Web Apps

The administrator of the environment is capable of providing custom web applications to the users.
This feature is useful for organizations that want to provide third party or custom apps to their users.

It's important to note that the feature at the moment is only capable of providing static (js, mjs, e.g.) web applications
and does not support injection of dynamic web applications (custom dynamic backends).

### Loading Themes

Web themes are loaded, if added in the OpenCloud source code, at build-time from
`<opencloud_repo>/services/web/assets/themes`.
This cannot be manipulated at runtime.

Additionally, the administrator can provide custom themes by storing it in the path defined by the environment
variable `WEB_ASSET_THEMES_PATH`.

With the theme root directory defined, the system needs to know which theme to use.
This can be done by setting the `WEB_UI_THEME_PATH` environment variable.

The final theme is composed of the built-in and the custom theme provided by the
administrator via `WEB_ASSET_THEMES_PATH` and `WEB_UI_THEME_PATH`.

For example, OpenCloud by default contains a built-in OpenCloud theme.
If the administrator provides a custom theme via the `WEB_ASSET_THEMES_PATH` directory like,
`WEB_ASSET_THEMES_PATH/opencloud/themes.json`, this one will be used instead of the built-in one.

Some theme keys are mandatory, like the `common.shareRoles` settings.
Such mandatory keys are injected automatically at runtime if not provided.

### Loading Applications

Web applications are loaded, if added in the OpenCloud source code, at build-time from
`<opencloud_repo>/services/web/assets/apps`. This cannot be manipulated at runtime.

Additionally, the administrator can provide custom applications by storing them in the path defined by the environment
variable `WEB_ASSET_APPS_PATH`.

This environment variable defaults to the OpenCloud base data directory `$OC_BASE_DATA_PATH/web/assets/apps`,
but can be redefined with any path set manually.

The final list of available applications is composed of the built-in and the custom applications provided by the
administrator via `WEB_ASSET_APPS_PATH`.

For example, if OpenCloud contains a built-in extension named `image-viewer-dfx` and the administrator provides a custom application named `image-viewer-obj` via the `WEB_ASSET_APPS_PATH` directory, the user will be able to access both
applications from the WebUI.

### Application Structure

* Applications always have to follow a strict structure.\
Everything else is skipped and not considered as an application.
   *   Each application must be in its own directory accessed via `WEB_ASSET_APPS_PATH`.
   *   Each application directory must contain a `manifest.json` file.
   *   Each application directory can contain a `config.json` file.

* The `manifest.json` file contains the following fields:
   *   `entrypoint` - required\
       The entrypoint of the application like `index.js`, the path is relative to the parent directory.
   *   `config` - optional\
       A list of key-value pairs that are passed to the global web application configuration `apps.yaml`.

### Application Configuration

If a custom configuration is needed, the administrator must provide the required configuration inside the `$OC_BASE_DATA_PATH/config/apps.yaml` file.

NOTE: An application manifest should _never_ be changed manually, see [Using Custom Assets](#using-custom-assets) for customisation.

The `apps.yaml` file must contain a list of key-value pairs which gets merged with the `config` field. For example, if the `image-viewer-obj` application contains the following configuration:

```json
{
  "entrypoint": "index.js",
  "config": {
    "maxWidth": 1280,
    "maxHeight": 1280
  }
}
```

The `apps.yaml` file contains the following configuration:

```yaml
image-viewer-obj:
  config:
    maxHeight: 640
    maxSize: 512
```

optional each application can have its own configuration file, which will be loaded by the WEB service.

```json
{
  "config": {
    "maxWidth": 320
  }
}
```

The Merge order is as follows: local.config overwrites > global.config overwrites > manifest.config.
The result will be:

```json
{
  "external_apps": [
    {
      "id": "image-viewer-obj",
      "path": "index.js",
      "config": {
        "maxWidth": 320,
        "maxHeight": 640,
        "maxSize": 512
      }
    }
  ]
}
```

Besides the configuration from the `manifest.json` file,
the `apps.yaml` or the `config.json` file can also contain the following fields:

*   `disabled` - optional\
    Defaults to `false`. If set to `true`, the application will not be loaded.

### Using Custom Assets

Besides the configuration and application registration, in the process of loading the application assets, the system uses a mechanism to load custom assets.

This is useful for cases where just a single asset should be overwritten, like a logo or similar.

Consider the following: OpenCloud is shipped with a default web app named `image-viewer-dfx` which contains a logo,
but the administrator wants to provide a custom logo for that application.

This can be achieved using the path defined via `WEB_ASSET_APPS_PATH` and adding a custom structure like `WEB_ASSET_APPS_PATH/image-viewer-dfx/`. Here you can add all custom assets to load like `logo.png`. On loading the web app, custom assets defined overwrite default ones.

This also applies for the `manifest.json` file, if the administrator wants to provide a custom one.

## Miscellaneous

Please note that OpenCloud, in particular the web service, needs a restart to load new applications or changes to the `apps.yaml` file.
