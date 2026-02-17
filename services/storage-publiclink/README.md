# Storage PublicLink

The `storage-publiclink` service provides storage backend functionality for public link shares in OpenCloud. It implements the CS3 storage provider interface specifically for working with public link shared resources.

## Overview

This service is part of the storage services family and is responsible for:
- Providing access to publicly shared resources
- Handling anonymous access to shared content

## Integration

The storage-publiclink service integrates with:
- `sharing` service - Manages and persists public link shares
- `frontend` service - Provides HTTP/WebDAV access to public links
- Storage drivers - Accesses the actual file content

## Storage Registry

The service is registered in the gateway's storage registry with:
- Provider ID: `7993447f-687f-490d-875c-ac95e89a62a4`
- Mount point: `/public`
- Space types: `grant` and `mountpoint`

See the `gateway` README for more details on storage registry configuration.

## Access Control

Public link shares can be configured with:
- Password protection
- Expiration dates
- Read-only or read-write permissions
- Download limits

## Scalability

The storage-publiclink service can be scaled horizontally.
