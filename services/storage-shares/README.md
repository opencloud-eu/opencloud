# Storage Shares

The `storage-shares` service provides storage backend functionality for user and group shares in OpenCloud. It implements the CS3 storage provider interface specifically for working with shared resources.

## Overview

This service is part of the storage services family and is responsible for:
- Providing a virtual view of received shares
- Handling access to resources shared by other users

## Integration

The storage-shares service integrates with:
- `sharing` service - Manages and persists shares
- `storage-users` service - Accesses the underlying file content
- `frontend` service - Provides HTTP/WebDAV access to shares

## Virtual Shares Folder

The service provides a virtual "Shares" folder for each user where all received shares are mounted. This allows users to access all files and folders that have been shared with them in a centralized location.

## Storage Registry

The service is registered in the gateway's storage registry with:
- Provider ID: `a0ca6a90-a365-4782-871e-d44447bbc668`
- Mount point: `/users/{{.CurrentUser.Id.OpaqueId}}/Shares`
- Space types: `virtual`, `grant`, and `mountpoint`

See the `gateway` README for more details on storage registry configuration.

## Scalability

The storage-shares service can be scaled horizontally.
