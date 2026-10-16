#include "stream.h"

#include <string.h>

// Option setters shared with the buffer path, defined in foreign.c.
// Reusing them keeps streaming output identical to the buffer output.
extern int set_jpegload_options(VipsOperation *operation, LoadParams *params);
extern int set_pngload_options(VipsOperation *operation, LoadParams *params);
extern int set_webpload_options(VipsOperation *operation, LoadParams *params);
extern int set_tiffload_options(VipsOperation *operation, LoadParams *params);
extern int set_gifload_options(VipsOperation *operation, LoadParams *params);
extern int set_heifload_options(VipsOperation *operation, LoadParams *params);
extern int set_svgload_options(VipsOperation *operation, LoadParams *params);
extern int set_pdfload_options(VipsOperation *operation, LoadParams *params);
extern int set_jp2kload_options(VipsOperation *operation, LoadParams *params);
extern int set_jxlload_options(VipsOperation *operation, LoadParams *params);
extern int set_magickload_options(VipsOperation *operation, LoadParams *params);

extern int set_jpegsave_options(VipsOperation *operation, SaveParams *params);
extern int set_pngsave_options(VipsOperation *operation, SaveParams *params);
extern int set_webpsave_options(VipsOperation *operation, SaveParams *params);
extern int set_heifsave_options(VipsOperation *operation, SaveParams *params);
extern int set_tiffsave_options(VipsOperation *operation, SaveParams *params);
extern int set_gifsave_options(VipsOperation *operation, SaveParams *params);

// Trampolines: extract the registry handle from signal user_data and
// forward to the exported Go callbacks. Buffers are owned by libvips and
// only valid for the duration of the call.

static gint64 source_read_handler(VipsSourceCustom *source, void *buffer,
                                  gint64 length, void *user_data) {
  (void)source;
  return goSourceReadCb(GPOINTER_TO_INT(user_data), buffer, length);
}

static gint64 source_seek_handler(VipsSourceCustom *source, gint64 offset,
                                  int whence, void *user_data) {
  (void)source;
  return goSourceSeekCb(GPOINTER_TO_INT(user_data), offset, whence);
}

static gint64 target_write_handler(VipsTargetCustom *target, const void *data,
                                   gint64 length, void *user_data) {
  (void)target;
  return goTargetWriteCb(GPOINTER_TO_INT(user_data), (void *)data, length);
}

static int target_end_handler(VipsTargetCustom *target, void *user_data) {
  (void)target;
  return goTargetEndCb(GPOINTER_TO_INT(user_data));
}

VipsSourceCustom *create_source_custom(int handle, int has_seek) {
  VipsSourceCustom *source = vips_source_custom_new();
  if (!source) {
    return NULL;
  }

  g_signal_connect(source, "read", G_CALLBACK(source_read_handler),
                   GINT_TO_POINTER(handle));
  if (has_seek) {
    g_signal_connect(source, "seek", G_CALLBACK(source_seek_handler),
                     GINT_TO_POINTER(handle));
  }

  return source;
}

VipsTargetCustom *create_target_custom(int handle) {
  VipsTargetCustom *target = vips_target_custom_new();
  if (!target) {
    return NULL;
  }

  g_signal_connect(target, "write", G_CALLBACK(target_write_handler),
                   GINT_TO_POINTER(handle));
  g_signal_connect(target, "end", G_CALLBACK(target_end_handler),
                   GINT_TO_POINTER(handle));

  return target;
}

int source_sniff_header(VipsSourceCustom *source, unsigned char *out, int len) {
  const unsigned char *p = vips_source_sniff(VIPS_SOURCE(source), (size_t) len);
  if (!p) {
    return -1;
  }
  memcpy(out, p, (size_t) len);
  return 0;
}

static ImageType image_type_for_loader(const char *name) {
  if (g_str_has_prefix(name, "jpegload")) return JPEG;
  if (g_str_has_prefix(name, "pngload")) return PNG;
  if (g_str_has_prefix(name, "webpload")) return WEBP;
  if (g_str_has_prefix(name, "tiffload")) return TIFF;
  if (g_str_has_prefix(name, "gifload")) return GIF;
  if (g_str_has_prefix(name, "heifload")) return HEIF;
  if (g_str_has_prefix(name, "svgload")) return SVG;
  if (g_str_has_prefix(name, "pdfload")) return PDF;
  if (g_str_has_prefix(name, "jp2kload")) return JP2K;
  if (g_str_has_prefix(name, "jxlload")) return JXL;
  if (g_str_has_prefix(name, "magickload")) return MAGICK;
  return UNKNOWN;
}

typedef int (*SetLoadOptionsFn)(VipsOperation *operation, LoadParams *params);

static int set_noload_options(VipsOperation *operation, LoadParams *params) {
  (void)operation;
  (void)params;
  return 0;
}

static SetLoadOptionsFn load_options_for_type(ImageType imageType) {
  switch (imageType) {
    case JPEG:
      return set_jpegload_options;
    case PNG:
      return set_pngload_options;
    case WEBP:
      return set_webpload_options;
    case TIFF:
      return set_tiffload_options;
    case GIF:
      return set_gifload_options;
    case HEIF:
      return set_heifload_options;
    case SVG:
      return set_svgload_options;
    case PDF:
      return set_pdfload_options;
    case JP2K:
      return set_jp2kload_options;
    case JXL:
      return set_jxlload_options;
    case MAGICK:
      return set_magickload_options;
    default:
      return set_noload_options;
  }
}

