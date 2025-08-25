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
	State State `json:"state"`
}

func (j *Client) GetBlob(accountId string, session *Session, ctx context.Context, logger *log.Logger, id string) (BlobResponse, SessionState, Error) {
	aid := session.BlobAccountId(accountId)

	cmd, err := request(
		invocation(CommandBlobUpload, BlobGetCommand{
			AccountId:  aid,
			Ids:        []string{id},
			Properties: []string{BlobPropertyData, BlobPropertyDigestSha512, BlobPropertySize},
		}, "0"),
	)
	if err != nil {
		logger.Error().Err(err)
		return BlobResponse{}, "", simpleError(err, JmapErrorInvalidJmapRequestPayload)
	}

	return command(j.api, logger, ctx, session, j.onSessionOutdated, cmd, func(body *Response) (BlobResponse, Error) {
		var response BlobGetResponse
		err = retrieveResponseMatchParameters(body, CommandBlobGet, "0", &response)
		if err != nil {
			logger.Error().Err(err)
			return BlobResponse{}, simpleError(err, JmapErrorInvalidJmapResponsePayload)
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

func (j *Client) UploadBlobStream(accountId string, session *Session, ctx context.Context, logger *log.Logger, contentType string, body io.Reader) (UploadedBlob, Error) {
	aid := session.BlobAccountId(accountId)
	// TODO(pbleser-oc) use a library for proper URL template parsing
	uploadUrl := strings.ReplaceAll(session.UploadUrlTemplate, "{accountId}", aid)
	return j.blob.UploadBinary(ctx, logger, session, uploadUrl, contentType, body)
}

func (j *Client) DownloadBlobStream(accountId string, blobId string, name string, typ string, session *Session, ctx context.Context, logger *log.Logger) (*BlobDownload, Error) {
	aid := session.BlobAccountId(accountId)
	// TODO(pbleser-oc) use a library for proper URL template parsing
	downloadUrl := session.DownloadUrlTemplate
	downloadUrl = strings.ReplaceAll(downloadUrl, "{accountId}", aid)
	downloadUrl = strings.ReplaceAll(downloadUrl, "{blobId}", blobId)
	downloadUrl = strings.ReplaceAll(downloadUrl, "{name}", name)
	downloadUrl = strings.ReplaceAll(downloadUrl, "{type}", typ)
	logger = log.From(logger.With().Str(logDownloadUrl, downloadUrl).Str(logBlobId, blobId).Str(logAccountId, aid))
	return j.blob.DownloadBinary(ctx, logger, session, downloadUrl)
}

func (j *Client) UploadBlob(accountId string, session *Session, ctx context.Context, logger *log.Logger, data []byte, contentType string) (UploadedBlob, SessionState, Error) {
	aid := session.MailAccountId(accountId)

	encoded := base64.StdEncoding.EncodeToString(data)

	upload := BlobUploadCommand{
		AccountId: aid,
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
		AccountId: aid,
		IdRef: &ResultReference{
			ResultOf: "0",
			Name:     CommandBlobUpload,
			Path:     "/ids",
		},
		Properties: []string{BlobPropertyDigestSha512},
	}

	cmd, err := request(
		invocation(CommandBlobUpload, upload, "0"),
		invocation(CommandBlobGet, getHash, "1"),
	)
	if err != nil {
		logger.Error().Err(err)
		return UploadedBlob{}, "", simpleError(err, JmapErrorInvalidJmapRequestPayload)
	}

	return command(j.api, logger, ctx, session, j.onSessionOutdated, cmd, func(body *Response) (UploadedBlob, Error) {
		var uploadResponse BlobUploadResponse
		err = retrieveResponseMatchParameters(body, CommandBlobUpload, "0", &uploadResponse)
		if err != nil {
			logger.Error().Err(err)
			return UploadedBlob{}, simpleError(err, JmapErrorInvalidJmapResponsePayload)
		}

		var getResponse BlobGetResponse
		err = retrieveResponseMatchParameters(body, CommandBlobGet, "1", &getResponse)
		if err != nil {
			logger.Error().Err(err)
			return UploadedBlob{}, simpleError(err, JmapErrorInvalidJmapResponsePayload)
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
