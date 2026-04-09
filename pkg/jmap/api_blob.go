package jmap

import (
	"encoding/base64"
	"io"
	"strings"

	"github.com/opencloud-eu/opencloud/pkg/log"
)

var NS_BLOB = ns(JmapBlob)

func (j *Client) GetBlobMetadata(accountId string, id string, ctx Context) (*Blob, SessionState, State, Language, Error) {
	get := BlobGetCommand{
		AccountId: accountId,
		Ids:       []string{id},
		// add BlobPropertyData to retrieve the data
		Properties: []string{BlobPropertyDigestSha256, BlobPropertyDigestSha512, BlobPropertySize},
	}
	cmd, jerr := j.request(ctx, NS_BLOB,
		invocation(get, "0"),
	)
	if jerr != nil {
		return nil, "", "", "", jerr
	}

	return command(j, ctx, cmd, func(body *Response) (*Blob, State, Error) {
		var response BlobGetResponse
		err := retrieveGet(ctx, body, get, "0", &response)
		if err != nil {
			return nil, "", err
		}

		if len(response.List) != 1 {
			ctx.Logger.Error().Msgf("%T.List has %v entries instead of 1", response, len(response.List))
			return nil, "", jmapError(err, JmapErrorInvalidJmapResponsePayload)
		}
		get := response.List[0]
		return &get, response.State, nil
	})
}

type UploadedBlobWithHash struct {
	BlobId string `json:"blobId"`
	Size   int    `json:"size,omitzero"`
	Type   string `json:"type,omitempty"`
	Sha512 string `json:"sha:512,omitempty"`
}

func (j *Client) UploadBlobStream(accountId string, contentType string, body io.Reader, ctx Context) (UploadedBlob, Language, Error) {
	logger := log.From(ctx.Logger.With().Str(logEndpoint, ctx.Session.UploadEndpoint))
	ctx = ctx.WithLogger(logger)
	// TODO(pbleser-oc) use a library for proper URL template parsing
	uploadUrl := strings.ReplaceAll(ctx.Session.UploadUrlTemplate, "{accountId}", accountId)
	return j.blob.UploadBinary(uploadUrl, ctx.Session.UploadEndpoint, contentType, body, ctx)
}

func (j *Client) DownloadBlobStream(accountId string, blobId string, name string, typ string, ctx Context) (*BlobDownload, Language, Error) { //NOSONAR
	logger := log.From(ctx.Logger.With().Str(logEndpoint, ctx.Session.DownloadEndpoint))
	ctx = ctx.WithLogger(logger)
	// TODO(pbleser-oc) use a library for proper URL template parsing
	downloadUrl := ctx.Session.DownloadUrlTemplate
	downloadUrl = strings.ReplaceAll(downloadUrl, "{accountId}", accountId)
	downloadUrl = strings.ReplaceAll(downloadUrl, "{blobId}", blobId)
	downloadUrl = strings.ReplaceAll(downloadUrl, "{name}", name)
	downloadUrl = strings.ReplaceAll(downloadUrl, "{type}", typ)
	logger = log.From(logger.With().Str(logDownloadUrl, downloadUrl).Str(logBlobId, blobId))
	return j.blob.DownloadBinary(downloadUrl, ctx.Session.DownloadEndpoint, ctx)
}

func (j *Client) UploadBlob(accountId string, data []byte, contentType string, ctx Context) (UploadedBlobWithHash, SessionState, State, Language, Error) {
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

	cmd, jerr := j.request(ctx, ns(JmapBlob),
		invocation(upload, "0"),
		invocation(getHash, "1"),
	)
	if jerr != nil {
		return UploadedBlobWithHash{}, "", "", "", jerr
	}

	return command(j, ctx, cmd, func(body *Response) (UploadedBlobWithHash, State, Error) {
		var uploadResponse BlobUploadResponse
		err := retrieveUpload(ctx, body, upload, "0", &uploadResponse)
		if err != nil {
			return UploadedBlobWithHash{}, "", err
		}

		var getResponse BlobGetResponse
		err = retrieveGet(ctx, body, getHash, "1", &getResponse)
		if err != nil {
			return UploadedBlobWithHash{}, "", err
		}

		if len(uploadResponse.Created) != 1 {
			ctx.Logger.Error().Msgf("%T.Created has %v entries instead of 1", uploadResponse, len(uploadResponse.Created))
			return UploadedBlobWithHash{}, "", jmapError(err, JmapErrorInvalidJmapResponsePayload)
		}
		upload, ok := uploadResponse.Created["0"]
		if !ok {
			ctx.Logger.Error().Msgf("%T.Created has no item '0'", uploadResponse)
			return UploadedBlobWithHash{}, "", jmapError(err, JmapErrorInvalidJmapResponsePayload)
		}

		if len(getResponse.List) != 1 {
			ctx.Logger.Error().Msgf("%T.List has %v entries instead of 1", getResponse, len(getResponse.List))
			return UploadedBlobWithHash{}, "", jmapError(err, JmapErrorInvalidJmapResponsePayload)
		}
		get := getResponse.List[0]

		return UploadedBlobWithHash{
			BlobId: upload.Id,
			Size:   upload.Size,
			Type:   upload.Type,
			Sha512: get.DigestSha512,
		}, getResponse.State, nil
	})

}
