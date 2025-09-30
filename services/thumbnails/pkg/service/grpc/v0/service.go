package svc

import (
	"context"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/golang-jwt/jwt/v5"
	revactx "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	"github.com/opencloud-eu/reva/v2/pkg/storagespace"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
	"github.com/pkg/errors"
	merrors "go-micro.dev/v4/errors"
	"google.golang.org/grpc/metadata"

	"github.com/opencloud-eu/opencloud/pkg/log"
	thumbnailssvc "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/thumbnails/v0"
	terrors "github.com/opencloud-eu/opencloud/services/thumbnails/pkg/errors"
	"github.com/opencloud-eu/opencloud/services/thumbnails/pkg/preprocessor"
	"github.com/opencloud-eu/opencloud/services/thumbnails/pkg/service/grpc/v0/decorators"
	tjwt "github.com/opencloud-eu/opencloud/services/thumbnails/pkg/service/jwt"
	"github.com/opencloud-eu/opencloud/services/thumbnails/pkg/thumbnail"
	"github.com/opencloud-eu/opencloud/services/thumbnails/pkg/thumbnail/imgsource"
)

// NewService returns a service implementation for Service.
func NewService(opts ...Option) decorators.DecoratedService {
	options := newOptions(opts...)
	logger := options.Logger
	resolutions, err := thumbnail.ParseResolutions(options.Config.Thumbnail.Resolutions)
	if err != nil {
		logger.Fatal().Err(err).Msg("resolutions not configured correctly")
	}
	svc := Thumbnail{
		serviceID: options.Config.GRPC.Namespace + "." + options.Config.Service.Name,
		manager: thumbnail.NewSimpleManager(
			resolutions,
			options.ThumbnailStorage,
			logger,
			options.Config.Thumbnail.MaxInputWidth,
			options.Config.Thumbnail.MaxInputHeight,
		),
		webdavSource: options.ImageSource,
		cs3Source:    options.CS3Source,
		logger:       logger,
		selector:     options.GatewaySelector,
		preprocessorOpts: PreprocessorOpts{
			TxtFontFileMap: options.Config.Thumbnail.FontMapFile,
		},
		dataEndpoint:   options.Config.Thumbnail.DataEndpoint,
		transferSecret: options.Config.Thumbnail.TransferSecret,
	}

	return svc
}

// Thumbnail implements the GRPC handler.
type Thumbnail struct {
	serviceID        string
	dataEndpoint     string
	transferSecret   string
	manager          thumbnail.Manager
	webdavSource     imgsource.Source
	cs3Source        imgsource.Source
	logger           log.Logger
	selector         pool.Selectable[gateway.GatewayAPIClient]
	preprocessorOpts PreprocessorOpts
}

// PreprocessorOpts holds the options for the preprocessor
type PreprocessorOpts struct {
	TxtFontFileMap string
}

// GetThumbnail retrieves a thumbnail for an image
func (g Thumbnail) GetThumbnail(ctx context.Context, req *thumbnailssvc.GetThumbnailRequest, rsp *thumbnailssvc.GetThumbnailResponse) error {
	var err error
	var key string
	switch {
	case req.GetWebdavSource() != nil:
		key, err = g.handleWebdavSource(ctx, req)
	case req.GetCs3Source() != nil:
		key, err = g.handleCS3Source(ctx, req)
	default:
		g.logger.Error().Msg("no image source provided")
		return merrors.BadRequest(g.serviceID, "image source is missing")
	}
	if err != nil {
		return err
	}

	claims := tjwt.ThumbnailClaims{
		Key: key,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Minute)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	transferToken, err := token.SignedString([]byte(g.transferSecret))
	if err != nil {
		g.logger.Error().
			Err(err).
			Msg("GetThumbnail: failed to sign token")
		return merrors.InternalServerError(g.serviceID, "couldn't finish request")
	}
	rsp.DataEndpoint = g.dataEndpoint
	rsp.TransferToken = transferToken
	return nil
}

