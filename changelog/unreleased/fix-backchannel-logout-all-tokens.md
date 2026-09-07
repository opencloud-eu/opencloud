Bugfix: Revoke every cached token during backchannel logout

Backchannel logout now invalidates all accepted access tokens associated with
the requested session or subject, including tokens issued before a refresh.
Revoked tokens cannot authenticate again through local JWT verification when
userinfo lookup is disabled, and concurrent claims writes cannot overwrite a
revocation. Notification failures no longer prevent token invalidation.

OIDC logout state uses a dedicated cache namespace with per-record expiry and
migrates existing persistent claims on startup. Tokens without a verified
expiry retain their logout state indefinitely. See the proxy caching
documentation for persistence and upgrade details.
