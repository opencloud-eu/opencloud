# Groups

The `groups` service provides the CS3 Groups API for OpenCloud. It is responsible for managing group information and memberships within the OpenCloud instance.

This service implements the CS3 identity group provider interface, allowing other services to query and manage groups. It works as a backend provider for the `graph` service when using the CS3 backend mode.

## Backend Integration

The groups service can work with different storage backends:
- LDAP integration through the graph service
- Direct CS3 API implementation

When using the `graph` service with the CS3 backend (`GRAPH_IDENTITY_BACKEND=cs3`), the graph service queries group information through this service.

## API

The service provides CS3 gRPC APIs for:
- Listing groups
- Getting group information
- Finding groups by name or ID
- Managing group memberships

## Usage

The groups service is only used internally by other OpenCloud services and not being accessed directly by clients. The `frontend` and `ocs` services translate HTTP API requests into CS3 API calls to this service.

## Scalability

Since the groups service queries backend systems (like LDAP through the configured identity backend), it can be scaled horizontally without additional configuration when using stateless backends.
