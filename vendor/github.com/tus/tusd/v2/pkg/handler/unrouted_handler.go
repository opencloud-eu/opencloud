package handler

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"math"
	"mime"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"golang.org/x/exp/slog"
)

const UploadLengthDeferred = "1"

type draftVersion string

// These are the different interoperability versions defines in the different
// versions of the resumable uploads draft from the HTTP working group.
// See https://datatracker.ietf.org/doc/draft-ietf-httpbis-resumable-upload/
const (
	interopVersion3 draftVersion = "3" // From draft version -01
	interopVersion4 draftVersion = "4" // From draft version -02
	interopVersion5 draftVersion = "5" // From draft version -03
	interopVersion6 draftVersion = "6" // From draft version -04 and -05
)

var (
	reForwardedHost  = regexp.MustCompile(`host="?([^;"]+)`)
	reForwardedProto = regexp.MustCompile(`proto=(https?)`)
	// We only allow certain URL-safe characters in upload IDs. URL-safe in this means
	// that their are allowed in a URI's path component according to RFC 3986.
	// See https://datatracker.ietf.org/doc/html/rfc3986#section-3.3
	reValidUploadId = regexp.MustCompile(`^[A-Za-z0-9\-._~%!$'()*+,;=/:@]*$`)
)

var (
	ErrUnsupportedVersion               = NewError("ERR_UNSUPPORTED_VERSION", "missing, invalid or unsupported Tus-Resumable header", http.StatusPreconditionFailed)
	ErrMaxSizeExceeded                  = NewError("ERR_MAX_SIZE_EXCEEDED", "maximum size exceeded", http.StatusRequestEntityTooLarge)
	ErrInvalidContentType               = NewError("ERR_INVALID_CONTENT_TYPE", "missing or invalid Content-Type header", http.StatusBadRequest)
	ErrInvalidUploadLength              = NewError("ERR_INVALID_UPLOAD_LENGTH", "missing or invalid Upload-Length header", http.StatusBadRequest)
	ErrInvalidOffset                    = NewError("ERR_INVALID_OFFSET", "missing or invalid Upload-Offset header", http.StatusBadRequest)
	ErrNotFound                         = NewError("ERR_UPLOAD_NOT_FOUND", "upload not found", http.StatusNotFound)
	ErrFileLocked                       = NewError("ERR_UPLOAD_LOCKED", "file currently locked", http.StatusLocked)
	ErrLockTimeout                      = NewError("ERR_LOCK_TIMEOUT", "failed to acquire lock before timeout", http.StatusInternalServerError)
	ErrMismatchOffset                   = NewError("ERR_MISMATCHED_OFFSET", "mismatched offset", http.StatusConflict)
	ErrSizeExceeded                     = NewError("ERR_UPLOAD_SIZE_EXCEEDED", "upload's size exceeded", http.StatusRequestEntityTooLarge)
	ErrNotImplemented                   = NewError("ERR_NOT_IMPLEMENTED", "feature not implemented", http.StatusNotImplemented)
	ErrUploadNotFinished                = NewError("ERR_UPLOAD_NOT_FINISHED", "one of the partial uploads is not finished", http.StatusBadRequest)
	ErrInvalidConcat                    = NewError("ERR_INVALID_CONCAT", "invalid Upload-Concat header", http.StatusBadRequest)
	ErrConcatenationUnsupported         = NewError("ERR_CONCATENATION_UNSUPPORTED", "Upload-Concat header is not supported by server", http.StatusBadRequest)
	ErrModifyFinal                      = NewError("ERR_MODIFY_FINAL", "modifying a final upload is not allowed", http.StatusForbidden)
	ErrUploadLengthAndUploadDeferLength = NewError("ERR_AMBIGUOUS_UPLOAD_LENGTH", "provided both Upload-Length and Upload-Defer-Length", http.StatusBadRequest)
	ErrInvalidUploadDeferLength         = NewError("ERR_INVALID_UPLOAD_LENGTH_DEFER", "invalid Upload-Defer-Length header", http.StatusBadRequest)
	ErrUploadStoppedByServer            = NewError("ERR_UPLOAD_STOPPED", "upload has been stopped by server", http.StatusBadRequest)
	ErrUploadRejectedByServer           = NewError("ERR_UPLOAD_REJECTED", "upload creation has been rejected by server", http.StatusBadRequest)
	ErrUploadTerminationRejected        = NewError("ERR_UPLOAD_TERMINATION_REJECTED", "upload termination has been rejected by server", http.StatusBadRequest)
	ErrUploadInterrupted                = NewError("ERR_UPLOAD_INTERRUPTED", "upload has been interrupted by another request for this upload resource", http.StatusBadRequest)
	ErrServerShutdown                   = NewError("ERR_SERVER_SHUTDOWN", "request has been interrupted because the server is shutting down", http.StatusServiceUnavailable)
	ErrOriginNotAllowed                 = NewError("ERR_ORIGIN_NOT_ALLOWED", "request origin is not allowed", http.StatusForbidden)
	ErrUnexpectedEOF                    = NewError("ERR_UNEXPECTED_EOF", "server expected to receive more bytes", http.StatusBadRequest)

	// These two responses are 500 for backwards compatability. Clients might receive a timeout response
	// when the upload got interrupted. Most clients will not retry 4XX but only 5XX, so we responsd with 500 here.
	ErrReadTimeout     = NewError("ERR_READ_TIMEOUT", "timeout while reading request body", http.StatusInternalServerError)
	ErrConnectionReset = NewError("ERR_CONNECTION_RESET", "TCP connection reset by peer", http.StatusInternalServerError)
)

// UnroutedHandler exposes methods to handle requests as part of the tus protocol,
// such as PostFile, HeadFile, PatchFile and DelFile. In addition the GetFile method
// is provided which is, however, not part of the specification.
type UnroutedHandler struct {
	config        Config
	composer      *StoreComposer
	isBasePathAbs bool
	basePath      string
	logger        *slog.Logger
	extensions    string

	// CompleteUploads is used to send notifications whenever an upload is
	// completed by a user. The HookEvent will contain information about this
	// upload after it is completed. Sending to this channel will only
	// happen if the NotifyCompleteUploads field is set to true in the Config
	// structure. Notifications will also be sent for completions using the
	// Concatenation extension.
	CompleteUploads chan HookEvent
	// TerminatedUploads is used to send notifications whenever an upload is
	// terminated by a user. The HookEvent will contain information about this
	// upload gathered before the termination. Sending to this channel will only
	// happen if the NotifyTerminatedUploads field is set to true in the Config
	// structure.
	TerminatedUploads chan HookEvent
	// UploadProgress is used to send notifications about the progress of the
	// currently running uploads. For each open PATCH request, every second
	// a HookEvent instance will be send over this channel with the Offset field
	// being set to the number of bytes which have been transfered to the server.
	// Please be aware that this number may be higher than the number of bytes
	// which have been stored by the data store! Sending to this channel will only
	// happen if the NotifyUploadProgress field is set to true in the Config
	// structure.
	UploadProgress chan HookEvent
	// CreatedUploads is used to send notifications about the uploads having been
	// created. It triggers post creation and therefore has all the HookEvent incl.
	// the ID available already. It facilitates the post-create hook. Sending to
	// this channel will only happen if the NotifyCreatedUploads field is set to
	// true in the Config structure.
	CreatedUploads chan HookEvent
	// Metrics provides numbers of the usage for this handler.
	Metrics Metrics
}

