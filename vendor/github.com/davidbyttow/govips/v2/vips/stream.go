package vips

// Streaming I/O via VipsSourceCustom/VipsTargetCustom.
//
// libvips worker threads pull/push bytes through C trampolines (stream.c)
// that dispatch to the exported Go callbacks below via integer handles in
// a global registry. Holding a registry reference also keeps the Go
// reader/writer alive while the C side may still call back into it.

// #include "stream.h"
import "C"

import (
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"unsafe"
)

// sourceEntry is the registry entry for one streaming load. The mutex
// serializes callback invocations so callers never need thread-safe
// readers, even though libvips may call from multiple worker threads.
type sourceEntry struct {
	reader  io.Reader
	seeker  io.Seeker // non-nil only when reader implements io.Seeker
	mu      sync.Mutex
	lastErr error

	// size is the lazily resolved length of the seekable source,
	// cached for clamping past-EOF seek targets. Guarded by mu.
	size      int64
	sizeKnown bool
}

// sourceSizeLocked resolves and caches the source length without
// disturbing the current position. The entry mutex must be held.
func (e *sourceEntry) sourceSizeLocked() (int64, error) {
	if e.sizeKnown {
		return e.size, nil
	}
	cur, err := e.seeker.Seek(0, io.SeekCurrent)
	if err != nil {
		return 0, err
	}
	size, err := e.seeker.Seek(0, io.SeekEnd)
	if err != nil {
		return 0, err
	}
	if _, err := e.seeker.Seek(cur, io.SeekStart); err != nil {
		return 0, err
	}
	e.size, e.sizeKnown = size, true
	return size, nil
}

// targetEntry is the registry entry for one streaming save.
type targetEntry struct {
	writer  io.Writer
	mu      sync.Mutex
	lastErr error
}

func (e *sourceEntry) takeErr() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	err := e.lastErr
	e.lastErr = nil
	return err
}

func (e *targetEntry) takeErr() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	err := e.lastErr
	e.lastErr = nil
	return err
}

// streamCallbacks maps integer handles to active source/target entries.
// The registry mutex only guards the maps; per-entry mutexes guard the
// actual I/O so the global lock is never held during a Read/Write.
var streamCallbacks = struct {
	sync.Mutex
	sources    map[int]*sourceEntry
	targets    map[int]*targetEntry
	nextHandle int
}{
	sources: make(map[int]*sourceEntry),
	targets: make(map[int]*targetEntry),
}

// allocStreamHandle returns the next free handle. Handles cross the CGo
// boundary as C int (via GLib's GINT_TO_POINTER), so they must stay
// within int32 range: wrap instead of overflowing, and skip any handle
// that is still registered after a wrap. The caller must hold the
// streamCallbacks lock.
func allocStreamHandle() int {
	for {
		streamCallbacks.nextHandle++
		if streamCallbacks.nextHandle > math.MaxInt32 {
			streamCallbacks.nextHandle = 1
		}
		h := streamCallbacks.nextHandle
		if _, live := streamCallbacks.sources[h]; live {
			continue
		}
		if _, live := streamCallbacks.targets[h]; live {
			continue
		}
		return h
	}
}

func registerSource(r io.Reader) (int, *sourceEntry) {
	entry := &sourceEntry{reader: r}
	if s, ok := r.(io.Seeker); ok {
		entry.seeker = s
	}

	streamCallbacks.Lock()
	defer streamCallbacks.Unlock()
	handle := allocStreamHandle()
	streamCallbacks.sources[handle] = entry
	return handle, entry
}

func deregisterSource(handle int) {
	streamCallbacks.Lock()
	defer streamCallbacks.Unlock()
	delete(streamCallbacks.sources, handle)
}

func lookupSource(handle int) *sourceEntry {
	streamCallbacks.Lock()
	defer streamCallbacks.Unlock()
	return streamCallbacks.sources[handle]
}

func registerTarget(w io.Writer) (int, *targetEntry) {
	entry := &targetEntry{writer: w}

	streamCallbacks.Lock()
	defer streamCallbacks.Unlock()
	handle := allocStreamHandle()
	streamCallbacks.targets[handle] = entry
	return handle, entry
}

