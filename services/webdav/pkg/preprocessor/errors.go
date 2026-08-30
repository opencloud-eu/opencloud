package preprocessor

import "errors"

var (
	ErrNoImageFromAudioFile                      = errors.New("preprocessor: could not extract image from audio file")
	ErrNoConverterForExtractedImageFromGgsFile   = errors.New("preprocessor: could not find converter for image extracted from ggs file")
	ErrNoConverterForExtractedImageFromAudioFile = errors.New("preprocessor: could not find converter for image extracted from audio file")
	ErrNoThumbnail                               = errors.New("preprocessor: the document has no thumbnail")
)