// NewUnroutedHandler creates a new handler without routing using the given
// configuration. It exposes the http handlers which need to be combined with
// a router (aka mux) of your choice. If you are looking for preconfigured
// handler see NewHandler.
func NewUnroutedHandler(config Config) (*UnroutedHandler, error) {
	if err := config.validate(); err != nil {
		return nil, err
	}

	// Only promote extesions using the Tus-Extension header which are implemented
	extensions := "creation,creation-with-upload"
	if config.StoreComposer.UsesTerminater && !config.DisableTermination {
		extensions += ",termination"
	}
	if config.StoreComposer.UsesConcater && !config.DisableConcatenation {
		extensions += ",concatenation"
	}
	if config.StoreComposer.UsesLengthDeferrer {
		extensions += ",creation-defer-length"
	}

	handler := &UnroutedHandler{
		config:            config,
		composer:          config.StoreComposer,
		basePath:          config.BasePath,
		isBasePathAbs:     config.isAbs,
		CompleteUploads:   make(chan HookEvent),
		TerminatedUploads: make(chan HookEvent),
		UploadProgress:    make(chan HookEvent),
		CreatedUploads:    make(chan HookEvent),
		logger:            config.Logger,
		extensions:        extensions,
		Metrics:           newMetrics(),
	}

	return handler, nil
}

// SupportedExtensions returns a comma-separated list of the supported tus extensions.
// The availability of an extension usually depends on whether the provided data store
// implements some additional interfaces.
func (handler *UnroutedHandler) SupportedExtensions() string {
	return handler.extensions
}

// Middleware checks various aspects of the request and ensures that it
// conforms with the spec. Also handles method overriding for clients which
// cannot make PATCH AND DELETE requests. If you are using the tusd handlers
// directly you will need to wrap at least the POST and PATCH endpoints in
// this middleware.
func (handler *UnroutedHandler) Middleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Construct our own context and make it available in the request. Successive logic
		// should use handler.getContext to retrieve it
		c := handler.newContext(w, r)
		r = r.WithContext(c)

		// Set the initial read deadline for consuming the request body. All headers have already been read,
		// so this is only for reading the request body. While reading, we regularly update the read deadline
		// so this deadline is usually not final. See the bodyReader and writeChunk.
		// We also update the write deadline, but makes sure that it is larger than the read deadline, so we
		// can still write a response in the case of a read timeout.
		if err := c.resC.SetReadDeadline(time.Now().Add(handler.config.NetworkTimeout)); err != nil {
			c.log.WarnContext(c, "NetworkControlError", "error", err)
		}
		if err := c.resC.SetWriteDeadline(time.Now().Add(2 * handler.config.NetworkTimeout)); err != nil {
			c.log.WarnContext(c, "NetworkControlError", "error", err)
		}

		// Allow overriding the HTTP method. The reason for this is
		// that some libraries/environments do not support PATCH and
		// DELETE requests, e.g. Flash in a browser and parts of Java.
		if newMethod := r.Header.Get("X-HTTP-Method-Override"); r.Method == "POST" && newMethod != "" {
			r.Method = newMethod
		}

		c.log.InfoContext(c, "RequestIncoming")

		handler.Metrics.incRequestsTotal(r.Method)

		header := w.Header()

		cors := handler.config.Cors
		if origin := r.Header.Get("Origin"); !cors.Disable && origin != "" {
			originIsAllowed := cors.AllowOrigin.MatchString(origin)
			if !originIsAllowed {
				handler.sendError(c, ErrOriginNotAllowed)
				return
			}

			header.Set("Access-Control-Allow-Origin", origin)
			header.Set("Vary", "Origin")

			if cors.AllowCredentials {
				header.Add("Access-Control-Allow-Credentials", "true")
			}

			if r.Method == "OPTIONS" {
				// Preflight request
				header.Add("Access-Control-Allow-Methods", cors.AllowMethods)
				header.Add("Access-Control-Allow-Headers", cors.AllowHeaders)
				header.Set("Access-Control-Max-Age", cors.MaxAge)
			} else {
				// Actual request
				header.Add("Access-Control-Expose-Headers", cors.ExposeHeaders)
			}
		}

		// Detect requests with tus v1 protocol vs the IETF resumable upload draft
		isTusV1 := !handler.usesIETFDraft(r)

		if isTusV1 {
			// Set current version used by the server
			header.Set("Tus-Resumable", "1.0.0")
		}

		// Add nosniff to all responses https://golang.org/src/net/http/server.go#L1429
		header.Set("X-Content-Type-Options", "nosniff")

		// Set appropriated headers in case of OPTIONS method allowing protocol
		// discovery and end with an 204 No Content
		if r.Method == "OPTIONS" {
			ietfDraftLimits := "min-size=0"

			if handler.config.MaxSize > 0 {
				maxSizeStr := strconv.FormatInt(handler.config.MaxSize, 10)
				header.Set("Tus-Max-Size", maxSizeStr)
				ietfDraftLimits += ",max-size=" + maxSizeStr
			}

			if handler.usesIETFDraft(r) {
				header.Set("Upload-Limit", ietfDraftLimits)
			}

			header.Set("Tus-Version", "1.0.0")
			header.Set("Tus-Extension", handler.extensions)

			// Although the 204 No Content status code is a better fit in this case,
			// since we do not have a response body included, we cannot use it here
			// as some browsers only accept 200 OK as successful response to a
			// preflight request. If we send them the 204 No Content the response
			// will be ignored or interpreted as a rejection.
			// For example, the Presto engine, which is used in older versions of
			// Opera, Opera Mobile and Opera Mini, handles CORS this way.
			handler.sendResp(c, HTTPResponse{
				StatusCode: http.StatusOK,
			})
			return
		}

		// Test if the version sent by the client is supported
		// GET and HEAD methods are not checked since a browser may visit this URL and does
		// not include this header. GET requests are not part of the specification.
		if r.Method != "GET" && r.Method != "HEAD" && r.Header.Get("Tus-Resumable") != "1.0.0" && isTusV1 {
			handler.sendError(c, ErrUnsupportedVersion)
			return
		}

		// Proceed with routing the request
		h.ServeHTTP(w, r)
	})
}

