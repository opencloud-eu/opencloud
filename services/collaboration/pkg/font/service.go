package font

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"mime"
	"net/http"
	"net/url"
	"path"
	"path/filepath"
	"strconv"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	"github.com/go-playground/validator/v10"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/collaboration/pkg/collaboration"
)

type ServiceOptions struct {
	logger          log.Logger                                `validate:"required"`
	fontFS          afero.Fs                                  `validate:"required"`
	rootURI         string                                    `validate:"required,url"`
	previewText     string                                    `validate:"required,min=1"`
	gatewaySelector pool.Selectable[gateway.GatewayAPIClient] `validate:"required"`
}

func (o ServiceOptions) WithFontFS(fSys afero.Fs) ServiceOptions {
	o.fontFS = fSys
	return o
}

func (o ServiceOptions) WithRootURI(rootURI string) ServiceOptions {
	o.rootURI = rootURI
	return o
}

func (o ServiceOptions) WithLogger(logger log.Logger) ServiceOptions {
	o.logger = logger
	return o
}

func (o ServiceOptions) WithPreviewText(txt string) ServiceOptions {
	o.previewText = txt
	return o
}

func (o ServiceOptions) WithGatewaySelector(gws pool.Selectable[gateway.GatewayAPIClient]) ServiceOptions {
	o.gatewaySelector = gws
	return o
}

func (o ServiceOptions) validate() error {
	return validator.New(
		validator.WithPrivateFieldValidation(),
		validator.WithRequiredStructEnabled(),
	).Struct(o)
}

type Service struct {
	logger          log.Logger
	fontFS          afero.Fs
	rootURI         string
	previewText     string
	gatewaySelector pool.Selectable[gateway.GatewayAPIClient]
}

func NewService(options ServiceOptions) (Service, error) {
	if err := options.validate(); err != nil {
		return Service{}, err
	}

	return Service(options), nil
}

func (s Service) DeleteFont(w http.ResponseWriter, r *http.Request) {
	gatewayClient, err := s.gatewaySelector.Next()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}

	_, canManage, err := collaboration.CheckPermissions(gatewayClient, r.Context(), collaboration.PermissionCollaborationManageFonts)
	switch {
	case err != nil:
		w.WriteHeader(http.StatusInternalServerError)
		return
	case !canManage:
		w.WriteHeader(http.StatusForbidden)
		return
	}

	fontName := r.PathValue("id")
	if fontName == "" {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = s.fontFS.Remove(fontName)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s Service) GetFont(w http.ResponseWriter, r *http.Request) {
	fontName := r.PathValue("id")
	if fontName == "" {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	b, err := s.getFont(fontName)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	contentType := mime.TypeByExtension(filepath.Ext(fontName))
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Length", strconv.Itoa(len(b)))
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("X-Content-Type-Options", "nosniff")

	_, _ = w.Write(b)
}

func (s Service) PreviewFont(w http.ResponseWriter, r *http.Request) {
	fontName := r.PathValue("id")
	if fontName == "" {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	b, err := s.getFont(fontName)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	f, err := opentype.Parse(b)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	face, err := opentype.NewFace(f, &opentype.FaceOptions{
		Size:    55,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer func() {
		_ = face.Close()
	}()

	width := 300
	height := 70
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	bg := color.RGBA{R: 255, G: 255, B: 255, A: 255}
	for i := range img.Pix {
		img.Pix[i] = bg.R
	}

	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(color.RGBA{0, 0, 0, 255}),
		Face: face,
		Dot:  fixed.Point26_6{X: fixed.I(10), Y: fixed.I(50)},
	}

	d.DrawString(s.previewText)

	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")

	if err := png.Encode(w, img); err != nil {
		http.Error(w, "Failed to encode image", http.StatusInternalServerError)
		return
	}
}

func (s Service) ListFonts(w http.ResponseWriter, _ *http.Request) {
	fontFiles, err := afero.NewIOFS(s.fontFS).ReadDir(".")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	fontMap := []map[string]any{}
	for _, fontFile := range fontFiles {
		func() {
			uri, err := url.JoinPath(s.rootURI, path.Base(fontFile.Name()))
			if err != nil {
				return
			}

			fontLogger := s.logger.Debug().Str("font", fontFile.Name())

			b, err := s.getFont(fontFile.Name())
			if err != nil {
				fontLogger.Err(err).Msg("could not get font")
				return
			}

			fnt, err := sfnt.Parse(b)
			if err != nil {
				fontLogger.Err(err).Msg("could not parse font")
				return
			}

			buf := new(sfnt.Buffer)
			nameID := func(id sfnt.NameID) string {
				name, err := fnt.Name(buf, id)
				if err != nil {
					fontLogger.Err(err).Msg("could not extract font details")
				}

				return name
			}

			fontMap = append(fontMap, map[string]any{
				"file":         path.Base(fontFile.Name()),
				"copyright":    nameID(sfnt.NameIDCopyright),
				"family":       nameID(sfnt.NameIDFamily),
				"version":      nameID(sfnt.NameIDVersion),
				"trademark":    nameID(sfnt.NameIDTrademark),
				"manufacturer": nameID(sfnt.NameIDManufacturer),
				"designer":     nameID(sfnt.NameIDDesigner),
				"description":  nameID(sfnt.NameIDDescription),
				"vendor_url":   nameID(sfnt.NameIDVendorURL),
				"designer_url": nameID(sfnt.NameIDDesignerURL),
				"license":      nameID(sfnt.NameIDLicense),
				"license_url":  nameID(sfnt.NameIDLicenseURL),
				"uri":          uri,
				// if stamp property changes, the font file will be re-downloaded by collabora
				"stamp": fmt.Sprintf("%x", sha256.Sum256(b)),
			})
		}()
	}

	b, err := json.Marshal(map[string]any{
		"kind":   "fontconfiguration",
		"server": "OpenCloud Fonts",
		"fonts":  fontMap,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(b)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s Service) UploadFont(w http.ResponseWriter, r *http.Request) {
	gatewayClient, err := s.gatewaySelector.Next()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}

	_, canManage, err := collaboration.CheckPermissions(gatewayClient, r.Context(), collaboration.PermissionCollaborationManageFonts)
	switch {
	case err != nil:
		w.WriteHeader(http.StatusInternalServerError)
		return
	case !canManage:
		w.WriteHeader(http.StatusForbidden)
		return
	}

	file, fileHeader, err := r.FormFile("font")
	switch {
	case err != nil && errors.Is(err, http.ErrMissingFile):
		w.WriteHeader(http.StatusBadRequest)
		return
	case err != nil:
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer func() {
		_ = file.Close()
	}()

	b, err := io.ReadAll(file)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if _, err = sfnt.Parse(b); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = afero.WriteFile(s.fontFS, fileHeader.Filename, b, 0o666)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s Service) getFont(name string) ([]byte, error) {
	fontFile, err := s.fontFS.Open(name)
	if err != nil {
		return nil, fmt.Errorf("could not open font file: %w", err)
	}
	defer func() {
		_ = fontFile.Close()
	}()

	fontStat, err := fontFile.Stat()
	if err != nil || fontStat.IsDir() {
		return nil, fmt.Errorf("could not stat font file: %w", err)
	}

	b, err := io.ReadAll(fontFile)
	if err != nil || len(b) == 0 {
		return nil, fmt.Errorf("could not read font file: %w", err)
	}

	return b, nil
}
