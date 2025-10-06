package jmap

import (
	"context"
	"encoding/base64"
	"io"
	"strings"

	"github.com/opencloud-eu/opencloud/pkg/log"
)

type BlobResponse struct {
	Blob  *Blob `json:"blob,omitempty"`
	State State `json:"state,omitempty"`
}

func (j *Client) GetBlobMetadata(accountId string, session *Session, ctx context.Context, logger *log.Logger, acceptLanguage string, id string) (BlobResponse, SessionState, Language, Error) {
	cmd, jerr := j.request(session, logger,
		invocation(CommandBlobGet, BlobGetCommand{
			AccountId: accountId,
			Ids:       []string{id},
			// add BlobPropertyData to retrieve the data
			Properties: []string{BlobPropertyDigestSha256, BlobPropertyDigestSha512, BlobPropertySize},
		}, "0"),
	)
	if jerr != nil {
		return BlobResponse{}, "", "", jerr
	}

	return command(j.api, logger, ctx, session, j.onSessionOutdated, cmd, acceptLanguage, func(body *Response) (BlobResponse, Error) {
		var response BlobGetResponse
		err := retrieveResponseMatchParameters(logger, body, CommandBlobGet, "0", &response)
		if err != nil {
			return BlobResponse{}, err
		}

		if len(response.List) != 1 {
			logger.Error().Msgf("%T.List has %v entries instead of 1", response, len(response.List))
			return BlobResponse{}, simpleError(err, JmapErrorInvalidJmapResponsePayload)
		}
		get := response.List[0]
		return BlobResponse{Blob: &get, State: response.State}, nil
	})
}

type UploadedBlob struct {
	Id     string `json:"id"`
	Size   int    `json:"size"`
	Type   string `json:"type"`
	Sha512 string `json:"sha:512"`
	State  State  `json:"state"`
}

func (j *Client) UploadBlobStream(accountId string, session *Session, ctx context.Context, logger *log.Logger, acceptLanguage string, contentType string, body io.Reader) (UploadedBlob, Language, Error) {
	logger = log.From(logger.With().Str(logEndpoint, session.UploadEndpoint))
	// TODO(pbleser-oc) use a library for proper URL template parsing
	uploadUrl := strings.ReplaceAll(session.UploadUrlTemplate, "{accountId}", accountId)
	return j.blob.UploadBinary(ctx, logger, session, uploadUrl, session.UploadEndpoint, contentType, acceptLanguage, body)
}

func (j *Client) DownloadBlobStream(accountId string, blobId string, name string, typ string, session *Session, ctx context.Context, logger *log.Logger, acceptLanguage string) (*BlobDownload, Language, Error) {
	logger = log.From(logger.With().Str(logEndpoint, session.DownloadEndpoint))
	// TODO(pbleser-oc) use a library for proper URL template parsing
	downloadUrl := session.DownloadUrlTemplate
	downloadUrl = strings.ReplaceAll(downloadUrl, "{accountId}", accountId)
	downloadUrl = strings.ReplaceAll(downloadUrl, "{blobId}", blobId)
	downloadUrl = strings.ReplaceAll(downloadUrl, "{name}", name)
	downloadUrl = strings.ReplaceAll(downloadUrl, "{type}", typ)
	logger = log.From(logger.With().Str(logDownloadUrl, downloadUrl).Str(logBlobId, blobId))
	return j.blob.DownloadBinary(ctx, logger, session, downloadUrl, session.DownloadEndpoint, acceptLanguage)
}

func (j *Client) UploadBlob(accountId string, session *Session, ctx context.Context, logger *log.Logger, acceptLanguage string, data []byte, contentType string) (UploadedBlob, SessionState, Language, Error) {
	encoded := base64.StdEncoding.EncodeToString(data)

	upload := BlobUploadCommand{
		AccountId: accountId,
		Create: map[string]UploadObject{
			"0": {
				Data: []DataSourceObject{{
					DataAsBase64: encoded,
				}},
				Type: contentType,
			},
		},
	}

	getHash := BlobGetRefCommand{
		AccountId: accountId,
		IdRef: &ResultReference{
			ResultOf: "0",
			Name:     CommandBlobUpload,
			Path:     "/ids",
		},
		Properties: []string{BlobPropertyDigestSha512},
	}

	cmd, jerr := j.request(session, logger,
		invocation(CommandBlobUpload, upload, "0"),
		invocation(CommandBlobGet, getHash, "1"),
	)
	if jerr != nil {
		return UploadedBlob{}, "", "", jerr
	}

	return command(j.api, logger, ctx, session, j.onSessionOutdated, cmd, acceptLanguage, func(body *Response) (UploadedBlob, Error) {
		var uploadResponse BlobUploadResponse
		err := retrieveResponseMatchParameters(logger, body, CommandBlobUpload, "0", &uploadResponse)
		if err != nil {
			return UploadedBlob{}, err
		}

		var getResponse BlobGetResponse
		err = retrieveResponseMatchParameters(logger, body, CommandBlobGet, "1", &getResponse)
		if err != nil {
			return UploadedBlob{}, err
		}

		if len(uploadResponse.Created) != 1 {
			logger.Error().Msgf("%T.Created has %v entries instead of 1", uploadResponse, len(uploadResponse.Created))
			return UploadedBlob{}, simpleError(err, JmapErrorInvalidJmapResponsePayload)
		}
		upload, ok := uploadResponse.Created["0"]
		if !ok {
			logger.Error().Msgf("%T.Created has no item '0'", uploadResponse)
			return UploadedBlob{}, simpleError(err, JmapErrorInvalidJmapResponsePayload)
		}

		if len(getResponse.List) != 1 {
			logger.Error().Msgf("%T.List has %v entries instead of 1", getResponse, len(getResponse.List))
			return UploadedBlob{}, simpleError(err, JmapErrorInvalidJmapResponsePayload)
		}
		get := getResponse.List[0]

		return UploadedBlob{
			Id:     upload.Id,
			Size:   upload.Size,
			Type:   upload.Type,
			Sha512: get.DigestSha512,
			State:  getResponse.State,
		}, nil
	})

}