// PostFile creates a new file upload using the datastore after validating the
// length and parsing the metadata.
func (handler *UnroutedHandler) PostFile(w http.ResponseWriter, r *http.Request) {
	if handler.usesIETFDraft(r) {
		handler.PostFileV2(w, r)
		return
	}

	c := handler.getContext(w, r)

	// Check for presence of application/offset+octet-stream. If another content
	// type is defined, it will be ignored and treated as none was set because
	// some HTTP clients may enforce a default value for this header.
	containsChunk := r.Header.Get("Content-Type") == "application/offset+octet-stream"

	// Only use the proper Upload-Concat header if the concatenation extension
	// is even supported by the data store.
	var concatHeader string
	if handler.composer.UsesConcater {
		concatHeader = r.Header.Get("Upload-Concat")
	}

	if concatHeader != "" && handler.config.DisableConcatenation {
		handler.sendError(c, ErrConcatenationUnsupported)
		return
	}

	// Parse Upload-Concat header
	isPartial, isFinal, partialUploadIDs, err := parseConcat(concatHeader, handler.basePath)
	if err != nil {
		handler.sendError(c, err)
		return
	}

	// If the upload is a final upload created by concatenation multiple partial
	// uploads the size is sum of all sizes of these files (no need for
	// Upload-Length header)
	var size int64
	var sizeIsDeferred bool
	var partialUploads []Upload
	if isFinal {
		// A final upload must not contain a chunk within the creation request
		if containsChunk {
			handler.sendError(c, ErrModifyFinal)
			return
		}

		partialUploads, size, err = handler.sizeOfUploads(c, partialUploadIDs)
		if err != nil {
			handler.sendError(c, err)
			return
		}
	} else {
		uploadLengthHeader := r.Header.Get("Upload-Length")
		uploadDeferLengthHeader := r.Header.Get("Upload-Defer-Length")
		size, sizeIsDeferred, err = handler.validateNewUploadLengthHeaders(uploadLengthHeader, uploadDeferLengthHeader)
		if err != nil {
			handler.sendError(c, err)
			return
		}
	}

	// Test whether the size is still allowed
	if handler.config.MaxSize > 0 && size > handler.config.MaxSize {
		handler.sendError(c, ErrMaxSizeExceeded)
		return
	}

	// Parse metadata
	meta := ParseMetadataHeader(r.Header.Get("Upload-Metadata"))

	info := FileInfo{
		Size:           size,
		SizeIsDeferred: sizeIsDeferred,
		MetaData:       meta,
		IsPartial:      isPartial,
		IsFinal:        isFinal,
		PartialUploads: partialUploadIDs,
	}

	resp := HTTPResponse{
		StatusCode: http.StatusCreated,
		Header:     HTTPHeader{},
	}

	if handler.config.PreUploadCreateCallback != nil {
		resp2, changes, err := handler.config.PreUploadCreateCallback(newHookEvent(c, info))
		if err != nil {
			handler.sendError(c, err)
			return
		}
		resp = resp.MergeWith(resp2)

		// Apply changes returned from the pre-create hook.
		if changes.ID != "" {
			if err := validateUploadId(changes.ID); err != nil {
				handler.sendError(c, err)
				return
			}

			info.ID = changes.ID
		}

		if changes.MetaData != nil {
			info.MetaData = changes.MetaData
		}

		if changes.Storage != nil {
			info.Storage = changes.Storage
		}
	}

	upload, err := handler.composer.Core.NewUpload(c, info)
	if err != nil {
		handler.sendError(c, err)
		return
	}

	info, err = upload.GetInfo(c)
	if err != nil {
		handler.sendError(c, err)
		return
	}

	id := info.ID

	// Add the Location header directly after creating the new resource to even
	// include it in cases of failure when an error is returned
	url := handler.absFileURL(r, id)
	resp.Header["Location"] = url

	handler.Metrics.incUploadsCreated()
	c.log = c.log.With("id", id)
	c.log.InfoContext(c, "UploadCreated", "size", size, "url", url)

	if handler.config.NotifyCreatedUploads {
		handler.CreatedUploads <- newHookEvent(c, info)
	}

	if isFinal {
		concatableUpload := handler.composer.Concater.AsConcatableUpload(upload)
		if err := concatableUpload.ConcatUploads(c, partialUploads); err != nil {
			handler.sendError(c, err)
			return
		}
		info.Offset = size

		resp, err = handler.emitFinishEvents(c, resp, info)
		if err != nil {
			handler.sendError(c, err)
			return
		}
	}

	if containsChunk {
		if handler.composer.UsesLocker {
			lock, err := handler.lockUpload(c, id)
			if err != nil {
				handler.sendError(c, err)
				return
			}

			defer lock.Unlock()
		}

		resp, err = handler.writeChunk(c, resp, upload, info)
		if err != nil {
			handler.sendError(c, err)
			return
		}
	} else if !sizeIsDeferred && size == 0 {
		// Directly finish the upload if the upload is empty (i.e. has a size of 0).
		// This statement is in an else-if block to avoid causing duplicate calls
		// to finishUploadIfComplete if an upload is empty and contains a chunk.
		resp, err = handler.finishUploadIfComplete(c, resp, upload, info)
		if err != nil {
			handler.sendError(c, err)
			return
		}

	}

	handler.sendResp(c, resp)
}

// PostFile creates a new file upload using the datastore after validating the
// length and parsing the metadata.
func (handler *UnroutedHandler) PostFileV2(w http.ResponseWriter, r *http.Request) {
	currentUploadDraftInteropVersion := getIETFDraftInteropVersion(r)
	c := handler.getContext(w, r)

	// Parse headers
	contentType := r.Header.Get("Content-Type")
	contentDisposition := r.Header.Get("Content-Disposition")
	willCompleteUpload := isIETFDraftUploadComplete(r)

	info := FileInfo{
		MetaData: make(MetaData),
	}

	size, sizeIsDeferred, err := getIETFDraftUploadLength(r)
	if err != nil {
		handler.sendError(c, err)
		return
	}

	if !sizeIsDeferred {
		info.Size = size
	} else {
		// Error out if the storage does not support upload length deferring, but we need it.
		if !handler.composer.UsesLengthDeferrer {
			handler.sendError(c, ErrNotImplemented)
			return
		}

		info.SizeIsDeferred = true
	}

	// Parse Content-Type and Content-Disposition to get file type or file name
	if contentType != "" {
		fileType, _, err := mime.ParseMediaType(contentType)
		if err != nil {
			handler.sendError(c, err)
			return
		}

		info.MetaData["filetype"] = fileType
	}

	if contentDisposition != "" {
		_, values, err := mime.ParseMediaType(contentDisposition)
		if err != nil {
			handler.sendError(c, err)
			return
		}

		if values["filename"] != "" {
			info.MetaData["filename"] = values["filename"]
		}
	}

	resp := HTTPResponse{
		StatusCode: http.StatusCreated,
		Header:     HTTPHeader{},
	}

	// 1. Create upload resource
	if handler.config.PreUploadCreateCallback != nil {
		resp2, changes, err := handler.config.PreUploadCreateCallback(newHookEvent(c, info))
		if err != nil {
			handler.sendError(c, err)
			return
		}
		resp = resp.MergeWith(resp2)

		// Apply changes returned from the pre-create hook.
		if changes.ID != "" {
			if err := validateUploadId(changes.ID); err != nil {
				handler.sendError(c, err)
				return
			}

			info.ID = changes.ID
		}

		if changes.MetaData != nil {
			info.MetaData = changes.MetaData
		}

		if changes.Storage != nil {
			info.Storage = changes.Storage
		}
	}

	upload, err := handler.composer.Core.NewUpload(c, info)
	if err != nil {
		handler.sendError(c, err)
		return
	}

	info, err = upload.GetInfo(c)
	if err != nil {
		handler.sendError(c, err)
		return
	}

	id := info.ID
	url := handler.absFileURL(r, id)
	limits := handler.getIETFDraftUploadLimits(info)
	resp.Header["Location"] = url
	resp.Header["Upload-Limit"] = limits

	// Send 104 response
	w.Header().Set("Location", url)
	w.Header().Set("Upload-Draft-Interop-Version", string(currentUploadDraftInteropVersion))
	w.Header().Set("Upload-Limit", limits)
	w.WriteHeader(104)

	handler.Metrics.incUploadsCreated()
	c.log = c.log.With("id", id)
	c.log.InfoContext(c, "UploadCreated", "size", info.Size, "url", url)

	if handler.config.NotifyCreatedUploads {
		handler.CreatedUploads <- newHookEvent(c, info)
	}

	// 2. Lock upload
	if handler.composer.UsesLocker {
		lock, err := handler.lockUpload(c, id)
		if err != nil {
			handler.sendError(c, err)
			return
		}

		defer lock.Unlock()
	}

	// 3. Write chunk
	resp, err = handler.writeChunk(c, resp, upload, info)
	if err != nil {
		handler.sendError(c, err)
		return
	}

	// 4. Finish upload, if necessary
	if willCompleteUpload && info.SizeIsDeferred {
		info, err = upload.GetInfo(c)
		if err != nil {
			handler.sendError(c, err)
			return
		}

		uploadLength := info.Offset

		lengthDeclarableUpload := handler.composer.LengthDeferrer.AsLengthDeclarableUpload(upload)
		if err := lengthDeclarableUpload.DeclareLength(c, uploadLength); err != nil {
			handler.sendError(c, err)
			return
		}

		info.Size = uploadLength
		info.SizeIsDeferred = false

		resp, err = handler.finishUploadIfComplete(c, resp, upload, info)
		if err != nil {
			handler.sendError(c, err)
			return
		}

	}

	handler.sendResp(c, resp)
}

