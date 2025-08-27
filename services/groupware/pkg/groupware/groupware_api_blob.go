package groupware

import (
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/opencloud-eu/opencloud/pkg/jmap"
)

const (
	DefaultBlobDownloadType = "application/octet-stream"
)

func (g *Groupware) GetBlob(w http.ResponseWriter, r *http.Request) {
	g.respond(w, r, func(req Request) Response {
		blobId := chi.URLParam(req.r, UriParamBlobId)
		if blobId == "" {
			return req.parameterErrorResponse(UriParamBlobId, fmt.Sprintf("Invalid value for path parameter '%v': empty", UriParamBlobId))
		}

		res, _, err := g.jmap.GetBlob(req.GetAccountId(), req.session, req.ctx, req.logger, blobId)
		if err != nil {
			return req.errorResponseFromJmap(err)
		}
		blob := res.Blob
		if blob == nil {
			return notFoundResponse("")
		}
		return etagOnlyResponse(res, jmap.State(blob.Digest()))
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

		resp, err := g.jmap.UploadBlobStream(req.GetAccountId(), req.session, req.ctx, req.logger, contentType, body)
		if err != nil {
			return req.errorResponseFromJmap(err)
		}

		return etagOnlyResponse(resp, jmap.State(resp.Sha512))
	})
}

func (g *Groupware) DownloadBlob(w http.ResponseWriter, r *http.Request) {
	g.stream(w, r, func(req Request, w http.ResponseWriter) *Error {
		blobId := chi.URLParam(req.r, UriParamBlobId)
		name := chi.URLParam(req.r, UriParamBlobName)
		q := req.r.URL.Query()
		tipe := q.Get(QueryParamBlobType)
		if tipe == "" {
			tipe = DefaultBlobDownloadType
		}

		blob, jerr := g.jmap.DownloadBlobStream(req.GetAccountId(), blobId, name, tipe, req.session, req.ctx, req.logger)
		if blob != nil && blob.Body != nil {
			defer func(Body io.ReadCloser) {
				err := Body.Close()
				if err != nil {
					req.logger.Error().Err(err).Msg("failed to close response body")
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

		_, err := io.Copy(w, blob.Body)
		if err != nil {
			return req.observedParameterError(ErrorStreamingResponse)
		}

		return nil
	})
}
