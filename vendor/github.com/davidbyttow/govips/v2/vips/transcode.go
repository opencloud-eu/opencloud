package vips

import (
	"errors"
	"fmt"
	"io"
)

// TranscodeOptions configures TranscodeStream.
type TranscodeOptions struct {
	// Format is the output image type. ImageTypeUnknown keeps the
	// input format (which must be one of the streaming-save formats:
	// JPEG, PNG, WebP, HEIF, TIFF, GIF).
	Format ImageType

	// ImportParams configures the load. The Access field is ignored:
	// TranscodeStream picks the access mode itself.
	ImportParams *ImportParams

	// ExportParams configures the encode; nil uses the format's
	// defaults, exactly like SaveToWriter.
	ExportParams *ExportParams

	// AutoRotate applies the EXIF orientation to the pixels and clears
	// the orientation tag. Orientations that need a non-sequential
	// transform (3-8: 180° and 90°/270° families) force the image to be
	// materialized first — see the package streaming documentation.
	AutoRotate bool
}

// TranscodeStream decodes an image from r, optionally applies EXIF
// auto-rotation, and encodes it to w in the requested format. Encoded
// bytes are handed to w in chunks as they are produced and are
// byte-identical to the corresponding Export* method with the same
// parameters.
//
// The pipeline picks the cheapest decode strategy automatically:
//
//   - Sequential fast path: when no random-access transform is needed
//     (AutoRotate false, or EXIF orientation is 1 or 2), pixels flow
//     from r to w in one pass. Peak memory is bounded by libvips line
//     caches, independent of both file size and pixel count.
//   - Materialized path: when AutoRotate requires a rotation, the
//     decoded image is first rendered to memory or to an unlinked
//     scratch file, depending on its decoded size and the configured
//     threshold (SetStreamDiscThreshold, SetStreamScratchDir).
//
// Reader/decode errors (including truncated input) surface from this
// call: on the sequential path during the encode (the first full pass
// over the pixels), on the materialized path during materialization.
// In both cases the original reader/writer error is wrapped and
// matchable with errors.Is.
//
// govips knows nothing about the destination: w is the chunk callback.
// Compose on the consumer side with io.MultiWriter (hash while
// writing), io.Pipe (bridge to reader-shaped sinks), or io.Copy.
func TranscodeStream(r io.Reader, w io.Writer, opts *TranscodeOptions) error {
	if r == nil {
		return errors.New("transcode: reader is nil")
	}
	if w == nil {
		return errors.New("transcode: writer is nil")
	}
	if opts == nil {
		opts = &TranscodeOptions{}
	}

	importParams := NewImportParams()
	if opts.ImportParams != nil {
		*importParams = *opts.ImportParams
	}
	importParams.Access.Set(AccessSequential)

	img, err := LoadImageFromReader(r, importParams)
	if err != nil {
		return fmt.Errorf("transcode: %w", err)
	}
	defer img.Close()

	if opts.AutoRotate {
		// Orientation is header metadata, available before any pixel is
		// decoded. Orientations 1 (none) and 2 (horizontal flip) are
		// sequential-safe; everything else transposes or reverses line
		// order and needs random access.
		if img.Orientation() >= 3 {
			if err := img.materialize(); err != nil {
				return fmt.Errorf("transcode: %w", err)
			}
		}
		if err := img.AutoRotate(); err != nil {
			return fmt.Errorf("transcode: autorotate: %w", err)
		}
	}

	format := opts.Format
	if format == ImageTypeUnknown {
		format = img.Format()
	}

	if err := img.SaveToWriter(w, format, opts.ExportParams); err != nil {
		return fmt.Errorf("transcode: %w", err)
	}
	return nil
}