func deregisterTarget(handle int) {
	streamCallbacks.Lock()
	defer streamCallbacks.Unlock()
	delete(streamCallbacks.targets, handle)
}

func lookupTarget(handle int) *targetEntry {
	streamCallbacks.Lock()
	defer streamCallbacks.Unlock()
	return streamCallbacks.targets[handle]
}

//export goSourceReadCb
func goSourceReadCb(handle C.int, buffer unsafe.Pointer, length C.gint64) C.gint64 {
	if length <= 0 {
		return 0
	}
	buf := unsafe.Slice((*byte)(buffer), int(length))
	return C.gint64(sourceRead(int(handle), buf))
}

//export goSourceSeekCb
func goSourceSeekCb(handle C.int, offset C.gint64, whence C.int) C.gint64 {
	return C.gint64(sourceSeek(int(handle), int64(offset), int(whence)))
}

//export goTargetWriteCb
func goTargetWriteCb(handle C.int, data unsafe.Pointer, length C.gint64) C.gint64 {
	if length <= 0 {
		return 0
	}
	buf := unsafe.Slice((*byte)(data), int(length))
	return C.gint64(targetWrite(int(handle), buf))
}

//export goTargetEndCb
func goTargetEndCb(handle C.int) C.int {
	return C.int(targetEnd(int(handle)))
}

func sourceRead(handle int, buf []byte) int64 {
	entry := lookupSource(handle)
	if entry == nil {
		return -1
	}

	entry.mu.Lock()
	defer entry.mu.Unlock()

	// (0, nil) is allowed by the io.Reader contract; retry rather than
	// returning 0, which libvips would treat as EOF. Bound the retries so
	// a pathological reader that returns (0, nil) forever fails the load
	// instead of pinning a libvips worker thread inside this callback.
	const maxZeroReads = 100
	for i := 0; i < maxZeroReads; i++ {
		n, err := entry.reader.Read(buf)
		if n > 0 {
			// A non-EOF error alongside n>0 will surface on the next call.
			return int64(n)
		}
		if errors.Is(err, io.EOF) {
			return 0
		}
		if err != nil {
			entry.lastErr = err
			return -1
		}
	}
	entry.lastErr = fmt.Errorf("reader returned (0, nil) %d times in a row", maxZeroReads)
	return -1
}

func sourceSeek(handle int, offset int64, whence int) int64 {
	entry := lookupSource(handle)
	if entry == nil || entry.seeker == nil {
		return -1
	}

	entry.mu.Lock()
	defer entry.mu.Unlock()

	size, err := entry.sourceSizeLocked()
	if err != nil {
		entry.lastErr = err
		return -1
	}

	// Resolve the absolute target ourselves. libvips whence values are
	// SEEK_SET/SEEK_CUR/SEEK_END, matching io.SeekStart/Current/End.
	var base int64
	switch whence {
	case io.SeekStart:
		base = 0
	case io.SeekCurrent:
		cur, err := entry.seeker.Seek(0, io.SeekCurrent)
		if err != nil {
			entry.lastErr = err
			return -1
		}
		base = cur
	case io.SeekEnd:
		base = size
	default:
		return -1
	}

	target := base + offset
	if target < 0 {
		// POSIX lseek rejects negative positions (EINVAL) without
		// affecting the stream; treat it as a probe, not a stream
		// error, so it is not wrapped into the load error later.
		return -1
	}
	if target > size {
		// POSIX permits seeking beyond EOF, and codecs probe container
		// structures that way (libheif seeks past the final HEIC box).
		// Report the requested position: libvips itself range-checks it
		// against the source length and turns it into the failure the
		// heif glue's wait_for_file_size protocol relies on. Park the
		// underlying seeker at EOF so strict io.Seeker implementations
		// are never asked for an out-of-range position; any read then
		// returns 0 bytes. One divergence from POSIX: a subsequent
		// SEEK_CUR resolves from EOF, not from the reported past-EOF
		// position. That is fine in practice because libvips' codec glue
		// only ever follows a past-EOF probe with an absolute seek.
		if _, err := entry.seeker.Seek(size, io.SeekStart); err != nil {
			entry.lastErr = err
			return -1
		}
		return target
	}

	pos, err := entry.seeker.Seek(target, io.SeekStart)
	if err != nil {
		entry.lastErr = err
		return -1
	}
	return pos
}

