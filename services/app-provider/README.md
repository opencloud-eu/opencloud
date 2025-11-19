# App Provider

The `app-provider` service provides the CS3 App Provider API for OpenCloud. It is responsible for managing and serving applications that can open files based on their MIME types.

The service works in conjunction with the `app-registry` service, which maintains the registry of available applications and their supported MIME types. When a client requests to open a file with a specific application, the `app-provider` service handles the request and coordinates with the application to provide the appropriate interface.

## Integration

The `app-provider` service integrates with:
- `app-registry` - For discovering which applications are available for specific MIME types
- `frontend` - The frontend service forwards app provider requests (default endpoint `/app`) to this service

## Configuration

The service can be configured via environment variables. Key configuration options include:
- `APP_PROVIDER_EXTERNAL_ADDR` - External address where the gateway service can reach the app provider

## Scalability

The app-provider service can be scaled horizontally as it primarily acts as a coordinator between applications and the OpenCloud backend services.
