# Sharing

The `sharing` service provides the CS3 Sharing API for OpenCloud. It manages user shares and public link shares, implementing the core sharing functionality.

## Overview

The sharing service handles:
- User-to-user shares (share a file or folder with another user)
- Public link shares (share via a public URL)
- Share permissions and roles
- Share lifecycle management (create, update, delete)

This service works in conjunction with the storage providers (`storage-shares` and `storage-publiclink`) to persist and manage share information.

## Integration

The sharing service integrates with:
- `frontend` and `ocs` - Provide HTTP APIs that translate to CS3 sharing calls
- `storage-shares` - Stores and manages received shares
- `storage-publiclink` - Manages public link shares
- `graph` - Provides LibreGraph API for sharing with roles

## Share Types

The service supports different types of shares:
- **User shares** - Share resources with specific users
- **Group shares** - Share resources with groups
- **Public link shares** - Create public URLs for sharing
- **Federated shares** - Share with users on other OpenCloud instances (via `ocm` service)

## Configuration

Share behavior can be configured via environment variables:
- Password enforcement for public shares
- Auto-acceptance of shares
- Share permissions and restrictions

See the `frontend` service README for more details on share-related configuration options.

## Scalability

The sharing service depends on the configured storage backends for share metadata. Scalability characteristics depend on the chosen storage backend configuration.