func targetWrite(handle int, buf []byte) int64 {
	entry := lookupTarget(handle)
	if entry == nil {
		return -1
	}

	entry.mu.Lock()
	defer entry.mu.Unlock()

	n, err := entry.writer.Write(buf)
	if err != nil {
		entry.lastErr = err
		return -1
	}
	return int64(n)
}

func targetEnd(handle int) int {
	entry := lookupTarget(handle)
	if entry == nil {
		return -1
	}

	entry.mu.Lock()
	defer entry.mu.Unlock()

	if entry.lastErr != nil {
		return -1
	}
	return 0
}

// streamSourceRef ties a live VipsSourceCustom (and its registry entry)
// to a sequentially stream-loaded ImageRef. The image pulls pixels from
// the source on demand, so both stay alive until the image is closed or
// materialized.
type streamSourceRef struct {
	handle int
	entry  *sourceEntry
	source *C.VipsSourceCustom
}

func (s *streamSourceRef) release() {
	C.clear_source(&s.source)
	deregisterSource(s.handle)
}

// streamMaterialize holds the knobs for the default (random-access)
// load path: decoded images at most threshold bytes are materialized in
// memory, larger ones in a scratch file on disc.
var streamMaterialize = struct {
	sync.RWMutex
	scratchDir string // "" → os.TempDir()
	threshold  int64  // <0 → resolve from VIPS_DISC_THRESHOLD / default
}{threshold: -1}

// defaultDiscThreshold mirrors the libvips default for
// VIPS_DISC_THRESHOLD: decoded images above 100 MB go to disc.
const defaultDiscThreshold = 100 << 20

// SetStreamScratchDir sets the directory used for scratch files when a
// stream-loaded image is materialized to disc (see
// SetStreamDiscThreshold). An empty string restores the default
// (os.TempDir()). Scratch files are unlinked as soon as the decoded
// image has been written and reopened, so on the happy path they never
// outlive the ImageRef; a crash mid-materialization can leave one
// behind in this directory.
func SetStreamScratchDir(dir string) {
	streamMaterialize.Lock()
	defer streamMaterialize.Unlock()
	streamMaterialize.scratchDir = dir
}

// SetStreamDiscThreshold sets the decoded-size threshold (in bytes)
// above which LoadImageFromReader materializes images to a scratch file
// on disc instead of memory. Zero sends every image to disc. A negative
// value restores the default: the VIPS_DISC_THRESHOLD environment
// variable (number with optional k/m/g suffix) or 100 MB.
func SetStreamDiscThreshold(bytes int64) {
	streamMaterialize.Lock()
	defer streamMaterialize.Unlock()
	streamMaterialize.threshold = bytes
}

func streamDiscThresholdBytes() int64 {
	streamMaterialize.RLock()
	t := streamMaterialize.threshold
	streamMaterialize.RUnlock()
	if t >= 0 {
		return t
	}
	if env := os.Getenv("VIPS_DISC_THRESHOLD"); env != "" {
		if v, ok := parseVipsSize(env); ok {
			return v
		}
	}
	return defaultDiscThreshold
}

func streamScratchDir() string {
	streamMaterialize.RLock()
	defer streamMaterialize.RUnlock()
	if streamMaterialize.scratchDir != "" {
		return streamMaterialize.scratchDir
	}
	return os.TempDir()
}

// parseVipsSize parses libvips-style size strings: a number with an
// optional k/m/g suffix (powers of 1024).
func parseVipsSize(s string) (int64, bool) {
	s = strings.TrimSpace(strings.ToLower(s))
	mult := int64(1)
	switch {
	case strings.HasSuffix(s, "k"):
		mult, s = 1<<10, s[:len(s)-1]
	case strings.HasSuffix(s, "m"):
		mult, s = 1<<20, s[:len(s)-1]
	case strings.HasSuffix(s, "g"):
		mult, s = 1<<30, s[:len(s)-1]
	}
	v, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil || v < 0 {
		return 0, false
	}
	return int64(v * float64(mult)), true
}

