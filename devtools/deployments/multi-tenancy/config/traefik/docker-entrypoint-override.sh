set -e

printenv
# Function to add arguments to the command
add_arg() {
    TRAEFIK_CMD="$TRAEFIK_CMD $1"
}

# Initialize the base command
TRAEFIK_CMD="traefik"

# Base Traefik arguments (from your existing configuration)
add_arg "--log.level=${TRAEFIK_LOG_LEVEL:-ERROR}"
# enable dashboard
add_arg "--api.dashboard=true"
# define entrypoints
add_arg "--entryPoints.http.address=:80"
add_arg "--entryPoints.http.http.redirections.entryPoint.to=https"
add_arg "--entryPoints.http.http.redirections.entryPoint.scheme=https"
add_arg "--entryPoints.https.address=:443"
# change default timeouts for long-running requests
# this is needed for webdav clients that do not support the TUS protocol
add_arg "--entryPoints.https.transport.respondingTimeouts.readTimeout=12h"
add_arg "--entryPoints.https.transport.respondingTimeouts.writeTimeout=12h"
add_arg "--entryPoints.https.transport.respondingTimeouts.idleTimeout=3m"
# docker provider (get configuration from container labels)
add_arg "--providers.docker.endpoint=unix:///var/run/docker.sock"
add_arg "--providers.docker.exposedByDefault=false"
# access log
add_arg "--accessLog=${TRAEFIK_ACCESS_LOG:-false}"
add_arg "--accessLog.format=json"
add_arg "--accessLog.fields.headers.names.X-Request-Id=keep"

# Add Let's Encrypt configuration if enabled
if [ "${TRAEFIK_SERVICES_TLS_CONFIG}" = "tls.certresolver=letsencrypt" ]; then
    echo "Configuring Traefik with Let's Encrypt..."
    add_arg "--certificatesResolvers.letsencrypt.acme.email=${TRAEFIK_ACME_MAIL:-example@example.org}"
    add_arg "--certificatesResolvers.letsencrypt.acme.storage=/certs/acme.json"
    add_arg "--certificatesResolvers.letsencrypt.acme.httpChallenge.entryPoint=http"
    add_arg "--certificatesResolvers.letsencrypt.acme.caserver=${TRAEFIK_ACME_CASERVER:-https://acme-v02.api.letsencrypt.org/directory}"
fi

# Add local certificate configuration if enabled
if [ "${TRAEFIK_SERVICES_TLS_CONFIG}" = "tls=true" ]; then
    echo "Configuring Traefik with local certificates..."
    add_arg "--providers.file.directory=/etc/traefik/dynamic"
    add_arg "--providers.file.watch=true"
fi

# Warning if neither certificate method is enabled
if [ "${TRAEFIK_SERVICES_TLS_CONFIG}" != "tls=true" ] && [ "${TRAEFIK_SERVICES_TLS_CONFIG}" != "tls.certresolver=letsencrypt" ]; then
    echo "WARNING: Neither Let's Encrypt nor local certificates are enabled."
    echo "HTTPS will not work properly without certificate configuration."
fi

# Add any custom arguments from environment variable
if [ -n "${TRAEFIK_CUSTOM_ARGS}" ]; then
    echo "Adding custom Traefik arguments: ${TRAEFIK_CUSTOM_ARGS}"
    TRAEFIK_CMD="$TRAEFIK_CMD $TRAEFIK_CUSTOM_ARGS"
fi

# Add any additional arguments passed to the script
for arg in "$@"; do
    add_arg "$arg"
done

# Print the final command for debugging
echo "Starting Traefik with command:"
echo "$TRAEFIK_CMD"

# Execute Traefik
exec $TRAEFIK_CMD
