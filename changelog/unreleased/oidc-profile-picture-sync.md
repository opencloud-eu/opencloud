Enhancement: Sync user profile pictures from OIDC claims

The proxy now reads a profile picture URL from the OIDC `picture` claim (configurable
via `PROXY_AUTOPROVISION_CLAIM_PICTURE`, defaults to the standard `picture` claim,
set to an empty string to disable) and emits a `ProfilePictureSyncRequested` event.
The graph service consumes this event, downloads the image and stores it as the
user's avatar. Allowed image URLs can be restricted via
`GRAPH_PROFILE_PICTURE_URL_ALLOWLIST` (glob patterns, defaults to the OpenCloud URL
host). A `UserProfilePictureUpdated` event is emitted after a successful update so
the UI can refresh the avatar without a page reload.

https://github.com/opencloud-eu/opencloud/issues/1499
https://github.com/opencloud-eu/opencloud/pull/2704