// materializeImage renders the lazy image in to either memory or a
// scratch file on disc, depending on its decoded size and the configured
// threshold. The returned image no longer reads from the source.
func materializeImage(in *C.VipsImage) (*C.VipsImage, error) {
	decoded := int64(C.image_decoded_size(in))

	if decoded <= streamDiscThresholdBytes() {
		var out *C.VipsImage
		if C.copy_image_to_memory(in, &out) != 0 {
			return nil, handleVipsError()
		}
		return out, nil
	}

	f, err := os.CreateTemp(streamScratchDir(), "govips-scratch-*.v")
	if err != nil {
		return nil, fmt.Errorf("streaming load: create scratch file: %w", err)
	}
	path := f.Name()
	_ = f.Close()

	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))

	var out *C.VipsImage
	code := C.write_image_to_disc(in, cPath, &out)
	// Unlink immediately: the open file keeps the data alive until the
	// image is closed. (A crash during write_image_to_disc itself can
	// still leave the scratch file behind; see SetStreamScratchDir.)
	_ = os.Remove(path)
	if code != 0 {
		return nil, handleVipsError()
	}
	return out, nil
}

// sequentialAccess reports whether params request sequential streaming.
func sequentialAccess(params *ImportParams) bool {
	if !params.Access.IsSet() {
		return false
	}
	a := params.Access.Get()
	return a == AccessSequential || a == AccessSequentialUnbuffered
}

// LoadImageFromReader loads an image from the given io.Reader using
// libvips streaming (VipsSourceCustom). If r also implements io.Seeker,
// seek is exposed to libvips for efficient random-access loading.
// Otherwise libvips uses sequential mode with automatic header buffering
// (see SetPipeReadLimit).
//
// By default the image is fully materialized before this function
// returns — in memory when its decoded size is at most the disc
// threshold, otherwise in an unlinked scratch file (see
// SetStreamDiscThreshold and SetStreamScratchDir). The reader is
// consumed during this call and is not retained; callers may close it
// immediately. Decode errors (including truncated input) surface here.
//
// If params.Access is AccessSequential, the image is instead loaded
// lazily: libvips pulls compressed bytes from r on demand while later
// operations consume pixels, so peak memory stays bounded by libvips
// line caches regardless of image size. The reader MUST stay open until
// the ImageRef is closed (Close releases it). Only operations that read
// pixels strictly top-to-bottom are valid on such images — see the
// "Streaming" section of the README. Decode errors surface from the
// operation that first consumes the pixels (typically SaveToWriter or
// Export*), not from this function.
//
// In every mode the returned ImageRef has no buffer backing: the full
// compressed input is never held in Go memory. (NewImageFromReader, by
// contrast, slurps the whole reader into a buffer via io.ReadAll before
// decoding; prefer this function when input size matters.)
//
// Error strictness: when params.FailOnError is set, streaming loads use
// libvips' fail_on=truncated, which is slightly laxer than the
// fail_on=error used by the buffer loaders — truncated input always
// errors deterministically, but some recoverable decode warnings that
// the buffer path would reject are tolerated.
//
// params may be nil for default import settings.
func LoadImageFromReader(r io.Reader, params *ImportParams) (*ImageRef, error) {
	if r == nil {
		return nil, errors.New("reader is nil")
	}
	if err := startupIfNeeded(); err != nil {
		return nil, err
	}
	if params == nil {
		params = NewImportParams()
	}

	incOpCounter("load_source")

	handle, entry := registerSource(r)

	source := C.create_source_custom(C.int(handle), C.int(boolToInt(entry.seeker != nil)))
	if source == nil {
		deregisterSource(handle)
		return nil, handleVipsError()
	}

	// Sniff the signature up front (buffered and rewound, not consumed):
	// the loader nickname alone cannot express sub-formats — AVIF loads
	// via heifload, BMP/PSD via magickload — but the buffer path reports
	// them via DetermineImageType, and the streaming path must match.
	var header [12]byte
	headerKnown := C.source_sniff_header(source, (*C.uchar)(unsafe.Pointer(&header[0])), C.int(len(header))) == 0

	loadParams := createImportParams(ImageTypeUnknown, params)

	if code := C.load_from_source(source, &loadParams); code != 0 {
		err := wrapStreamError("streaming load", handleImageError(loadParams.outputImage), entry.takeErr())
		C.clear_source(&source)
		deregisterSource(handle)
		return nil, err
	}

	lazy := loadParams.outputImage
	format := ImageType(loadParams.inputFormat)
	originalFormat := format
	if headerKnown && (format == ImageTypeHEIF || format == ImageTypeMagick) {
		if sniffed := DetermineImageType(header[:]); sniffed != ImageTypeUnknown {
			originalFormat = sniffed
			if !isNeedToChangeLoaderToMagick(sniffed) {
				format = sniffed
			}
		}
	}

	if sequentialAccess(params) {
		ref := newImageRef(lazy, format, originalFormat, nil)
		ref.streamSource = &streamSourceRef{handle: handle, entry: entry, source: source}
		govipsLog("govips", LogLevelDebug, fmt.Sprintf("created sequential imageRef %p from reader", ref))
		return ref, nil
	}

	// Default path: materialize now so the source (and the caller's
	// reader) can be released before returning. Upstream resources
	// (file handles, HTTP connections) are freed early.
	out, err := materializeImage(lazy)
	clearImage(lazy)
	C.clear_source(&source)
	deregisterSource(handle)
	if err != nil {
		return nil, wrapStreamError("streaming load", err, entry.takeErr())
	}

	ref := newImageRef(out, format, originalFormat, nil)
	govipsLog("govips", LogLevelDebug, fmt.Sprintf("created imageRef %p from reader", ref))
	return ref, nil
}

