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

		res, sessionState, lang, jerr := g.jmap.GetBlobMetadata(accountId, req.session, req.ctx, logger, req.language(), blobId)
		if jerr != nil {
			return req.errorResponseFromJmap(jerr)
		}
		blob := res.Blob
		if blob == nil {
			return notFoundResponse(sessionState)
		}
		digest := blob.Digest()
		if digest != "" {
			return etagResponse(res, sessionState, jmap.State(digest), lang)
		} else {
			return response(res, sessionState, lang)
		}
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
		if typ == "" {
			typ = DefaultBlobDownloadType
		}

		accountId, gwerr := req.GetAccountIdForBlob()
		if gwerr != nil {
			return gwerr
		}
		logger := log.From(req.logger.With().Str(logAccountId, accountId))

		blob, lang, jerr := g.jmap.DownloadBlobStream(accountId, blobId, name, typ, req.session, req.ctx, logger, req.language())
		if blob != nil && blob.Body != nil {
			defer func(Body io.ReadCloser) {
				err := Body.Close()
				if err != nil {
					logger.Error().Err(err).Msg("failed to close response body")
				}
			}(blob.Body)
		}
		if jerr != nil {
			return req.apiErrorFromJmap(jerr)
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
			return req.observedParameterError(ErrorStreamingResponse)
		}

		return nil
	})
}
