# Users

The `users` service provides the CS3 Users API for OpenCloud. It is responsible for managing user information and authentication within the OpenCloud instance.

This service implements the CS3 identity user provider interface, allowing other services to query and manage user accounts. It works as a backend provider for the `graph` service when using the CS3 backend mode.

## Backend Integration

The users service can work with different storage backends:
- LDAP integration through the graph service
- Direct CS3 API implementation

When using the `graph` service with the CS3 backend (`GRAPH_IDENTITY_BACKEND=cs3`), the graph service queries user information through this service.

## API

The service provides CS3 gRPC APIs for:
- Listing users
- Getting user information
- Finding users by username, email, or ID

## Usage

The users service is only used internally by other OpenCloud services and not being accessed directly by clients. The `frontend`, `ocs`, and `graph` services translate HTTP API requests into CS3 API calls to this service.

## Scalability

Since the users service queries backend systems (like LDAP through the configured identity backend), it can be scaled horizontally without additional configuration when using stateless backends.