// materialize converts a sequentially stream-loaded image into a
// materialized one (memory or scratch disc, by threshold) and releases
// its source, making random-access operations valid. It is a no-op for
// images that are already materialized.
func (r *ImageRef) materialize() error {
	r.lock.Lock()

	if r.image == nil {
		r.lock.Unlock()
		return errors.New("attempt to materialize a closed ImageRef")
	}
	src := r.streamSource
	if src == nil {
		r.lock.Unlock()
		return nil
	}

	out, err := materializeImage(r.image)
	if err != nil {
		r.lock.Unlock()
		return wrapStreamError("streaming load", err, src.entry.takeErr())
	}

	clearImage(r.image)
	r.image = out
	r.streamSource = nil
	r.lock.Unlock()

	src.release()
	return nil
}

// SaveToWriter encodes the image in the specified format and writes the
// encoded bytes directly to w using libvips streaming (VipsTargetCustom).
// The writer receives chunks of encoded data as they are produced and is
// not retained after SaveToWriter returns.
//
// Supported formats: ImageTypeJPEG, ImageTypePNG, ImageTypeWEBP,
// ImageTypeHEIF, ImageTypeTIFF, ImageTypeGIF. Output is byte-identical to
// the corresponding Export* method with equivalent parameters.
//
// TIFF is encoded in memory and written in a single chunk, because the
// TIFF container requires seekable output; all other formats stream
// encoded chunks to w as they are produced.
//
// params may be nil for the format's default export settings; the format
// argument takes precedence over params.Format. For access to the full
// set of format-specific options, use the typed variants
// (SaveToWriterJpeg, SaveToWriterPng, SaveToWriterWebp, SaveToWriterTiff,
// SaveToWriterHeif, SaveToWriterGif).
func (r *ImageRef) SaveToWriter(w io.Writer, format ImageType, params *ExportParams) error {
	return r.saveToWriter(w, format, func(in *C.VipsImage) (C.struct_SaveParams, func(), error) {
		return streamSaveParams(in, format, params)
	})
}

