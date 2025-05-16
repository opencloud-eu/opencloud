package thumbnail

import (
	"bytes"
	"image"
	"mime"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/thumbnails/pkg/errors"
	"github.com/opencloud-eu/opencloud/services/thumbnails/pkg/thumbnail/storage"
)

// Request bundles information needed to generate a thumbnail for a file
type Request struct {
	Resolution image.Rectangle
	Encoder    Encoder
	Generator  Generator
	Checksum   string
}

// Manager is responsible for generating thumbnails
type Manager interface {
	// Generate creates a thumbnail and stores it.
	// The function returns a key with which the actual file can be retrieved.
	Generate(r Request, img interface{}) (string, error)
	// CheckThumbnail checks if a thumbnail with the requested attributes exists.
	// The function will return a status if the file exists and the key to the file.
	CheckThumbnail(r Request) (string, bool)
	// GetThumbnail will load the thumbnail from the storage and return its content.
	GetThumbnail(key string) ([]byte, error)
}

// NewSimpleManager creates a new instance of SimpleManager
func NewSimpleManager(resolutions Resolutions, storage storage.Storage, logger log.Logger, maxInputWidth, maxInputHeight int) SimpleManager {
	return SimpleManager{
		storage:      storage,
		logger:       logger,
		resolutions:  resolutions,
		maxDimension: image.Point{X: maxInputWidth, Y: maxInputHeight},
	}
}

// SimpleManager is a simple implementation of Manager
type SimpleManager struct {
	storage      storage.Storage
	logger       log.Logger
	resolutions  Resolutions
	maxDimension image.Point
}

// Generate creates a thumbnail and stores it
func (s SimpleManager) Generate(r Request, img interface{}) (string, error) {
	var match image.Rectangle

	inputDimensions, err := r.Generator.Dimensions(img)
	if err != nil {
		return "", err
	}
	match = s.resolutions.ClosestMatch(r.Resolution, inputDimensions)

	// validate max input image dimensions - 6016x4000
	if inputDimensions.Size().X > s.maxDimension.X || inputDimensions.Size().Y > s.maxDimension.Y {
		return "", errors.ErrImageTooLarge
	}

	thumbnail, err := r.Generator.Generate(match, img)
	if err != nil {
		return "", err
	}

	buf := new(bytes.Buffer)
	if err := r.Encoder.Encode(buf, thumbnail); err != nil {
		return "", err
	}

	k := s.storage.BuildKey(mapToStorageRequest(r))
	if err := s.storage.Put(k, buf.Bytes()); err != nil {
		s.logger.Error().Err(err).Msg("could not store thumbnail")
		return "", err
	}
	return k, nil
}

// CheckThumbnail checks if a thumbnail with the requested attributes exists.
func (s SimpleManager) CheckThumbnail(r Request) (string, bool) {
	k := s.storage.BuildKey(mapToStorageRequest(r))
	return k, s.storage.Stat(k)
}

// GetThumbnail will load the thumbnail from the storage and return its content.
func (s SimpleManager) GetThumbnail(key string) ([]byte, error) {
	return s.storage.Get(key)
}

func mapToStorageRequest(r Request) storage.Request {
	return storage.Request{
		Checksum:       r.Checksum,
		Resolution:     r.Resolution,
		Types:          r.Encoder.Types(),
		Characteristic: r.Generator.ProcessorID(),
	}
}

// IsMimeTypeSupported validate if the mime type is supported
func IsMimeTypeSupported(m string) bool {
	mimeType, _, err := mime.ParseMediaType(m)
	if err != nil {
		return false
	}
	_, supported := SupportedMimeTypes[mimeType]
	return supported
}

// PrepareRequest prepare the request based on image parameters
func PrepareRequest(width, height int, tType, checksum, pID string) (Request, error) {
	generator, err := GeneratorFor(tType, pID)
	if err != nil {
		return Request{}, err
	}
	encoder, err := EncoderForType(tType)
	if err != nil {
		return Request{}, err
	}

	return Request{
		Resolution: image.Rect(0, 0, width, height),
		Generator:  generator,
		Encoder:    encoder,
		Checksum:   checksum,
	}, nil
}