// HeadFile returns the length and offset for the HEAD request
func (handler *UnroutedHandler) HeadFile(w http.ResponseWriter, r *http.Request) {
	c := handler.getContext(w, r)

	id, err := extractIDFromPath(r.URL.Path)
	if err != nil {
		handler.sendError(c, err)
		return
	}
	c.log = c.log.With("id", id)

	if handler.composer.UsesLocker {
		lock, err := handler.lockUpload(c, id)
		if err != nil {
			handler.sendError(c, err)
			return
		}

		defer lock.Unlock()
	}

	upload, err := handler.composer.Core.GetUpload(c, id)
	if err != nil {
		handler.sendError(c, err)
		return
	}

	info, err := upload.GetInfo(c)
	if err != nil {
		handler.sendError(c, err)
		return
	}

	resp := HTTPResponse{
		Header: HTTPHeader{
			"Cache-Control": "no-store",
			"Upload-Offset": strconv.FormatInt(info.Offset, 10),
		},
	}

	if !handler.usesIETFDraft(r) {
		// Add Upload-Concat header if possible
		if info.IsPartial {
			resp.Header["Upload-Concat"] = "partial"
		}

		if info.IsFinal {
			v := "final;"
			for _, uploadID := range info.PartialUploads {
				v += handler.absFileURL(r, uploadID) + " "
			}
			// Remove trailing space
			v = v[:len(v)-1]

			resp.Header["Upload-Concat"] = v
		}

		if len(info.MetaData) != 0 {
			resp.Header["Upload-Metadata"] = SerializeMetadataHeader(info.MetaData)
		}

		if info.SizeIsDeferred {
			resp.Header["Upload-Defer-Length"] = UploadLengthDeferred
		} else {
			resp.Header["Upload-Length"] = strconv.FormatInt(info.Size, 10)
			// TODO: Shouldn't this rather be offset? Basically, whatever GET would return.
			// But this then also depends on the storage backend if that's even supported.
			resp.Header["Content-Length"] = strconv.FormatInt(info.Size, 10)
		}

		resp.StatusCode = http.StatusOK
	} else {
		isUploadCompleteNow := !info.SizeIsDeferred && info.Offset == info.Size
		setIETFDraftUploadComplete(r, resp, isUploadCompleteNow)
		resp.Header["Upload-Draft-Interop-Version"] = string(getIETFDraftInteropVersion(r))

		if !info.SizeIsDeferred {
			resp.Header["Upload-Length"] = strconv.FormatInt(info.Size, 10)
		}

		resp.Header["Upload-Limit"] = handler.getIETFDraftUploadLimits(info)

		// Draft -01 and -02 require a 204 No Content response. Version -03 allows 200 OK as well,
		// but we stick to 204 to not make the logic less complex.
		resp.StatusCode = http.StatusNoContent
	}

	handler.sendResp(c, resp)
}

// PatchFile adds a chunk to an upload. This operation is only allowed
// if enough space in the upload is left.
func (handler *UnroutedHandler) PatchFile(w http.ResponseWriter, r *http.Request) {
	c := handler.getContext(w, r)

	isTusV1 := !handler.usesIETFDraft(r)

	// Check for presence of application/offset+octet-stream (tus v1) or application/partial-upload (IETF draft since -04)
	if isTusV1 && r.Header.Get("Content-Type") != "application/offset+octet-stream" {
		handler.sendError(c, ErrInvalidContentType)
		return
	}

	if !isTusV1 {
		currentInteropVersion := getIETFDraftInteropVersion(r)
		if currentInteropVersion != interopVersion3 && currentInteropVersion != interopVersion4 && currentInteropVersion != interopVersion5 && r.Header.Get("Content-Type") != "application/partial-upload" {
			handler.sendError(c, ErrInvalidContentType)
			return
		}
	}

	// Check for presence of a valid Upload-Offset Header
	offset, err := strconv.ParseInt(r.Header.Get("Upload-Offset"), 10, 64)
	if err != nil || offset < 0 {
		handler.sendError(c, ErrInvalidOffset)
		return
	}

	id, err := extractIDFromPath(r.URL.Path)
	if err != nil {
		handler.sendError(c, err)
		return
	}
	c.log = c.log.With("id", id)

	if handler.composer.UsesLocker {
		lock, err := handler.lockUpload(c, id)
		if err != nil {
			handler.sendError(c, err)
			return
		}

		defer lock.Unlock()
	}

	upload, err := handler.composer.Core.GetUpload(c, id)
	if err != nil {
		handler.sendError(c, err)
		return
	}

	info, err := upload.GetInfo(c)
	if err != nil {
		handler.sendError(c, err)
		return
	}

	// Modifying a final upload is not allowed
	if info.IsFinal {
		handler.sendError(c, ErrModifyFinal)
		return
	}

	if offset != info.Offset {
		handler.sendError(c, ErrMismatchOffset)
		return
	}

	// TODO: If (Upload-Incomplete: ?0 OR Upload-Complete: ?1) and (Content-Length is set), we can
	// - declare the length already here
	// - validate that the length from this request matches info.Size if !info.SizeIsDeferred

	resp := HTTPResponse{
		StatusCode: http.StatusNoContent,
		Header:     make(HTTPHeader, 1), // Initialize map, so writeChunk can set the Upload-Offset header.
	}

	// Do not proxy the call to the data store if the upload is already completed
	if !info.SizeIsDeferred && info.Offset == info.Size {
		resp.Header["Upload-Offset"] = strconv.FormatInt(offset, 10)
		handler.sendResp(c, resp)
		return
	}

	if r.Header.Get("Upload-Length") != "" {
		if !handler.composer.UsesLengthDeferrer {
			handler.sendError(c, ErrNotImplemented)
			return
		}
		if !info.SizeIsDeferred {
			handler.sendError(c, ErrInvalidUploadLength)
			return
		}
		uploadLength, err := strconv.ParseInt(r.Header.Get("Upload-Length"), 10, 64)
		if err != nil || uploadLength < 0 || uploadLength < info.Offset || (handler.config.MaxSize > 0 && uploadLength > handler.config.MaxSize) {
			handler.sendError(c, ErrInvalidUploadLength)
			return
		}

		lengthDeclarableUpload := handler.composer.LengthDeferrer.AsLengthDeclarableUpload(upload)
		if err := lengthDeclarableUpload.DeclareLength(c, uploadLength); err != nil {
			handler.sendError(c, err)
			return
		}

		info.Size = uploadLength
		info.SizeIsDeferred = false
	}

	resp, err = handler.writeChunk(c, resp, upload, info)
	if err != nil {
		handler.sendError(c, err)
		return
	}

	willCompleteUpload := isIETFDraftUploadComplete(r)
	if willCompleteUpload && info.SizeIsDeferred {
		info, err = upload.GetInfo(c)
		if err != nil {
			handler.sendError(c, err)
			return
		}

		uploadLength := info.Offset

		lengthDeclarableUpload := handler.composer.LengthDeferrer.AsLengthDeclarableUpload(upload)
		if err := lengthDeclarableUpload.DeclareLength(c, uploadLength); err != nil {
			handler.sendError(c, err)
			return
		}

		info.Size = uploadLength
		info.SizeIsDeferred = false

		resp, err = handler.finishUploadIfComplete(c, resp, upload, info)
		if err != nil {
			handler.sendError(c, err)
			return
		}
	}

	handler.sendResp(c, resp)
}

