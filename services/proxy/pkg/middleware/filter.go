package middleware

import (
	"net/http"
	"slices"

	"github.com/justinas/alice"

	"github.com/opencloud-eu/opencloud/pkg/log"
)

// FilterChain builds a dynamic middleware chain by invoking the factory with
// (w, r) and applies the returned constructors.
// Errors in the factory halt the chain; the caller is responsible for failing and must set status and body on w.
func FilterChain(constructorFactory func(w http.ResponseWriter, r *http.Request) ([]Constructor, error), logger log.Logger) Constructor {
	mwLogger := middlewareLogger(logger, "Probe")

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			chain, err := constructorFactory(w, r)
			if err != nil {
				rLogger := requestLogger(mwLogger, r)
				rLogger.Error().Err(err).Msg("probe failed")
				return
			}

			// remove invalid middlewares from the chain
			clean := slices.DeleteFunc(chain, func(c Constructor) bool {
				return c == nil
			})

			alice.New(clean...).Then(next).ServeHTTP(w, r)
		})
	}
}
