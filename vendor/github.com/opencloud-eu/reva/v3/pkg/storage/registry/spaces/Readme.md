# Spaces Registry

The spaces registry recognizes individual spaces instead of storage providers.
While it is configured with a list of storage providers, it will query them for all storage spaces and use the space ids to resolve id based lookups.
Furthermore, path based lookups will take into account a path template to present a human readable file tree.

## Configuration

The spaces registry takes two configuration options:

1.  `home_template` is used to build a path for the users home. It uses a template that can access the current user in the context, e.g. `/users/{{.Id.OpaqueId}}`
2.  `rules` is a map of path patterns to config rules

### Patterns

A pattern can be a simple path like `/users` or a regex like `/users/[0-9]`. It can also contain a template `/users/{{.CurrentUser.Id.OpaqueId}}/Shares`.
The pattern is used when matching the path for path based requests.

### Rules

A rule has several properties:

*   `address` The ip address of the CS3 storage provider
*   `path_template` Will be renamed to space\_path or space\_mount\_point. If set this must be a valid go template.
*   `allowed_user_agents` FIXME this seems to be used to route requests based on user agent

It also carries filters that are sent with a ListStorageSpaces call to a storage provider

*   `space_type` only list spaces of this type
*   `space_owner_self` only list spaces where the user is owner (eg. for the users home)
*   `space_id` only list a specific space

## How to deal with name collisions

1.  The registry manages path segments that are aliases for storage space ids
2.  every user can have their own paths (because every user can have multiple incoming project / share spaces with the same display name, eg two incoming shares for 'Documents' or two different Project spaces with the same Name. To distinguish spaces with the same display name in the webdav api they need to be assigned a unique path = space id alias)
3.  aliases are uniqe per user
4.  a space has three identifiers:

*   a unique space id, used to allow clients to always distinguish spaces
*   a display name, that is assigned by the owner or managers, eg. project names or 'Phils Home' for personal spaces. They are not unique
*   an alias that is human readable and unique per user. It is used when listing paths on the CS3 global names as well as oc10 `/webdav` and `/dav/files/{username}` endpoints

5.  on the OpenCloud `/dav/spaces/{spaceid}/` endpoint the alias is actually not used because navigation happens by `{spaceid}`
6.  Every user has their own list of path to spaceid mappings, like one config file per user.

## consequences for storage providers

1.  when creating a spaces the storage provider does not know anything about aliases
2.  when listing the root of a storage provider with a path based reference it will present a list of storageids, not aliases (that is what the registry is for)
