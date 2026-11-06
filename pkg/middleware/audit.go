package middleware

import (
	"net/http"

	"github.com/opencloud-eu/reva/v2/pkg/audit"
)

// AuditContext reads the audit request context the proxy sets in the
// X-Audit-Context header. The service can then emit audit events with it, and
// its gRPC calls pass it on. Only the proxy is reachable from outside and it
// always replaces the header, so the value can be trusted here.
func AuditContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if v := r.Header.Get(audit.Header); v != "" {
			r = r.WithContext(audit.ContextSetRequest(r.Context(), audit.ParseRequestContext(v)))
		}
		next.ServeHTTP(w, r)
	})
}