// SaveToWriterJpeg streams the image to w as JPEG with the full set of
// JPEG export options. Behaves like SaveToWriter; params may be nil for
// defaults. Output is byte-identical to ExportJpeg.
func (r *ImageRef) SaveToWriterJpeg(w io.Writer, params *JpegExportParams) error {
	if params == nil {
		params = NewJpegExportParams()
	}
	p := *params
	return r.saveToWriter(w, ImageTypeJPEG, func(in *C.VipsImage) (C.struct_SaveParams, func(), error) {
		return newSaveParamsJPEG(in, p), func() {}, nil
	})
}

// SaveToWriterPng streams the image to w as PNG with the full set of PNG
// export options (Palette, Dither, Bitdepth, Filter, ...). Behaves like
// SaveToWriter; params may be nil for defaults. Output is byte-identical
// to ExportPng.
func (r *ImageRef) SaveToWriterPng(w io.Writer, params *PngExportParams) error {
	if params == nil {
		params = NewPngExportParams()
	}
	p := *params
	return r.saveToWriter(w, ImageTypePNG, func(in *C.VipsImage) (C.struct_SaveParams, func(), error) {
		return newSaveParamsPNG(in, p), func() {}, nil
	})
}

// SaveToWriterWebp streams the image to w as WebP with the full set of
// WebP export options (NearLossless, IccProfile, TargetSize, ...).
// Behaves like SaveToWriter; params may be nil for defaults. Output is
// byte-identical to ExportWebp.
func (r *ImageRef) SaveToWriterWebp(w io.Writer, params *WebpExportParams) error {
	if params == nil {
		params = NewWebpExportParams()
	}
	p := *params
	return r.saveToWriter(w, ImageTypeWEBP, func(in *C.VipsImage) (C.struct_SaveParams, func(), error) {
		return newSaveParamsWebP(in, p)
	})
}

// SaveToWriterTiff writes the image to w as TIFF with the full set of
// TIFF export options (Compression, Predictor, Tile, ...). Like the TIFF
// path of SaveToWriter, the image is encoded in memory and written in a
// single chunk (libtiff requires seekable output). params may be nil for
// defaults. Output is byte-identical to ExportTiff.
func (r *ImageRef) SaveToWriterTiff(w io.Writer, params *TiffExportParams) error {
	if params == nil {
		params = NewTiffExportParams()
	}
	p := *params
	return r.saveToWriter(w, ImageTypeTIFF, func(in *C.VipsImage) (C.struct_SaveParams, func(), error) {
		return newSaveParamsTIFF(in, p), func() {}, nil
	})
}

// SaveToWriterHeif streams the image to w as HEIF with the full set of
// HEIF export options (Bitdepth, Effort, ...). Behaves like SaveToWriter;
// params may be nil for defaults. Output is byte-identical to ExportHeif.
func (r *ImageRef) SaveToWriterHeif(w io.Writer, params *HeifExportParams) error {
	if params == nil {
		params = NewHeifExportParams()
	}
	p := *params
	return r.saveToWriter(w, ImageTypeHEIF, func(in *C.VipsImage) (C.struct_SaveParams, func(), error) {
		return newSaveParamsHEIF(in, p), func() {}, nil
	})
}

// SaveToWriterGif streams the image to w as GIF with the full set of GIF
// export options (Dither, Effort, Bitdepth, ...). Behaves like
// SaveToWriter; params may be nil for defaults. Output is byte-identical
// to ExportGIF.
func (r *ImageRef) SaveToWriterGif(w io.Writer, params *GifExportParams) error {
	if params == nil {
		params = NewGifExportParams()
	}
	p := *params
	return r.saveToWriter(w, ImageTypeGIF, func(in *C.VipsImage) (C.struct_SaveParams, func(), error) {
		return newSaveParamsGIF(in, p), func() {}, nil
	})
}

