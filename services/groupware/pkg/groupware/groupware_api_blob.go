package groupware

import (
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/opencloud-eu/opencloud/pkg/jmap"
	"github.com/opencloud-eu/opencloud/pkg/log"
)

const (
	DefaultBlobDownloadType = "application/octet-stream"
)

func (g *Groupware) GetBlobMeta(w http.ResponseWriter, r *http.Request) {
	g.respond(w, r, func(req Request) Response {
		blobId := chi.URLParam(req.r, UriParamBlobId)
		if blobId == "" {
			return req.parameterErrorResponse(UriParamBlobId, fmt.Sprintf("Invalid value for path parameter '%v': empty", UriParamBlobId))
		}

		accountId, err := req.GetAccountIdForBlob()
		if err != nil {
			return errorResponse(err)
		}
		logger := log.From(req.logger.With().Str(logAccountId, accountId))

		res, sessionState, state, lang, jerr := g.jmap.GetBlobMetadata(accountId, req.session, req.ctx, logger, req.language(), blobId)
		if jerr != nil {
			return req.errorResponseFromJmap(jerr)
		}
		blob := res
		if blob == nil {
			return notFoundResponse(sessionState)
		}
		return etagResponse(res, sessionState, state, lang)
	})
}

func (g *Groupware) UploadBlob(w http.ResponseWriter, r *http.Request) {
	g.respond(w, r, func(req Request) Response {
		contentType := r.Header.Get("Content-Type")
		body := r.Body
		if body != nil {
			defer func(Body io.ReadCloser) {
				err := Body.Close()
				if err != nil {
					req.logger.Error().Err(err).Msg("failed to close response body")
				}
			}(body)
		}

		accountId, err := req.GetAccountIdForBlob()
		if err != nil {
			return errorResponse(err)
		}
		logger := log.From(req.logger.With().Str(logAccountId, accountId))

		resp, lang, jerr := g.jmap.UploadBlobStream(accountId, req.session, req.ctx, logger, contentType, req.language(), body)
		if jerr != nil {
			return req.errorResponseFromJmap(jerr)
		}

		return etagOnlyResponse(resp, jmap.State(resp.Sha512), lang)
	})
}

func (g *Groupware) DownloadBlob(w http.ResponseWriter, r *http.Request) {
	g.stream(w, r, func(req Request, w http.ResponseWriter) *Error {
		blobId := chi.URLParam(req.r, UriParamBlobId)
		name := chi.URLParam(req.r, UriParamBlobName)
		q := req.r.URL.Query()
		typ := q.Get(QueryParamBlobType)

		accountId, gwerr := req.GetAccountIdForBlob()
		if gwerr != nil {
			return gwerr
		}
		logger := log.From(req.logger.With().Str(logAccountId, accountId))

		return req.serveBlob(blobId, name, typ, logger, accountId, w)
	})
}

func (r *Request) serveBlob(blobId string, name string, typ string, logger *log.Logger, accountId string, w http.ResponseWriter) *Error {
	if typ == "" {
		typ = DefaultBlobDownloadType
	}
	blob, lang, jerr := r.g.jmap.DownloadBlobStream(accountId, blobId, name, typ, r.session, r.ctx, logger, r.language())
	if blob != nil && blob.Body != nil {
		defer func(Body io.ReadCloser) {
			err := Body.Close()
			if err != nil {
				logger.Error().Err(err).Msg("failed to close response body")
			}
		}(blob.Body)
	}
	if jerr != nil {
		return r.apiErrorFromJmap(jerr)
	}
	if blob == nil {
		w.WriteHeader(http.StatusNotFound)
		return nil
	}

	if blob.Type != "" {
		w.Header().Add("Content-Type", blob.Type)
	}
	if blob.CacheControl != "" {
		w.Header().Add("Cache-Control", blob.CacheControl)
	}
	if blob.ContentDisposition != "" {
		w.Header().Add("Content-Disposition", blob.ContentDisposition)
	}
	if blob.Size >= 0 {
		w.Header().Add("Content-Size", strconv.Itoa(blob.Size))
	}
	if lang != "" {
		w.Header().Add("Content-Language", string(lang))
	}

	_, err := io.Copy(w, blob.Body)
	if err != nil {
		return r.observedParameterError(ErrorStreamingResponse)
	}

	return nil
}