// writeChunk reads the body from the requests r and appends it to the upload
// with the corresponding id. Afterwards, it will set the necessary response
// headers but will not send the response.
func (handler *UnroutedHandler) writeChunk(c *httpContext, resp HTTPResponse, upload Upload, info FileInfo) (HTTPResponse, error) {
	// Get Content-Length if possible
	r := c.req
	length := r.ContentLength
	offset := info.Offset

	// Test if this upload fits into the file's size
	if !info.SizeIsDeferred && offset+length > info.Size {
		return resp, ErrSizeExceeded
	}

	maxSize := info.Size - offset
	// If the upload's length is deferred and the PATCH request does not contain the Content-Length
	// header (which is allowed if 'Transfer-Encoding: chunked' is used), we still need to set limits for
	// the body size.
	if info.SizeIsDeferred {
		if handler.config.MaxSize > 0 {
			// Ensure that the upload does not exceed the maximum upload size
			maxSize = handler.config.MaxSize - offset
		} else {
			// If no upload limit is given, we allow arbitrary sizes
			maxSize = math.MaxInt64
		}
	}
	if length > 0 {
		maxSize = length
	}

	c.log.InfoContext(c, "ChunkWriteStart", "maxSize", maxSize, "offset", offset)

	var bytesWritten int64
	var err error
	// Prevent a nil pointer dereference when accessing the body which may not be
	// available in the case of a malicious request.
	if r.Body != nil {
		// Limit the data read from the request's body to the allowed maximum. We use
		// http.MaxBytesReader instead of io.LimitedReader because it returns an error
		// if too much data is provided (handled in bodyReader) and also stops the server
		// from reading the remaining request body.
		c.body = newBodyReader(c, maxSize)
		c.body.onReadDone = func() {
			// Update the read deadline for every successful read operation. This ensures that the request handler
			// keeps going while data is transmitted but that dead connections can also time out and be cleaned up.
			if err := c.resC.SetReadDeadline(time.Now().Add(handler.config.NetworkTimeout)); err != nil {
				c.log.WarnContext(c, "NetworkTimeoutError", "error", err)
			}

			// The write deadline is updated accordingly to ensure that we can also write responses.
			if err := c.resC.SetWriteDeadline(time.Now().Add(2 * handler.config.NetworkTimeout)); err != nil {
				c.log.WarnContext(c, "NetworkTimeoutError", "error", err)
			}
		}

		// We use a callback to allow the hook system to cancel an upload. The callback
		// cancels the request context causing the request body to be closed with the
		// provided error.
		info.stopUpload = func(res HTTPResponse) {
			cause := ErrUploadStoppedByServer
			cause.HTTPResponse = cause.HTTPResponse.MergeWith(res)
			c.cancel(cause)
		}

		if handler.config.NotifyUploadProgress {
			handler.sendProgressMessages(c, info)
		}

		bytesWritten, err = upload.WriteChunk(c, offset, c.body)

		// If we encountered an error while reading the body from the HTTP request, log it, but only include
		// it in the response, if the store did not also return an error.
		bodyErr := c.body.hasError()
		if bodyErr != nil {
			c.log.ErrorContext(c, "BodyReadError", "error", bodyErr.Error())
			if err == nil {
				err = bodyErr
			}
		}

		// Terminate the upload if it was stopped, as indicated by the ErrUploadStoppedByServer error.
		terminateUpload := errors.Is(bodyErr, ErrUploadStoppedByServer)
		if terminateUpload && handler.composer.UsesTerminater {
			if terminateErr := handler.terminateUpload(c, upload, info); terminateErr != nil {
				// We only log this error and not show it to the user since this
				// termination error is not relevant to the uploading client
				c.log.ErrorContext(c, "UploadStopTerminateError", "error", terminateErr.Error())
			}
		}
	}

	c.log.InfoContext(c, "ChunkWriteComplete", "bytesWritten", bytesWritten)

	// Send new offset to client
	newOffset := offset + bytesWritten
	resp.Header["Upload-Offset"] = strconv.FormatInt(newOffset, 10)
	handler.Metrics.incBytesReceived(uint64(bytesWritten))
	info.Offset = newOffset

	// We try to finish the upload, even if an error occurred. If we have a previous error,
	// we return it and its HTTP response.
	finishResp, finishErr := handler.finishUploadIfComplete(c, resp, upload, info)
	if err != nil {
		return resp, err
	}

	return finishResp, finishErr
}

// finishUploadIfComplete checks whether an upload is completed (i.e. upload offset
// matches upload size) and if so, it will call the data store's FinishUpload
// function and emit the necessary events for the hooks.
func (handler *UnroutedHandler) finishUploadIfComplete(c *httpContext, resp HTTPResponse, upload Upload, info FileInfo) (HTTPResponse, error) {
	// If the upload is completed, ...
	if !info.SizeIsDeferred && info.Offset == info.Size {
		var err error
		// ... allow the data storage to finish and cleanup the upload
		if err = upload.FinishUpload(c); err != nil {
			return resp, err
		}

		// ... and call pre-finish callback and send post-finish notification.
		resp, err = handler.emitFinishEvents(c, resp, info)
		if err != nil {
			return resp, err
		}
	}

	return resp, nil
}

// emitFinishEvents calls the PreFinishResponseCallback function and sends
// the necessary message on the CompleteUpload channel.
func (handler *UnroutedHandler) emitFinishEvents(c *httpContext, resp HTTPResponse, info FileInfo) (HTTPResponse, error) {
	if handler.config.PreFinishResponseCallback != nil {
		resp2, err := handler.config.PreFinishResponseCallback(newHookEvent(c, info))
		if err != nil {
			return resp, err
		}
		resp = resp.MergeWith(resp2)
	}

	c.log.InfoContext(c, "UploadFinished", "size", info.Size)
	handler.Metrics.incUploadsFinished()

	if handler.config.NotifyCompleteUploads {
		handler.CompleteUploads <- newHookEvent(c, info)
	}

	return resp, nil
}