int load_from_source(VipsSourceCustom *source, LoadParams *params) {
  // Sniffs the stream start to find a loader. This returns the GType
  // class name (e.g. "VipsForeignLoadJpegSource"); resolve it to the
  // operation nickname (e.g. "jpegload_source") for format mapping.
  const char *className = vips_foreign_find_load_source(VIPS_SOURCE(source));
  if (!className) {
    return 1;
  }
  const char *operationName = vips_nickname_find(g_type_from_name(className));
  if (!operationName) {
    return 1;
  }

  params->inputFormat = image_type_for_loader(operationName);
  SetLoadOptionsFn setLoadOptions = load_options_for_type(params->inputFormat);

  VipsOperation *operation = vips_operation_new(operationName);
  if (!operation) {
    return 1;
  }

  if (vips_object_set(VIPS_OBJECT(operation), "source", source, NULL)) {
    g_object_unref(operation);
    return 1;
  }

  if (setLoadOptions(operation, params)) {
    g_object_unref(operation);
    return 1;
  }

  // The deprecated "fail" bool does not make every loader reject
  // truncated input (e.g. spng pads missing rows with only a warning).
  // Streaming inputs are network-shaped, so when the caller asked for
  // fail-on-error escalate to the modern fail_on property at the
  // "truncated" level: truncated streams fail, ordinary warnings on
  // valid images do not.
  if (params->fail.is_set && params->fail.value.b &&
      g_object_class_find_property(G_OBJECT_GET_CLASS(operation),
                                   "fail_on")) {
    vips_object_set(VIPS_OBJECT(operation), "fail_on",
                    VIPS_FAIL_ON_TRUNCATED, NULL);
  }

  // Build uncached: the cache key includes the unique source object, so
  // a hit is impossible — caching would only pin the source and the
  // lazy image in the operation cache past their natural lifetime.
  if (vips_object_build(VIPS_OBJECT(operation))) {
    vips_object_unref_outputs(VIPS_OBJECT(operation));
    g_object_unref(operation);
    return 1;
  }

  g_object_get(VIPS_OBJECT(operation), "out", &params->outputImage, NULL);

  vips_object_unref_outputs(VIPS_OBJECT(operation));
  g_object_unref(operation);

  // The returned image is LAZY: libvips decodes on demand, pulling from
  // the source during later operations. The Go side decides whether to
  // materialize it now (memory or scratch disc, releasing the source
  // early) or keep the source connected for sequential streaming.
  return 0;
}

gint64 image_decoded_size(VipsImage *in) {
  return (gint64)VIPS_IMAGE_SIZEOF_PEL(in) * in->Xsize * in->Ysize;
}

int copy_image_to_memory(VipsImage *in, VipsImage **out) {
  VipsImage *memory = vips_image_copy_memory(in);
  if (!memory) {
    return 1;
  }
  *out = memory;
  return 0;
}

int write_image_to_disc(VipsImage *in, const char *path, VipsImage **out) {
  // Render the full image to a vips-native (.v) scratch file in one
  // sequential pass, then reopen it as a random-access image backed by
  // the file. Decode/reader errors surface here, during the write pass.
  if (vips_image_write_to_file(in, path, NULL)) {
    return 1;
  }

  VipsImage *disc =
      vips_image_new_from_file(path, "access", VIPS_ACCESS_RANDOM, NULL);
  if (!disc) {
    return 1;
  }
  *out = disc;
  return 0;
}

typedef int (*SetSaveOptionsFn)(VipsOperation *operation, SaveParams *params);

static int save_target(const char *operationName, SaveParams *params,
                       VipsTargetCustom *target,
                       SetSaveOptionsFn setSaveOptions) {
  VipsOperation *operation = vips_operation_new(operationName);
  if (!operation) {
    return 1;
  }

  if (vips_object_set(VIPS_OBJECT(operation), "in", params->inputImage,
                      "target", target, NULL)) {
    g_object_unref(operation);
    return 1;
  }

  if (setSaveOptions(operation, params)) {
    g_object_unref(operation);
    return 1;
  }

  // Build uncached, mirroring load_from_source: the unique target
  // object makes cache hits impossible.
  if (vips_object_build(VIPS_OBJECT(operation))) {
    vips_object_unref_outputs(VIPS_OBJECT(operation));
    g_object_unref(operation);
    return 1;
  }

  vips_object_unref_outputs(VIPS_OBJECT(operation));
  g_object_unref(operation);

  return 0;
}

int save_jpeg_to_target(SaveParams *params, VipsTargetCustom *target) {
  return save_target("jpegsave_target", params, target, set_jpegsave_options);
}

int save_png_to_target(SaveParams *params, VipsTargetCustom *target) {
  return save_target("pngsave_target", params, target, set_pngsave_options);
}

int save_webp_to_target(SaveParams *params, VipsTargetCustom *target) {
  return save_target("webpsave_target", params, target, set_webpsave_options);
}

int save_heif_to_target(SaveParams *params, VipsTargetCustom *target) {
  return save_target("heifsave_target", params, target, set_heifsave_options);
}

// No save_tiff_to_target: libtiff requires seekable output, so the Go
// side always encodes TIFF through the buffer path.

int save_gif_to_target(SaveParams *params, VipsTargetCustom *target) {
  return save_target("gifsave_target", params, target, set_gifsave_options);
}

void clear_source(VipsSourceCustom **source) {
  if (source && *source) {
    g_object_unref(*source);
    *source = NULL;
  }
}

void clear_target(VipsTargetCustom **target) {
  if (target && *target) {
    g_object_unref(*target);
    *target = NULL;
  }
}