// saveToWriter is the shared core of SaveToWriter and its typed
// variants: it locks the image, builds the C save params via
// buildParams, and runs the streaming (or, for TIFF, buffered) save.
func (r *ImageRef) saveToWriter(w io.Writer, format ImageType, buildParams func(*C.VipsImage) (C.struct_SaveParams, func(), error)) error {
	if w == nil {
		return errors.New("writer is nil")
	}

	r.lock.Lock()
	defer r.lock.Unlock()
	defer runtime.KeepAlive(r)

	if r.image == nil {
		return errors.New("attempt to save a closed ImageRef")
	}

	saveParams, cleanup, err := buildParams(r.image)
	if err != nil {
		return err
	}
	defer cleanup()

	incOpCounter("save_" + ImageTypes[format] + "_target")

	if format == ImageTypeTIFF {
		// libtiff requires a seekable, readable output stream (it
		// rewrites IFD offsets after encoding), which a plain io.Writer
		// cannot provide. Encode through the buffer path and emit a
		// single write; the bytes are identical to ExportTiff.
		buf, err := vipsSaveToBuffer(saveParams)
		if err != nil {
			var ioErr error
			if r.streamSource != nil {
				// Sequential decode runs during this encode; surface
				// the reader's error like the streaming-target path.
				ioErr = r.streamSource.entry.takeErr()
			}
			return wrapStreamError("streaming save", err, ioErr)
		}
		if _, err := w.Write(buf); err != nil {
			return fmt.Errorf("streaming save: writer error: %w", err)
		}
		return nil
	}

	handle, entry := registerTarget(w)
	defer deregisterTarget(handle)

	target := C.create_target_custom(C.int(handle))
	if target == nil {
		return handleVipsError()
	}
	defer C.clear_target(&target)

	var code C.int
	switch format {
	case ImageTypeJPEG:
		code = C.save_jpeg_to_target(&saveParams, target)
	case ImageTypePNG:
		code = C.save_png_to_target(&saveParams, target)
	case ImageTypeWEBP:
		code = C.save_webp_to_target(&saveParams, target)
	case ImageTypeHEIF:
		code = C.save_heif_to_target(&saveParams, target)
	// ImageTypeTIFF is handled by the buffer-path early return above.
	case ImageTypeGIF:
		code = C.save_gif_to_target(&saveParams, target)
	}

	if code != 0 {
		ioErr := entry.takeErr()
		if ioErr == nil && r.streamSource != nil {
			// For sequentially stream-loaded images the decode runs
			// during this save; surface the reader's error too.
			ioErr = r.streamSource.entry.takeErr()
		}
		return wrapStreamError("streaming save", handleVipsError(), ioErr)
	}
	if ioErr := entry.takeErr(); ioErr != nil {
		return fmt.Errorf("streaming save: writer error: %w", ioErr)
	}
	return nil
}

// streamSaveParams builds the C save parameters for SaveToWriter using
// the same ExportParams mapping as (*ImageRef).Export (via the shared
// *ParamsFromExport helpers) and the same C-struct population as the
// Export* buffer savers, so streaming output stays byte-identical to the
// buffer path. The returned cleanup must be called after the save
// completes.
func streamSaveParams(in *C.VipsImage, format ImageType, params *ExportParams) (C.struct_SaveParams, func(), error) {
	noop := func() {}

	switch format {
	case ImageTypeJPEG:
		return newSaveParamsJPEG(in, *jpegParamsFromExport(params)), noop, nil
	case ImageTypePNG:
		return newSaveParamsPNG(in, *pngParamsFromExport(params)), noop, nil
	case ImageTypeWEBP:
		return newSaveParamsWebP(in, *webpParamsFromExport(params))
	case ImageTypeHEIF:
		return newSaveParamsHEIF(in, *heifParamsFromExport(params)), noop, nil
	case ImageTypeTIFF:
		return newSaveParamsTIFF(in, *tiffParamsFromExport(params)), noop, nil
	case ImageTypeGIF:
		return newSaveParamsGIF(in, *gifParamsFromExport(params)), noop, nil
	default:
		return C.struct_SaveParams{}, noop, fmt.Errorf("streaming save does not support format %q", ImageTypes[format])
	}
}

// wrapStreamError combines the libvips error with the original Go
// reader/writer error, when one was stored during a callback.
func wrapStreamError(op string, vipsErr, ioErr error) error {
	if ioErr != nil {
		return fmt.Errorf("%s: %w (caused by: %w)", op, vipsErr, ioErr)
	}
	return vipsErr
}