// GetFile handles requests to download a file using a GET request. This is not
// part of the specification.
func (handler *UnroutedHandler) GetFile(w http.ResponseWriter, r *http.Request) {
	c := handler.getContext(w, r)

	id, err := extractIDFromPath(r.URL.Path)
	if err != nil {
		handler.sendError(c, err)
		return
	}
	c.log = c.log.With("id", id)

	if handler.composer.UsesLocker {
		lock, err := handler.lockUpload(c, id)
		if err != nil {
			handler.sendError(c, err)
			return
		}

		defer lock.Unlock()
	}

	upload, err := handler.composer.Core.GetUpload(c, id)
	if err != nil {
		handler.sendError(c, err)
		return
	}

	info, err := upload.GetInfo(c)
	if err != nil {
		handler.sendError(c, err)
		return
	}

	// Fall back to the existing GetReader implementation if ContentServerDataStore is not implemented
	contentType, contentDisposition := filterContentType(info)
	resp := HTTPResponse{
		StatusCode: http.StatusOK,
		Header: HTTPHeader{
			"Content-Length":      strconv.FormatInt(info.Offset, 10),
			"Content-Type":        contentType,
			"Content-Disposition": contentDisposition,
		},
		Body: "", // Body is intentionally left empty, and we copy it manually in later.
	}

	// If the data store implements ContentServerDataStore, use delegate the handling
	// of GET requests to the data store.
	// Otherwise, we will use the existing GetReader implementation.
	if handler.composer.UsesContentServer {
		servableUpload := handler.composer.ContentServer.AsServableUpload(upload)

		// Pass file type and name to the implementation, but it may override them.
		w.Header().Set("Content-Type", resp.Header["Content-Type"])
		w.Header().Set("Content-Disposition", resp.Header["Content-Disposition"])

		// Use loggingResponseWriter to get the ResponseOutgoing log entry that
		// normally handler.sendResp would produce.
		loggingW := &loggingResponseWriter{ResponseWriter: w, logger: c.log}

		err = servableUpload.ServeContent(c, loggingW, r)
		if err != nil {
			handler.sendError(c, err)
		}
		return
	}

	// If no data has been uploaded yet, respond with an empty "204 No Content" status.
	if info.Offset == 0 {
		resp.StatusCode = http.StatusNoContent
		handler.sendResp(c, resp)
		return
	}

	if handler.composer.UsesContentServer {
		servableUpload := handler.composer.ContentServer.AsServableUpload(upload)
		err = servableUpload.ServeContent(c, w, r)
		if err != nil {
			handler.sendError(c, err)
		}
		return
	}

	src, err := upload.GetReader(c)
	if err != nil {
		handler.sendError(c, err)
		return
	}

	handler.sendResp(c, resp)
	if _, err := io.Copy(w, src); err != nil {
		handler.sendError(c, err)
		return
	}

	src.Close()
}

// mimeInlineBrowserWhitelist is a map containing MIME types which should be
// allowed to be rendered by browser inline, instead of being forced to be
// downloaded. For example, HTML or SVG files are not allowed, since they may
// contain malicious JavaScript. In a similar fashion, PDF is not on this list
// as their parsers commonly contain vulnerabilities which can be exploited.
// The values of this map do not convey any meaning and are therefore just
// empty structs.
var mimeInlineBrowserWhitelist = map[string]struct{}{
	"text/plain": {},

	"image/png":  {},
	"image/jpeg": {},
	"image/gif":  {},
	"image/bmp":  {},
	"image/webp": {},

	"audio/wave":     {},
	"audio/wav":      {},
	"audio/x-wav":    {},
	"audio/x-pn-wav": {},
	"audio/webm":     {},
	"audio/ogg":      {},

	"video/mp4":  {},
	"video/webm": {},
	"video/ogg":  {},

	"application/ogg": {},
}

// filterContentType returns the values for the Content-Type and
// Content-Disposition headers for a given upload. These values should be used
// in responses for GET requests to ensure that only non-malicious file types
// are shown directly in the browser. It will extract the file name and type
// from the "filename" and "filetype".
// See https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Content-Disposition
func filterContentType(info FileInfo) (contentType string, contentDisposition string) {
	filetype := info.MetaData["filetype"]

	if ft, _, err := mime.ParseMediaType(filetype); err == nil {
		// If the filetype from metadata is well-formed, we forward use this for the Content-Type header.
		// However, only allowlisted mime types	will be allowed to be shown inline in the browser
		contentType = filetype
		if _, isWhitelisted := mimeInlineBrowserWhitelist[ft]; isWhitelisted {
			contentDisposition = "inline"
		} else {
			contentDisposition = "attachment"
		}
	} else {
		// If the filetype from the metadata is not well-formed, we use a
		// default type and force the browser to download the content.
		contentType = "application/octet-stream"
		contentDisposition = "attachment"
	}

	// Add a filename to Content-Disposition if one is available in the metadata
	if filename, ok := info.MetaData["filename"]; ok {
		contentDisposition += ";filename=" + strconv.Quote(filename)
	}

	return contentType, contentDisposition
}

// DelFile terminates an upload permanently.
func (handler *UnroutedHandler) DelFile(w http.ResponseWriter, r *http.Request) {
	c := handler.getContext(w, r)

	// Abort the request handling if the required interface is not implemented
	if !handler.composer.UsesTerminater {
		handler.sendError(c, ErrNotImplemented)
		return
	}

	id, err := extractIDFromPath(r.URL.Path)
	if err != nil {
		handler.sendError(c, err)
		return
	}
	c.log = c.log.With("id", id)

	if handler.composer.UsesLocker {
		lock, err := handler.lockUpload(c, id)
		if err != nil {
			handler.sendError(c, err)
			return
		}

		defer lock.Unlock()
	}

	upload, err := handler.composer.Core.GetUpload(c, id)
	if err != nil {
		handler.sendError(c, err)
		return
	}

	var info FileInfo
	if handler.config.NotifyTerminatedUploads || handler.config.PreUploadTerminateCallback != nil {
		info, err = upload.GetInfo(c)
		if err != nil {
			handler.sendError(c, err)
			return
		}
	}

	resp := HTTPResponse{
		StatusCode: http.StatusNoContent,
	}

	if handler.config.PreUploadTerminateCallback != nil {
		resp2, err := handler.config.PreUploadTerminateCallback(newHookEvent(c, info))
		if err != nil {
			handler.sendError(c, err)
			return
		}
		resp = resp.MergeWith(resp2)
	}

	err = handler.terminateUpload(c, upload, info)
	if err != nil {
		handler.sendError(c, err)
		return
	}

	handler.sendResp(c, resp)
}

// terminateUpload passes a given upload to the DataStore's Terminater,
// send the corresponding upload info on the TerminatedUploads channnel
// and updates the statistics.
// Note the the info argument is only needed if the terminated uploads
// notifications are enabled.
func (handler *UnroutedHandler) terminateUpload(c *httpContext, upload Upload, info FileInfo) error {
	terminatableUpload := handler.composer.Terminater.AsTerminatableUpload(upload)

	err := terminatableUpload.Terminate(c)
	if err != nil {
		return err
	}

	if handler.config.NotifyTerminatedUploads {
		handler.TerminatedUploads <- newHookEvent(c, info)
	}

	c.log.InfoContext(c, "UploadTerminated")
	handler.Metrics.incUploadsTerminated()

	return nil
}

// Send the error in the response body. The status code will be looked up in
// ErrStatusCodes. If none is found 500 Internal Error will be used.
func (handler *UnroutedHandler) sendError(c *httpContext, err error) {
	r := c.req

	var detailedErr Error

	if !errors.As(err, &detailedErr) {
		c.log.ErrorContext(c, "InternalServerError", "message", err.Error())
		detailedErr = NewError("ERR_INTERNAL_SERVER_ERROR", err.Error(), http.StatusInternalServerError)
	}

	// If we are sending the response for a HEAD request, ensure that we are not including
	// any response body.
	if r.Method == "HEAD" {
		detailedErr.HTTPResponse.Body = ""
	}

	handler.sendResp(c, detailedErr.HTTPResponse)
	handler.Metrics.incErrorsTotal(detailedErr)
}

