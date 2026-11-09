// Streaming I/O via VipsSourceCustom/VipsTargetCustom.
// C declarations for the callback bridge between libvips worker threads
// and Go io.Reader/io.Writer instances registered in stream.go.

#ifndef STREAM_H
#define STREAM_H

#include "foreign.h"

// Exported Go callbacks, defined in stream.go via //export.
// Called by the static C trampolines in stream.c. The handle identifies
// an entry in the Go-side callback registry. The buffer/data pointers are
// owned by libvips and are only valid for the duration of the call; Go
// must not retain them.

// Reads up to length bytes into buffer. Returns bytes read, 0 on EOF,
// -1 on error.
extern gint64 goSourceReadCb(int handle, void *buffer, gint64 length);

// Seeks the underlying reader. Returns the new absolute offset, or -1 on
// error or if the reader does not support seeking.
extern gint64 goSourceSeekCb(int handle, gint64 offset, int whence);

// Writes length bytes from data. Returns bytes written, -1 on error.
extern gint64 goTargetWriteCb(int handle, void *data, gint64 length);

// Signals end of output. Returns 0 on success, -1 on error.
extern int goTargetEndCb(int handle);

// Creates a VipsSourceCustom with the "read" trampoline connected, and the
// "seek" trampoline too when has_seek is non-zero. The handle is stored as
// signal user_data. Returns a new reference owned by the caller; release
// with clear_source. Returns NULL on failure.
VipsSourceCustom *create_source_custom(int handle, int has_seek);

// Creates a VipsTargetCustom with the "write" and "end" trampolines
// connected. The handle is stored as signal user_data. Returns a new
// reference owned by the caller; release with clear_target. Returns NULL
// on failure.
VipsTargetCustom *create_target_custom(int handle);

// Detects the image format by sniffing the source, then loads it.
// Borrows source (does not take ownership). On success returns 0, sets
// params->inputFormat to the detected format, and stores a new VipsImage
// reference in params->outputImage owned by the caller. The image is
// LAZY: it pulls from the source on demand until materialized, so the
// source must stay alive (and registered) until the image is either
// materialized or closed. On failure returns non-zero with the error in
// the vips error buffer.
int load_from_source(VipsSourceCustom *source, LoadParams *params);

// Returns the decoded (uncompressed) pixel size of in, in bytes.
gint64 image_decoded_size(VipsImage *in);

// Renders in into a memory image. Borrows in; on success stores a new
// reference in *out owned by the caller. Returns non-zero on failure.
int copy_image_to_memory(VipsImage *in, VipsImage **out);

// Renders in into the vips-native file at path (one sequential pass),
// then reopens it read-only with random access. Borrows in; on success
// stores a new reference in *out owned by the caller; the caller may
// unlink path immediately (the open file keeps the data alive).
// Returns non-zero on failure.
int write_image_to_disc(VipsImage *in, const char *path, VipsImage **out);

// Encode params->inputImage to the target. Both the image and the target
// are borrowed (the caller retains ownership). Return 0 on success,
// non-zero on failure with the error in the vips error buffer.
int save_jpeg_to_target(SaveParams *params, VipsTargetCustom *target);
int save_png_to_target(SaveParams *params, VipsTargetCustom *target);
int save_webp_to_target(SaveParams *params, VipsTargetCustom *target);
int save_heif_to_target(SaveParams *params, VipsTargetCustom *target);
int save_gif_to_target(SaveParams *params, VipsTargetCustom *target);

// Copies the first len bytes of the source into out without consuming
// them (vips_source_sniff buffers and rewinds). Returns non-zero if the
// stream is shorter than len or unreadable. Call before loading so the
// sniff window is still rewindable for every source kind.
int source_sniff_header(VipsSourceCustom *source, unsigned char *out, int len);

// Unrefs *source / *target and sets it to NULL. NULL-safe.
void clear_source(VipsSourceCustom **source);
void clear_target(VipsTargetCustom **target);

#endif  // STREAM_H