func (g Thumbnail) checkThumbnail(req *thumbnailssvc.GetThumbnailRequest, sRes *provider.StatResponse) (string, thumbnail.Request, error) {
	tr := thumbnail.Request{}
	if !sRes.GetInfo().GetPermissionSet().GetInitiateFileDownload() {
		return "", tr, merrors.Forbidden(g.serviceID, "no download permission")
	}

	tType := thumbnail.GetExtForMime(sRes.GetInfo().GetMimeType())
	if tType == "" {
		tType = req.GetThumbnailType().String()
	}
	tr, err := thumbnail.PrepareRequest(int(req.GetWidth()), int(req.GetHeight()), tType, sRes.GetInfo().GetChecksum().GetSum(), req.GetProcessor())
	if err != nil {
		return "", tr, merrors.BadRequest(g.serviceID, "%s", err.Error())
	}

	if key, exists := g.manager.CheckThumbnail(tr); exists {
		return key, tr, nil
	}
	return "", tr, nil
}

func (g Thumbnail) handleCS3Source(ctx context.Context, req *thumbnailssvc.GetThumbnailRequest) (string, error) {
	src := req.GetCs3Source()
	sRes, err := g.stat(src.GetPath(), src.GetAuthorization())
	if err != nil {
		return "", err
	}

	key, tr, err := g.checkThumbnail(req, sRes)
	switch {
	case err != nil:
		return "", err
	case key != "":
		// we have matching thumbnail already, use that
		return key, nil
	}

	ctx = imgsource.ContextSetAuthorization(ctx, src.GetAuthorization())
	r, err := g.cs3Source.Get(ctx, src.GetPath())
	switch {
	case errors.Is(err, terrors.ErrImageTooLarge):
		return "", merrors.Forbidden(g.serviceID, "%s", err.Error())
	case err != nil:
		return "", merrors.InternalServerError(g.serviceID, "could not get image from source: %s", err.Error())
	}

	defer r.Close()
	ppOpts := map[string]interface{}{
		"fontFileMap": g.preprocessorOpts.TxtFontFileMap,
	}
	pp := preprocessor.ForType(sRes.GetInfo().GetMimeType(), ppOpts)
	img, err := pp.Convert(r)
	if img == nil || err != nil {
		return "", merrors.NotFound(g.serviceID, "could not get image")
	}

	key, err = g.manager.Generate(tr, img)
	if errors.Is(err, terrors.ErrImageTooLarge) {
		return "", merrors.Forbidden(g.serviceID, "%s", err.Error())
	}
	return key, err
}

func (g Thumbnail) handleWebdavSource(ctx context.Context, req *thumbnailssvc.GetThumbnailRequest) (string, error) {
	src := req.GetWebdavSource()
	imgURL, err := url.Parse(src.GetUrl())
	if err != nil {
		return "", errors.Wrap(err, "source url is invalid")
	}

	var auth, statPath string
	if src.GetIsPublicLink() {
		q := imgURL.Query()
		var rsp *gateway.AuthenticateResponse
		client, err := g.selector.Next()
		if err != nil {
			return "", merrors.InternalServerError(g.serviceID, "could not select next gateway client: %s", err.Error())
		}
		if q.Get("signature") != "" && q.Get("expiration") != "" {
			// Handle pre-signed public links
			sig := q.Get("signature")
			exp := q.Get("expiration")
			rsp, err = client.Authenticate(ctx, &gateway.AuthenticateRequest{
				Type:         "publicshares",
				ClientId:     src.GetPublicLinkToken(),
				ClientSecret: strings.Join([]string{"signature", sig, exp}, "|"),
			})
		} else {
			rsp, err = client.Authenticate(ctx, &gateway.AuthenticateRequest{
				Type:     "publicshares",
				ClientId: src.GetPublicLinkToken(),
				// We pass an empty password because we expect non pre-signed public links
				// to not be password protected
				ClientSecret: "password|",
			})
		}

		if err != nil {
			return "", merrors.InternalServerError(g.serviceID, "could not authenticate: %s", err.Error())
		}
		auth = rsp.GetToken()
		statPath = path.Join("/public", src.GetPublicLinkToken(), req.GetFilepath())
	} else {
		auth = src.GetRevaAuthorization()
		statPath = req.GetFilepath()
	}
	sRes, err := g.stat(statPath, auth)
	if err != nil {
		return "", err
	}

	key, tr, err := g.checkThumbnail(req, sRes)
	switch {
	case err != nil:
		return "", err
	case key != "":
		// we have matching thumbnail already, use that
		return key, nil
	}

	if src.GetWebdavAuthorization() != "" {
		ctx = imgsource.ContextSetAuthorization(ctx, src.GetWebdavAuthorization())
	}

	// add signature and expiration to webdav url
	signature, expiration := imgURL.Query().Get("signature"), imgURL.Query().Get("expiration")
	params := url.Values{}
	params.Add("signature", signature)
	params.Add("expiration", expiration)
	imgURL.RawQuery = params.Encode()

	r, err := g.webdavSource.Get(ctx, imgURL.String())
	switch {
	case errors.Is(err, terrors.ErrImageTooLarge):
		return "", merrors.Forbidden(g.serviceID, "%s", err.Error())
	case err != nil:
		return "", merrors.InternalServerError(g.serviceID, "could not get image from source: %s", err.Error())
	}
	defer r.Close()
	ppOpts := map[string]interface{}{
		"fontFileMap": g.preprocessorOpts.TxtFontFileMap,
	}
	pp := preprocessor.ForType(sRes.GetInfo().GetMimeType(), ppOpts)
	img, err := pp.Convert(r)
	if img == nil || err != nil {
		return "", merrors.NotFound(g.serviceID, "could not get image")
	}

	key, err = g.manager.Generate(tr, img)
	if errors.Is(err, terrors.ErrImageTooLarge) {
		return "", merrors.Forbidden(g.serviceID, "%s", err.Error())
	}
	return key, err
}