// sendResp writes the header to w with the specified status code.
func (handler *UnroutedHandler) sendResp(c *httpContext, resp HTTPResponse) {
	resp.writeTo(c.res)

	c.log.InfoContext(c, "ResponseOutgoing", "status", resp.StatusCode, "body", resp.Body)
}

// Make an absolute URLs to the given upload id. If the base path is absolute
// it will be prepended else the host and protocol from the request is used.
func (handler *UnroutedHandler) absFileURL(r *http.Request, id string) string {
	if handler.isBasePathAbs {
		return handler.basePath + id
	}

	// Read origin and protocol from request
	host, proto := getHostAndProtocol(r, handler.config.RespectForwardedHeaders)

	url := proto + "://" + host + handler.basePath + id

	return url
}

// sendProgressMessage will send a notification over the UploadProgress channel
// indicating how much data has been transfered to the server.
// It will stop sending these instances once the provided context is done.
func (handler *UnroutedHandler) sendProgressMessages(c *httpContext, info FileInfo) {
	hook := newHookEvent(c, info)

	previousOffset := int64(0)
	originalOffset := hook.Upload.Offset

	emitProgress := func() {
		hook.Upload.Offset = originalOffset + c.body.bytesRead()
		if hook.Upload.Offset != previousOffset {
			handler.UploadProgress <- hook
			previousOffset = hook.Upload.Offset
		}
	}

	go func() {
		for {
			select {
			case <-c.Done():
				emitProgress()
				return
			case <-time.After(handler.config.UploadProgressInterval):
				emitProgress()
			}
		}
	}()
}

// getHostAndProtocol extracts the host and used protocol (either HTTP or HTTPS)
// from the given request. If `allowForwarded` is set, the X-Forwarded-Host,
// X-Forwarded-Proto and Forwarded headers will also be checked to
// support proxies.
func getHostAndProtocol(r *http.Request, allowForwarded bool) (host, proto string) {
	if r.TLS != nil {
		proto = "https"
	} else {
		proto = "http"
	}

	host = r.Host

	if !allowForwarded {
		return
	}

	if h := r.Header.Get("X-Forwarded-Host"); h != "" {
		host = h
	}

	if h := r.Header.Get("X-Forwarded-Proto"); h == "http" || h == "https" {
		proto = h
	}

	if h := r.Header.Get("Forwarded"); h != "" {
		if r := reForwardedHost.FindStringSubmatch(h); len(r) == 2 {
			host = r[1]
		}

		if r := reForwardedProto.FindStringSubmatch(h); len(r) == 2 {
			proto = r[1]
		}
	}

	// Remove default ports
	if proto == "http" {
		host = strings.TrimSuffix(host, ":80")
	}
	if proto == "https" {
		host = strings.TrimSuffix(host, ":443")
	}

	return
}

// The get sum of all sizes for a list of upload ids while checking whether
// all of these uploads are finished yet. This is used to calculate the size
// of a final resource.
func (handler *UnroutedHandler) sizeOfUploads(ctx context.Context, ids []string) (partialUploads []Upload, size int64, err error) {
	partialUploads = make([]Upload, len(ids))

	for i, id := range ids {
		upload, err := handler.composer.Core.GetUpload(ctx, id)
		if err != nil {
			return nil, 0, err
		}

		info, err := upload.GetInfo(ctx)
		if err != nil {
			return nil, 0, err
		}

		if info.SizeIsDeferred || info.Offset != info.Size {
			err = ErrUploadNotFinished
			return nil, 0, err
		}

		size += info.Size
		partialUploads[i] = upload
	}

	return
}

// Verify that the Upload-Length and Upload-Defer-Length headers are acceptable for creating a
// new upload
func (handler *UnroutedHandler) validateNewUploadLengthHeaders(uploadLengthHeader string, uploadDeferLengthHeader string) (uploadLength int64, uploadLengthDeferred bool, err error) {
	haveBothLengthHeaders := uploadLengthHeader != "" && uploadDeferLengthHeader != ""
	haveInvalidDeferHeader := uploadDeferLengthHeader != "" && uploadDeferLengthHeader != UploadLengthDeferred
	lengthIsDeferred := uploadDeferLengthHeader == UploadLengthDeferred

	if lengthIsDeferred && !handler.composer.UsesLengthDeferrer {
		err = ErrNotImplemented
	} else if haveBothLengthHeaders {
		err = ErrUploadLengthAndUploadDeferLength
	} else if haveInvalidDeferHeader {
		err = ErrInvalidUploadDeferLength
	} else if lengthIsDeferred {
		uploadLengthDeferred = true
	} else {
		uploadLength, err = strconv.ParseInt(uploadLengthHeader, 10, 64)
		if err != nil || uploadLength < 0 {
			err = ErrInvalidUploadLength
		}
	}

	return
}

// lockUpload creates a new lock for the given upload ID and attempts to lock it.
// The created lock is returned if it was aquired successfully.
func (handler *UnroutedHandler) lockUpload(c *httpContext, id string) (Lock, error) {
	lock, err := handler.composer.Locker.NewLock(id)
	if err != nil {
		return nil, err
	}

	ctx, cancelContext := context.WithTimeout(c, handler.config.AcquireLockTimeout)
	defer cancelContext()

	// No need to wrap this in a sync.OnceFunc because c.cancel will be a noop after the first call.
	releaseLock := func() {
		c.log.InfoContext(c, "UploadInterrupted")
		c.cancel(ErrUploadInterrupted)
	}

	if err := lock.Lock(ctx, releaseLock); err != nil {
		return nil, err
	}

	return lock, nil
}

// usesIETFDraft returns whether a HTTP request uses a supported version of the resumable upload draft from IETF
// (instead of tus v1) and support has been enabled in tusd.
func (handler UnroutedHandler) usesIETFDraft(r *http.Request) bool {
	interopVersionHeader := getIETFDraftInteropVersion(r)
	return handler.config.EnableExperimentalProtocol && interopVersionHeader != ""
}

// getIETFDraftUploadLimits returns the Upload-Limit header for a given upload
// according to the set resumable upload draft version from IETF.
func (handler UnroutedHandler) getIETFDraftUploadLimits(info FileInfo) string {
	limits := "min-size=0"
	if handler.config.MaxSize > 0 {
		limits += ",max-size=" + strconv.FormatInt(handler.config.MaxSize, 10)
	} else if !info.SizeIsDeferred {
		limits += ",max-size=" + strconv.FormatInt(info.Size, 10)
	}

	return limits
}

// getIETFDraftInteropVersion returns the resumable upload draft interop version from the headers.
func getIETFDraftInteropVersion(r *http.Request) draftVersion {
	version := draftVersion(r.Header.Get("Upload-Draft-Interop-Version"))
	switch version {
	case interopVersion3, interopVersion4, interopVersion5, interopVersion6:
		return version
	default:
		return ""
	}
}

// isIETFDraftUploadComplete returns whether a HTTP request upload is complete
// according to the set resumable upload draft version from IETF.
func isIETFDraftUploadComplete(r *http.Request) bool {
	currentUploadDraftInteropVersion := getIETFDraftInteropVersion(r)
	switch currentUploadDraftInteropVersion {
	case interopVersion4, interopVersion5, interopVersion6:
		return r.Header.Get("Upload-Complete") == "?1"
	case interopVersion3:
		return r.Header.Get("Upload-Incomplete") == "?0"
	default:
		return false
	}
}

