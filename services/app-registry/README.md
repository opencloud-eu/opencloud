# App Registry

The `app-registry` service is the single point where all apps register themselves and their respective supported mime types.

Administrators can set default applications on a per MIME type basis and also allow the creation of new files for certain MIME types. This per MIME type configuration also features a description, file extension option and an icon.

## MIME Type Configuration / Creation Allow List

The apps will register their supported MIME types automatically, so that users can open supported files with them.

Administrators can set default applications for each MIME type and also allow the creation of new files for certain mime types. This, per MIME type configuration, also features a description, file extension option and an icon.

### MIME Type Configuration

Modifing the MIME type config can only be achieved via a yaml configuration. Using environment variables is not possible. The following is a brief structure and a field description:

**Structure**

```yaml
app_registry:
  mimetypes:
  - mime_type: application/vnd.oasis.opendocument.spreadsheet
    extension: ods
    name: OpenSpreadsheet
    description: OpenDocument spreadsheet document
    icon: https://some-website.test/opendocument-spreadsheet-icon.png
    default_app: Collabora
    allow_creation: true
  - mime_type: ...
```

**Fields**

* `mime_type`\
The MIME type you want to configure.
* `extension`\
The file extension to be used for new files.
* `name`\
The name of the file / MIME type.
* `description`\
The human-readable description of the file / MIME type.
* `icon`\
The URL to an icon which should be used for that MIME type.
* `default_app`\
The name of the default app which opens this MIME type if the user doesn’t specify one.
* `allow_creation`\
Whether a user should be able to create new files of that MIME type (true or false).

## Endpoint Access

### Listing available apps and mime types

Clients, for example OpenCloud Web, need to offer users the available apps to open files and mime types for new file creation. This information can be obtained from this endpoint.

**Endpoint**: specified in the capabilities in `apps_url`, currently `/app/list`

**Method**: HTTP GET

**Authentication**: None

**Request example**:

```bash
curl 'https://opencloud.test/app/list'
```

**Response example**:

HTTP status code: 200

```json
{
  "mime-types": [
    {
      "mime_type": "application/pdf",
      "ext": "pdf",
      "app_providers": [
        {
          "name": "OnlyOffice",
          "icon": "https://some-website.test/onlyoffice-pdf-icon.png"
        }
      ],
      "name": "PDF",
      "description": "PDF document"
    },
    {
      "mime_type": "application/vnd.oasis.opendocument.text",
      "ext": "odt",
      "app_providers": [
        {
          "name": "Collabora",
          "icon": "https://some-website.test/collabora-odt-icon.png"
        },
        {
          "name": "OnlyOffice",
          "icon": "https://some-website.test/onlyoffice-odt-icon.png"
        }
      ],
      "name": "OpenDocument",
      "icon": "https://some-website.test/opendocument-text-icon.png",
      "description": "OpenDocument text document",
      "allow_creation": true,
      "default_application": "Collabora"
    },
    {
      "mime_type": "text/markdown",
      "ext": "md",
      "app_providers": [
        {
          "name": "CodiMD",
          "icon": "https://some-website.test/codimd-md-icon.png"
        }
      ],
      "name": "Markdown file",
      "description": "Markdown file",
      "allow_creation": true,
      "default_application": "CodiMD"
    },
    {
      "mime_type": "application/vnd.ms-word.document.macroenabled.12",
      "app_providers": [
        {
          "name": "Collabora",
          "icon": "https://some-website.test/collabora-word-icon.png"
        },
        {
          "name": "OnlyOffice",
          "icon": "https://some-website.test/onlyoffice-word-icon.png"
        }
      ]
    },
    {
      "mime_type": "application/vnd.ms-powerpoint.template.macroenabled.12",
      "app_providers": [
        {
          "name": "Collabora",
          "icon": "https://some-website.test/collabora-powerpoint-icon.png"
        }
      ]
    }
  ]
}
```

### Open a File With OpenCloud Web

**Endpoint**: specified in the capabilities in `open_web_url`, currently `/app/open-with-web`

**Method**: HTTP POST

**Authentication** (one of them):

- `Authorization` header with OIDC Bearer token for authenticated users or basic auth credentials (if enabled in OpenCloud)
- `X-Access-Token` header with a REVA token for authenticated users

**Query parameters**:

- `file_id` (mandatory): id of the file to be opened
- `app_name` (optional)
  - default (not given): default app for mime type
  - possible values depend on the app providers for a mimetype from the `/app/open` endpoint

**Request examples**:

```bash
curl -X POST 'https://opencloud.test/app/open-with-web?file_id=ZmlsZTppZAo='

curl -X POST 'https://opencloud.test/app/open-with-web?file_id=ZmlsZTppZAo=&app_name=Collabora'
```

**Response examples**:

The URI from the response JSON is intended to be opened with a GET request in a browser. If the user has not yet a session in the browser, a login flow is handled by OpenCloud Web.

HTTP status code: 200

```json
{
  "uri": "https://....."
}
```

**Example responses (error case)**:

