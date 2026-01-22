# Groupware

The OpenCloud Groupware service provides a REST API for performing all the backend operations needed by the OpenCloud Groupware frontends.

## OpenAPI Documentation

To generate the OpenAPI ("Swagger") documentation of the REST API, [`pnpm`](https://pnpm.io/) is a pre-requisite,
as well as [`the groupware-apidocs tool`](https://github.com/opencloud-eu/groupware-apidocs).

After building and installing `groupware-apidocs` somewhere in your `PATH`,
run the following command in this directory to generate the `swagger.yml` OpenAPI definition file:

```bash
make apidoc
```

To generate a static HTML file using [Redocly](https://redocly.com/), which will generate a file `api.html`:

```bash
make apidoc-static
```

### Favicon

A [favicon](https://developer.mozilla.org/en-US/docs/Glossary/Favicon) is inserted into the static (Redocly) HTML file as part of the build process in the `Makefile`, using [`favicon.png`](file:favicon.png) as the source, computing its base64 to insert it as an image using a [data URL](https://developer.mozilla.org/en-US/docs/Web/URI/Reference/Schemes/data) in order to embed it.

That is performed by the script [`apidoc-postprocess-html.ts`](file:apidoc-postprocess-html.ts) (which is done automatically when using then `Makefile` as described above.)

