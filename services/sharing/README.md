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

### Auto-Accept Shares

When setting the `SHARING_AUTO_ACCEPT_SHARES` to `true` (sharing service), all
incoming shares will be accepted automatically. Users can overwrite this
setting individually in their profile. The deprecated
`FRONTEND_AUTO_ACCEPT_SHARES` is still supported for backwards compatibility.

### Disable Creating New Shares

Two global switches stop users from creating new shares. Everything that already exists keeps working: existing shares and links can still be used, changed and deleted.

- `OC_DISABLE_NEW_USER_SHARES=true` rejects new user, group and federated shares. Adding members to a space is still possible. The OCS sharee search returns no results and the capabilities announce that user sharing is disabled.
- `OC_DISABLE_NEW_PUBLIC_LINKS=true` rejects new public links and turning an internal link into a public link. Internal links can still be created. The capabilities announce that public links are disabled, so the web UI hides the link section.

Both are read by the `frontend`, `graph`, `sharing` and (for federated shares) `ocm` services. When the services run separately, set them for all of them.

## Scalability

The sharing service depends on the configured storage backends for share metadata. Scalability characteristics depend on the chosen storage backend configuration.