// setIETFDraftUploadComplete sets the Upload-Complete (Upload-Incomplete) to the provided
// value, depending on the interop version used in the request.
func setIETFDraftUploadComplete(r *http.Request, resp HTTPResponse, isComplete bool) {
	currentUploadDraftInteropVersion := getIETFDraftInteropVersion(r)

	switch currentUploadDraftInteropVersion {
	case interopVersion3:
		if isComplete {
			resp.Header["Upload-Incomplete"] = "?0"
		} else {
			resp.Header["Upload-Incomplete"] = "?1"
		}
	case interopVersion4, interopVersion5, interopVersion6:
		if isComplete {
			resp.Header["Upload-Complete"] = "?1"
		} else {
			resp.Header["Upload-Complete"] = "?0"
		}
	}
}

// getIETFDraftUploadLength returns the length of an upload as defined in the
// resumable upload draft from IETF. This can either be in the Upload-Length
// header or in the Content-Length header.
func getIETFDraftUploadLength(r *http.Request) (length int64, lengthIsDeferred bool, err error) {
	var lengthFromUploadLength int64
	hasLengthFromUploadLength := false
	var lengthFromContentLength int64
	hasLengthFromContentLength := false

	willCompleteUpload := isIETFDraftUploadComplete(r)
	if willCompleteUpload && r.ContentLength != -1 {
		lengthFromContentLength = r.ContentLength
		hasLengthFromContentLength = true
	}

	uploadLengthStr := r.Header.Get("Upload-Length")
	if uploadLengthStr != "" {
		var err error
		lengthFromUploadLength, err = strconv.ParseInt(uploadLengthStr, 10, 64)
		if err != nil {
			return 0, false, ErrInvalidUploadLength
		}

		hasLengthFromUploadLength = true
	}

	// If both lengths are set, they must match
	if hasLengthFromContentLength && hasLengthFromUploadLength && lengthFromUploadLength != lengthFromContentLength {
		return 0, false, ErrInvalidUploadLength
	}

	// Return whichever length is set
	if hasLengthFromUploadLength {
		return lengthFromUploadLength, false, nil
	}
	if hasLengthFromContentLength {
		return lengthFromContentLength, false, nil
	}

	// No length set, so it's deferred
	return 0, true, nil
}

// ParseMetadataHeader parses the Upload-Metadata header as defined in the
// File Creation extension.
// e.g. Upload-Metadata: name bHVucmpzLnBuZw==,type aW1hZ2UvcG5n
func ParseMetadataHeader(header string) map[string]string {
	meta := make(map[string]string)

	for element := range strings.SplitSeq(header, ",") {
		element := strings.TrimSpace(element)

		parts := strings.Split(element, " ")

		if len(parts) > 2 {
			continue
		}

		key := parts[0]
		if key == "" {
			continue
		}

		value := ""
		if len(parts) == 2 {
			// Ignore current element if the value is no valid base64
			dec, err := base64.StdEncoding.DecodeString(parts[1])
			if err != nil {
				continue
			}

			value = string(dec)
		}

		meta[key] = value
	}

	return meta
}

// SerializeMetadataHeader serializes a map of strings into the Upload-Metadata
// header format used in the response for HEAD requests.
// e.g. Upload-Metadata: name bHVucmpzLnBuZw==,type aW1hZ2UvcG5n
func SerializeMetadataHeader(meta map[string]string) string {
	header := ""
	for key, value := range meta {
		valueBase64 := base64.StdEncoding.EncodeToString([]byte(value))
		header += key + " " + valueBase64 + ","
	}

	// Remove trailing comma
	if len(header) > 0 {
		header = header[:len(header)-1]
	}

	return header
}

// Parse the Upload-Concat header, e.g.
// Upload-Concat: partial
// Upload-Concat: final;http://tus.io/files/a /files/b/
func parseConcat(header string, basePath string) (isPartial bool, isFinal bool, partialUploads []string, err error) {
	if len(header) == 0 {
		return
	}

	if header == "partial" {
		isPartial = true
		return
	}

	l := len("final;")
	if strings.HasPrefix(header, "final;") && len(header) > l {
		isFinal = true

		list := strings.SplitSeq(header[l:], " ")
		for value := range list {
			value := strings.TrimSpace(value)
			if value == "" {
				continue
			}

			id, extractErr := extractIDFromURL(value, basePath)
			if extractErr != nil {
				err = extractErr
				return
			}

			partialUploads = append(partialUploads, id)
		}
	}

	// If no valid partial upload ids are extracted this is not a final upload.
	if len(partialUploads) == 0 {
		isFinal = false
		err = ErrInvalidConcat
	}

	return
}

// extractIDFromPath extracts the upload ID from a path, which has already
// been stripped of the base path (done by the user). Effectively, we only
// remove leading and trailing slashes.
func extractIDFromPath(path string) (string, error) {
	return strings.Trim(path, "/"), nil
}

// extractIDFromURL extracts the upload ID from a full URL or a full path
// (including the base path). For example:
//
//	https://example.com/files/1234/5678 -> 1234/5678
//	/files/1234/5678 -> 1234/5678
func extractIDFromURL(url string, basePath string) (string, error) {
	_, id, ok := strings.Cut(url, basePath)
	if !ok {
		return "", ErrNotFound
	}

	return extractIDFromPath(id)
}

// getRequestId returns the value of the X-Request-ID header, if available,
// and also takes care of truncating the input.
func getRequestId(r *http.Request) string {
	reqId := r.Header.Get("X-Request-ID")
	if reqId == "" {
		return ""
	}

	// Limit the length of the request ID to 36 characters, which is enough
	// to fit a UUID.
	if len(reqId) > 36 {
		reqId = reqId[:36]
	}

	return reqId
}

// validateUploadId checks whether an ID included in a FileInfoChanges struct is allowed.
func validateUploadId(newId string) error {
	if newId == "" {
		// An empty ID from FileInfoChanges is allowed. The store will then
		// just pick an ID.
		return nil
	}

	if strings.HasPrefix(newId, "/") || strings.HasSuffix(newId, "/") {
		// Disallow leading and trailing slashes, as these would be
		// stripped away by extractIDFromPath, which can cause problems and confusion.
		return fmt.Errorf("validation error in FileInfoChanges: ID must not begin or end with a forward slash (got: %s)", newId)
	}

	if !reValidUploadId.MatchString(newId) {
		// Disallow some non-URL-safe characters in the upload ID to
		// prevent issues with URL parsing, which are though to debug for users.
		return fmt.Errorf("validation error in FileInfoChanges: ID must contain only URL-safe character: %s (got: %s)", reValidUploadId.String(), newId)
	}

	return nil
}

// loggingResponseWriter is a wrapper around http.ResponseWriter that logs the
// final status code similar to UnroutedHandler.sendResp.
type loggingResponseWriter struct {
	http.ResponseWriter
	logger *slog.Logger
}

func (w *loggingResponseWriter) WriteHeader(statusCode int) {
	if statusCode >= 200 {
		w.logger.Info("ResponseOutgoing", "status", statusCode)
	}
	w.ResponseWriter.WriteHeader(statusCode)
}

// Unwrap provides access to the underlying http.ResponseWriter.
func (w *loggingResponseWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }
