---
title: 'Getting Started'
sidebar_position: 1
---


## Installation

### Docker

Make sure to have Docker, Docker-Compose, Node.js and pnpm installed.

:::note
This setup currently doesn't work on Windows out of the box.

<details>
  <summary>Workaround</summary>
  
  One of our contributors has opened a PR to a dependency that prevents us from successfully bundling the frontend.
  Feel free to check out [their changes](https://github.com/egoist/rollup-plugin-postcss/pull/384) and build them locally if you absolutely want to work on Windows.
</details>
:::

After cloning the [source code](https://github.com/opencloud-eu/web), install the dependencies via `pnpm install` and bundle the frontend code by running `pnpm build:w`.

Then, you can start the server by running `docker-compose up opencloud` and access it via [https://host.docker.internal:9200](https://host.docker.internal:9200). If you're not using Docker Desktop, you might have to modify your `/etc/hosts` and add `127.0.0.1 host.docker.internal` to make the `host.docker.internal` links work.

The bundled frontend code automatically gets mounted into the Docker containers, recompiles on changes and you can log in using the demo user (admin/admin) and take a look around!

For more details on how to set up Web for development, please see [tooling](./20-development/30-tooling.md).

## Source Code

The source code is hosted at [https://github.com/opencloud-eu/web](https://github.com/opencloud-eu/web).

## Configuration

There are sample config files provided in the [config folder](https://github.com/opencloud-eu/web/tree/main/config) of the OpenCloud Web git repository. See below for some configuration details.

- `customTranslations` You can specify one or multiple files to overwrite existing translations. For example set this option to `[{url: "https://localhost:9200/customTranslations.json"}]`.

#### Options

- `options.accountEditLink` This accepts an object with the following optional fields to have a link on the account page:
    - `options.accountEditLink.href` Set a different target URL for the edit link. Make sure to prepend it with `http(s)://`.
- `options.disableFeedbackLink` Set this option to `true` to disable the feedback link in the topbar. Keeping it enabled (value `false` or absence of the option)
  allows OpenCloud to get feedback from your user base through a dedicated survey website.
- `options.feedbackLink` This accepts an object with the following optional fields to customize the feedback link in the topbar:
    - `options.feedbackLink.href` Set a different target URL for the feedback link. Make sure to prepend it with `http(s)://`. Defaults to `https://opencloud.eu/web-design-feedback`.
    - `options.feedbackLink.ariaLabel` Since the link only has an icon, you can set an e.g. screen reader accessible label. Defaults to `OpenCloud feedback survey`.
    - `options.feedbackLink.description` Provide any description you want to see as tooltip and as accessible description. Defaults to `Provide your feedback: We'd like to improve the web design and would be happy to hear your feedback. Thank you! Your OpenCloud team.`
- `options.sharingRecipientsPerPage` Sets the amount of users shown as recipients in the dropdown when sharing resources. Default amount is 200.
- `options.runningOnEos` Set this option to `true` if running on an [EOS storage backend](https://eos-web.web.cern.ch/eos-web/) to enable its specific features. Defaults to `false`.
- `options.cernFeatures` Enabling this will activate CERN-specific features. Defaults to `false`.
- `options.editor.autosaveEnabled` Specifies if the autosave for the editor apps is enabled.
- `options.editor.autosaveInterval` Specifies the time interval for the autosave of editor apps in seconds.
- `options.editor.openAsPreview` Specifies if non-personal files i.e. files in shares, spaces or public links are being opened in read only mode so the user needs to manually switch to edit mode. Can be set to `true`, `false` or an array of web app/editor names.
- `options.contextHelpersReadMore` Specifies whether the "Read more" link should be displayed or not.
- `options.tokenStorageLocal` Specifies whether the access token will be stored in the local storage when set to `true` or in the session storage when set to `false`. If stored in the local storage, login state will be persisted across multiple browser tabs, means no additional logins are required. Defaults to `true`.
- `options.loginUrl` Specifies the target URL to the login page. This is helpful when an external IdP is used. This option is disabled by default. Example URL like: 'https://www.myidp.com/login'.
- `options.logoutUrl` Adds a link to the user's profile page to point him to an external page, where he can manage his session and devices. This is helpful when an external IdP is used. This option is disabled by default.
- `options.userListRequiresFilter` Defines whether one or more filters must be set in order to list users in the Web admin settings. Set this option to 'true' if running in an environment with a lot of users and listing all users could slow down performance. Defaults to `false`.
- `options.concurrentRequests` This accepts an object with the following optional fields to customize the maximum number of concurrent requests in code paths where we limit concurrent requests
    - `resourceBatchActions` Concurrent number of file/folder/space batch actions like e.g. accepting shares. Defaults to 4.
    - `sse` Concurrent number of SSE event handlers. Defaults to 4.
    - `shares` Accepts an object regarding the following sharing related options:
        - `create` Concurrent number of share invites. Defaults to 4.
        - `list` Concurrent number of individually loaded shares. Defaults to 2.

#### Scripts and Styles

Web supports adding additional CSS and JavaScript files to further customize the user experience and adapt it to your specific needs. Please consider opening a feature request if you feel like your customization could be a benefit to the upstream project, and keep an eye open for future major releases of `web` since this API may change.

- `styles` expects an array of objects that specify a `href` attribute, pointing to the path/URL of your stylesheet, like `[{ "href": "css/custom.css" }]`.
- `scripts` expects an array of objects that specify a `src` attribute, pointing to the path/URL of your script, and an optional `async` attribute (defaults to false), like `[{ "src": "js/custom.js", "async": true }]`.

### Sentry

Web supports [Sentry](https://sentry.io/welcome/) to provide monitoring and error tracking.
To enable sending data to a Sentry instance, you can use the following configuration keys:

- `sentry.dsn` Should contain the DSN for your sentry project.
- `sentry.environment`: Lets you specify the environment to use in Sentry. Defaults to `production`.

Any other key under `sentry` will be forwarded to the Sentry initialization. You can find out more
settings in the [Sentry docs](https://docs.sentry.io/platforms/javascript/configuration/).

:::note
If you are using an old version of Sentry (9 and before), you might want to add the setting `sentry.autoSessionTracking: false` to avoid errors related to breaking changes introduced in the
integration libraries.
:::
