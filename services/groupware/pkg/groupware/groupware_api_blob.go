package groupware

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func (g Groupware) GetBlob(w http.ResponseWriter, r *http.Request) {
	g.respond(w, r, func(req Request) (any, string, *Error) {
		blobId := chi.URLParam(req.r, UriParamBlobId)
		if blobId == "" {
			errorId := req.errorId()
			msg := fmt.Sprintf("Invalid value for path parameter '%v': empty", UriParamBlobId)
			return nil, "", apiError(errorId, ErrorInvalidRequestParameter,
				withDetail(msg),
				withSource(&ErrorSource{Parameter: UriParamBlobId}),
			)
		}

		res, err := g.jmap.GetBlob(req.GetAccountId(), req.session, req.ctx, req.logger, blobId)
		return res, res.Digest(), req.apiErrorFromJmap(err)
	})
}
