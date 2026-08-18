package errors

import "errors"

var (
	// ErrImageTooLarge defines an error when an input image is too large
	ErrImageTooLarge = errors.New("thumbnails: image is too large")
	// ErrInvalidType represents the error when a type can't be encoded.
	ErrInvalidType = errors.New("thumbnails: can't encode this type")
	// ErrNoEncoderForType represents the error when an encoder couldn't be found for a type.
	ErrNoEncoderForType = errors.New("thumbnails: no encoder for this type found")
	// ErrNoGeneratorForType represents the error when a generator couldn't be found for a type.
	ErrNoGeneratorForType = errors.New("thumbnails: no generator for this type found")
	// ErrNoImageFromAudioFile defines an error when an image cannot be extracted from an audio file
	ErrNoImageFromAudioFile = errors.New("thumbnails: could not extract image from audio file")
	// ErrNoConverterForExtractedImageFromGgsFile defines an error when the extracted image from an ggs file could not be converted
	ErrNoConverterForExtractedImageFromGgsFile = errors.New("thumbnails: could not find converter for image extracted from ggs file")
	// ErrNoConverterForExtractedImageFromAudioFile defines an error when the extracted image from an audio file could not be converted
	ErrNoConverterForExtractedImageFromAudioFile = errors.New("thumbnails: could not find converter for image extracted from audio file")
	// ErrNoImageFromRawFile defines an error when no embedded preview image could be extracted from a raw file
	ErrNoImageFromRawFile = errors.New("thumbnails: could not extract preview image from raw file")
	// ErrCS3AuthorizationMissing defines an error when the CS3 authorization is missing
	ErrCS3AuthorizationMissing = errors.New("thumbnails: cs3source - authorization missing")
)