func (g Thumbnail) stat(path, auth string) (*provider.StatResponse, error) {
	ctx := metadata.AppendToOutgoingContext(context.Background(), revactx.TokenHeader, auth)

	ref, err := storagespace.ParseReference(path)
	if err != nil {
		// If the path is not a spaces reference try to handle it like a plain
		// path reference.
		ref = provider.Reference{
			Path: path,
		}
	}

	client, err := g.selector.Next()
	if err != nil {
		return nil, merrors.InternalServerError(g.serviceID, "could not select next gateway client: %s", err.Error())
	}
	req := &provider.StatRequest{Ref: &ref}
	rsp, err := client.Stat(ctx, req)
	if err != nil {
		g.logger.Error().Err(err).Str("path", path).Msg("could not stat file")
		return nil, merrors.InternalServerError(g.serviceID, "could not stat file: %s", err.Error())
	}

	if rsp.GetStatus().GetCode() != rpc.Code_CODE_OK {
		switch rsp.GetStatus().GetCode() {
		case rpc.Code_CODE_NOT_FOUND:
			return nil, merrors.NotFound(g.serviceID, "could not stat file: %s", rsp.GetStatus().GetMessage())
		default:
			g.logger.Error().Str("status_message", rsp.GetStatus().GetMessage()).Str("path", path).Msg("could not stat file")
			return nil, merrors.InternalServerError(g.serviceID, "could not stat file: %s", rsp.GetStatus().GetMessage())
		}
	}
	if rsp.GetInfo().GetType() != provider.ResourceType_RESOURCE_TYPE_FILE {
		return nil, merrors.BadRequest(g.serviceID, "Unsupported file type")
	}
	if utils.ReadPlainFromOpaque(rsp.GetInfo().GetOpaque(), "status") == "processing" {
		return nil, &merrors.Error{
			Id:     g.serviceID,
			Code:   http.StatusTooEarly,
			Detail: "File Processing",
			Status: http.StatusText(http.StatusTooEarly),
		}
	}
	if rsp.GetInfo().GetChecksum().GetSum() == "" {
		g.logger.Error().Msg("resource info is missing checksum")
		return nil, merrors.NotFound(g.serviceID, "resource info is missing a checksum")
	}
	if !thumbnail.IsMimeTypeSupported(rsp.GetInfo().GetMimeType()) {
		return nil, merrors.NotFound(g.serviceID, "Unsupported file type")
	}
	return rsp, nil
}