See error cases for [Open a file with the app provider](#open-a-file-with-the-app-provider)

### Open a File With the App Provider

**Endpoint**: specified in the capabilities in `open_url`, currently `/app/open`

**Method**: HTTP POST

**Authentication** (one of them):

- `Authorization` header with OIDC Bearer token for authenticated users or basic auth credentials (if enabled in OpenCloud)
- `Public-Token` header with public link token for public links
- `X-Access-Token` header with a REVA token for authenticated users

**Query parameters**:

- `file_id` (mandatory): id of the file to be opened
- `app_name` (optional)
  - default (not given): default app for mime type
  - possible values depend on the app providers for a mimetype from the `/app/open` endpoint
- `view_mode` (optional)
  - default (not given): highest possible view mode, depending on the file permissions
  - possible values:
    - `write`: user can edit and download in the opening app
    - `read`: user can view and download from the opening app
    - `view`: user can view in the opening app (download is not possible)
- `lang` (optional)
  - default (not given): default language of the application (which might maybe use the browser language)
  - possible value is any ISO 639-1 language code. Examples:
    - de
    - en
    - es
    - ...

**Request examples**:

```bash
curl -X POST 'https://opencloud.test/app/open?file_id=ZmlsZTppZAo='

curl -X POST 'https://opencloud.test/app/open?file_id=ZmlsZTppZAo=&lang=de'

curl -X POST 'https://opencloud.test/app/open?file_id=ZmlsZTppZAo=&app_name=Collabora'

curl -X POST 'https://opencloud.test/app/open?file_id=ZmlsZTppZAo=&view_mode=read'

curl -X POST 'https://opencloud.test/app/open?file_id=ZmlsZTppZAo=&app_name=Collabora&view_mode=write'
```

**Response examples**:

All apps are expected to be opened in an iframe and the response will give some parameters for that action.

There are apps, which need to be opened in the iframe with a form post. The form post must include all form parameters included in the response. For these apps the response will look like this:

HTTP status code: 200

```json
{
  "app_url": "https://.....",
  "method": "POST",
  "form_parameters": {
    "access_token": "eyJ0...",
    "access_token_ttl": "1634300912000",
    "arbitrary_param": "lorem-ipsum"
  }
}
```

There are apps, which need to be opened in the iframe with a GET request. The GET request must have set all headers included in the response. For these apps the response will look like this:

HTTP status code: 200

```json
{
  "app_url": "https://...",
  "method": "GET",
  "headers": {
    "access_token": "eyJ0e...",
    "access_token_ttl": "1634300912000",
    "arbitrary_header": "lorem-ipsum"
  }
}
```

**Example responses (error case)**:

- missing `file_id`

  HTTP status code: 400

  ```json
  {
    "code": "INVALID_PARAMETER",
    "message": "missing file ID"
  }
  ```

- wrong `view_mode`

  HTTP status code: 400

  ```json
  {
    "code": "INVALID_PARAMETER",
    "message": "invalid view mode"
  }
  ```

- unknown `app_name`

  HTTP status code: 404

  ```json
  {
    "code": "RESOURCE_NOT_FOUND",
    "message": "error: not found: app 'Collabora' not found"
  }
  ```

- wrong / invalid file id

  HTTP status code: 400

  ```json
  {
    "code": "INVALID_PARAMETER",
    "message": "invalid file ID"
  }
  ```

- file id does not point to a file

  HTTP status code: 400

  ```json
  {
    "code": "INVALID_PARAMETER",
    "message": "the given file id does not point to a file"
  }
  ```

- file does not exist / unauthorized to open the file

  HTTP status code: 404

  ```json
  {
    "code": "RESOURCE_NOT_FOUND",
    "message": "file does not exist"
  }
  ```

- invalid language code

  HTTP status code: 400

  ```json
  {
    "code": "INVALID_PARAMETER",
    "message": "lang parameter does not contain a valid ISO 639-1 language code"
  }
  ```

### Creating a File With the App Provider

**Endpoint**: specified in the capabilities in `new_file_url`, currently `/app/new`

**Method**: HTTP POST

**Authentication** (one of them):

- `Authorization` header with OIDC Bearer token for authenticated users or basic auth credentials (if enabled in OpenCloud)
- `Public-Token` header with public link token for public links
- `X-Access-Token` header with a REVA token for authenticated users

**Query parameters**:

- `parent_container_id` (mandatory): ID of the folder in which the file will be created
- `filename` (mandatory): name of the new file
- `template` (optional): not yet implemented

**Request examples**:

```bash
curl -X POST 'https://opencloud.test/app/new?parent_container_id=c2lkOmNpZAo=&filename=test.odt'
```

**Response example**:

You will receive a file id of the freshly created file, which you can use to open the file in an editor.

```json
{
  "file_id": "ZmlsZTppZAo="
}
```

**Example responses (error case)**:

- missing `parent_container_id`

  HTTP status code: 400

  ```json
  {
    "code": "INVALID_PARAMETER",
    "message": "missing parent container ID"
  }
  ```

- missing `filename`

  HTTP status code: 400

  ```json
  {
    "code": "INVALID_PARAMETER",
    "message": "missing filename"
  }
  ```

- parent container not found

  HTTP status code: 404

  ```json
  {
    "code": "RESOURCE_NOT_FOUND",
    "message": "the parent container is not accessible or does not exist"
  }
  ```

- `parent_container_id` does not point to a container

  HTTP status code: 400

  ```json
  {
    "code": "INVALID_PARAMETER",
    "message": "the parent container id does not point to a container"
  }
  ```

- `filename` is invalid (e.g. includes a path segment)

  HTTP status code: 400

  ```json
  {
    "code": "INVALID_PARAMETER",
    "message": "the filename must not contain a path segment"
  }
  ```

- file already exists

  HTTP status code: 403

  ```json
  {
    "code": "RESOURCE_ALREADY_EXISTS",
    "message": "the file already exists"
  }
  ```
