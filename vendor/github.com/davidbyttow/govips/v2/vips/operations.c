// operations.c - Hand-written C bridge functions for libvips operations

#include "lang.h"
#include "operations.h"

#include <unistd.h>

static int is_16bit(VipsInterpretation interpretation);

// Arithmetic

int find_trim(VipsImage *in, int *left, int *top, int *width, int *height,
              double threshold, double r, double g, double b) {

  if (in->Type == VIPS_INTERPRETATION_RGB16 || in->Type == VIPS_INTERPRETATION_GREY16) {
    r = 65535 * r / 255;
    g = 65535 * g / 255;
    b = 65535 * b / 255;
  }

  double background[3] = {r, g, b};
  VipsArrayDouble *vipsBackground = vips_array_double_new(background, 3);

  int code = vips_find_trim(in, left, top, width, height, "threshold", threshold, "background", vipsBackground, NULL);

  vips_area_unref(VIPS_AREA(vipsBackground));
  return code;
}

int getpoint(VipsImage *in, double **vector, int n, int x, int y) {
  return vips_getpoint(in, vector, &n, x, y, NULL);
}

int minOp(VipsImage *in, double *out, int *x, int *y, int size) {
  return vips_min(in, out, "x", x, "y", y, "size", size, NULL);
}

// Color

int is_colorspace_supported(VipsImage *in) {
  return vips_colourspace_issupported(in) ? 1 : 0;
}

int to_colorspace(VipsImage *in, VipsImage **out, VipsInterpretation space) {
  return vips_colourspace(in, out, space, NULL);
}

// https://libvips.github.io/libvips/API/8.6/libvips-colour.html#vips-icc-transform
int icc_transform(VipsImage *in, VipsImage **out, const char *output_profile, const char *input_profile, VipsIntent intent,
	int depth, gboolean embedded) {
	return vips_icc_transform(
    	in, out, output_profile,
    	"input_profile", input_profile ? input_profile : "none",
    	"intent", intent,
    	"depth", depth ? depth : 8,
    	"embedded", embedded,
    	NULL);
}

// Conversion

int embed_image(VipsImage *in, VipsImage **out, int left, int top, int width,
                int height, int extend) {
  return vips_embed(in, out, left, top, width, height, "extend", extend, NULL);
}

int embed_image_background(VipsImage *in, VipsImage **out, int left, int top, int width,
                int height, double r, double g, double b, double a) {

  double background[3] = {r, g, b};
  double backgroundRGBA[4] = {r, g, b, a};

  VipsArrayDouble *vipsBackground;

  if (in->Bands <= 3) {
    vipsBackground = vips_array_double_new(background, 3);
  } else {
    vipsBackground = vips_array_double_new(backgroundRGBA, 4);
  }

  int code = vips_embed(in, out, left, top, width, height,
    "extend", VIPS_EXTEND_BACKGROUND, "background", vipsBackground, NULL);

  vips_area_unref(VIPS_AREA(vipsBackground));
  return code;
}

int embed_multi_page_image(VipsImage *in, VipsImage **out, int left, int top, int width,
                         int height, int extend) {
  VipsObject *base = VIPS_OBJECT(vips_image_new());
  int page_height = vips_image_get_page_height(in);
  int in_width = in->Xsize;
  int n_pages = in->Ysize / page_height;

  VipsImage **page = (VipsImage **) vips_object_local_array(base, n_pages);
  VipsImage **embedded_page = (VipsImage **) vips_object_local_array(base, n_pages);
  VipsImage **copy = (VipsImage **) vips_object_local_array(base, 1);

  // split image into cropped frames
  for (int i = 0; i < n_pages; i++) {
    if (
      vips_extract_area(in, &page[i], 0, page_height * i, in_width, page_height, NULL) ||
      vips_embed(page[i], &embedded_page[i], left, top, width, height, "extend", extend, NULL)
    ) {
      g_object_unref(base);
      return -1;
    }
  }
  // reassemble frames and set page height
  // copy before modifying metadata
  if(
    vips_arrayjoin(embedded_page, &copy[0], n_pages, "across", 1, NULL) ||
    vips_copy(copy[0], out, NULL)
  ) {
    g_object_unref(base);
    return -1;
  }
  vips_image_set_int(*out, VIPS_META_PAGE_HEIGHT, height);
  g_object_unref(base);
  return 0;
}

int embed_multi_page_image_background(VipsImage *in, VipsImage **out, int left, int top, int width,
                                   int height, double r, double g, double b, double a) {
  double background[3] = {r, g, b};
  double backgroundRGBA[4] = {r, g, b, a};

  VipsArrayDouble *vipsBackground;

  if (in->Bands <= 3) {
    vipsBackground = vips_array_double_new(background, 3);
  } else {
    vipsBackground = vips_array_double_new(backgroundRGBA, 4);
  }
  VipsObject *base = VIPS_OBJECT(vips_image_new());
  int page_height = vips_image_get_page_height(in);
  int in_width = in->Xsize;
  int n_pages = in->Ysize / page_height;

  VipsImage **page = (VipsImage **) vips_object_local_array(base, n_pages);
  VipsImage **embedded_page = (VipsImage **) vips_object_local_array(base, n_pages);
  VipsImage **copy = (VipsImage **) vips_object_local_array(base, 1);

  // split image into cropped frames
  for (int i = 0; i < n_pages; i++) {
    if (
      vips_extract_area(in, &page[i], 0, page_height * i, in_width, page_height, NULL) ||
      vips_embed(page[i], &embedded_page[i], left, top, width, height,
          "extend", VIPS_EXTEND_BACKGROUND, "background", vipsBackground, NULL)
    ) {
      vips_area_unref(VIPS_AREA(vipsBackground));
      g_object_unref(base);
      return -1;
    }
  }
  // reassemble frames and set page height
  // copy before modifying metadata
  if(
    vips_arrayjoin(embedded_page, &copy[0], n_pages, "across", 1, NULL) ||
    vips_copy(copy[0], out, NULL)
  ) {
    vips_area_unref(VIPS_AREA(vipsBackground));
    g_object_unref(base);
    return -1;
  }
  vips_image_set_int(*out, VIPS_META_PAGE_HEIGHT, height);
  vips_area_unref(VIPS_AREA(vipsBackground));
  g_object_unref(base);
  return 0;
}


int similarity(VipsImage *in, VipsImage **out, double scale, double angle,
               double r, double g, double b, double a, double idx, double idy,
               double odx, double ody) {
  if (is_16bit(in->Type)) {
    r = 65535 * r / 255;
    g = 65535 * g / 255;
    b = 65535 * b / 255;
    a = 65535 * a / 255;
  }

  double background[3] = {r, g, b};
  double backgroundRGBA[4] = {r, g, b, a};

  VipsArrayDouble *vipsBackground;

  // Ignore the alpha channel if the image doesn't have one
  if (in->Bands <= 3) {
    vipsBackground = vips_array_double_new(background, 3);
  } else {
    vipsBackground = vips_array_double_new(backgroundRGBA, 4);
  }

  int code = vips_similarity(in, out, "scale", scale, "angle", angle,
                             "background", vipsBackground, "idx", idx, "idy",
                             idy, "odx", odx, "ody", ody, NULL);

  vips_area_unref(VIPS_AREA(vipsBackground));
  return code;
}

int crop(VipsImage *in, VipsImage **out, int left, int top,
              int width, int height) {
  // resolve image pages
  int page_height = vips_image_get_page_height(in);
  int n_pages = in->Ysize / page_height;
  if (n_pages <= 1) {
    return vips_crop(in, out, left, top, width, height, NULL);
  }

  int in_width = in->Xsize;
  VipsObject *base = VIPS_OBJECT(vips_image_new());
  VipsImage **page = (VipsImage **) vips_object_local_array(base, n_pages);
  VipsImage **cropped_page = (VipsImage **) vips_object_local_array(base, n_pages);
  VipsImage **copy = (VipsImage **) vips_object_local_array(base, 1);
  // split image into cropped frames
  for (int i = 0; i < n_pages; i++) {
    if (
      vips_extract_area(in, &page[i], 0, page_height * i, in_width, page_height, NULL) ||
      vips_crop(page[i], &cropped_page[i], left, top, width, height, NULL)
    ) {
      g_object_unref(base);
      return -1;
    }
  }

  // reassemble frames and set page height
  // copy before modifying metadata
  if(
    vips_arrayjoin(cropped_page, &copy[0], n_pages, "across", 1, NULL) ||
    vips_copy(copy[0], out, NULL)
  ) {
    g_object_unref(base);
    return -1;
  }
  vips_image_set_int(*out, VIPS_META_PAGE_HEIGHT, height);
  g_object_unref(base);
  return 0;
}

static int is_16bit(VipsInterpretation interpretation) {
  return interpretation == VIPS_INTERPRETATION_RGB16 ||
         interpretation == VIPS_INTERPRETATION_GREY16;
}

int composite_image(VipsImage **in, VipsImage **out, int n, int *mode, int *x,
                    int *y) {
  VipsArrayInt *xs = vips_array_int_new(x, n - 1);
  VipsArrayInt *ys = vips_array_int_new(y, n - 1);

  int code = vips_composite(in, out, n, mode, "x", xs, "y", ys, NULL);

  vips_area_unref(VIPS_AREA(xs));
  vips_area_unref(VIPS_AREA(ys));
  return code;
}

int join(VipsImage *in1, VipsImage *in2, VipsImage **out, int direction) {
  return vips_join(in1, in2, out, direction, NULL);
}

int add_alpha(VipsImage *in, VipsImage **out) {
  return vips_addalpha(in, out, NULL);
}

// Create

// https://libvips.github.io/libvips/API/current/libvips-create.html#vips-text
int text(VipsImage **out, TextOptions *o) {
  return vips_text(out, o->Text, "font", o->Font, "width", o->Width, "height", o->Height, "align", o->Align,
  "dpi", o->DPI, "rgba", o->RGBA, "justify", o->Justify, "spacing", o->Spacing, "wrap", o->Wrap, NULL);
}

// Draw

int draw_rect(VipsImage *in, double r, double g, double b, double a, int left,
              int top, int width, int height, int fill) {
  if (is_16bit(in->Type)) {
    r = 65535 * r / 255;
    g = 65535 * g / 255;
    b = 65535 * b / 255;
    a = 65535 * a / 255;
  }

  double background[3] = {r, g, b};
  double backgroundRGBA[4] = {r, g, b, a};

  if (in->Bands <= 3) {
    return vips_draw_rect(in, background, 3, left, top, width, height, "fill",
                          fill, NULL);
  } else {
    return vips_draw_rect(in, backgroundRGBA, 4, left, top, width, height,
                          "fill", fill, NULL);
  }
}

// Header

unsigned long has_icc_profile(VipsImage *in) {
  return vips_image_get_typeof(in, VIPS_META_ICC_NAME);
}

unsigned long get_icc_profile(VipsImage *in, const void **data,
                              size_t *dataLength) {
  return image_get_blob(in, VIPS_META_ICC_NAME, data, dataLength);
}

gboolean remove_icc_profile(VipsImage *in) {
  return vips_image_remove(in, VIPS_META_ICC_NAME);
}

unsigned long has_iptc(VipsImage *in) {
  return vips_image_get_typeof(in, VIPS_META_IPTC_NAME);
}

char **image_get_fields(VipsImage *in) { return vips_image_get_fields(in); }

void image_set_string(VipsImage *in, const char *name, const char *str) {
  vips_image_set_string(in, name, str);
}

unsigned long image_get_string(VipsImage *in, const char *name,
                               const char **out) {
  return vips_image_get_string(in, name, out);
}

unsigned long image_get_as_string(VipsImage *in, const char *name, char **out) {
  return vips_image_get_as_string(in, name, out);
}

void remove_field(VipsImage *in, char *field) { vips_image_remove(in, field); }

int get_meta_orientation(VipsImage *in) {
  int orientation = 0;
  if (vips_image_get_typeof(in, VIPS_META_ORIENTATION) != 0) {
    vips_image_get_int(in, VIPS_META_ORIENTATION, &orientation);
  }

  return orientation;
}

void remove_meta_orientation(VipsImage *in) {
  vips_image_remove(in, VIPS_META_ORIENTATION);
}

void set_meta_orientation(VipsImage *in, int orientation) {
  vips_image_set_int(in, VIPS_META_ORIENTATION, orientation);
}

// https://libvips.github.io/libvips/API/current/libvips-header.html#vips-image-get-n-pages
int get_image_n_pages(VipsImage *in) {
  int n_pages = 0;
  n_pages = vips_image_get_n_pages(in);
  return n_pages;
}

void set_image_n_pages(VipsImage *in, int n_pages) {
  vips_image_set_int(in, VIPS_META_N_PAGES, n_pages);
}

// https://www.libvips.org/API/current/libvips-header.html#vips-image-get-page-height
int get_page_height(VipsImage *in) {
  int page_height = 0;
  page_height = vips_image_get_page_height(in);
  return page_height;
}

void set_page_height(VipsImage *in, int height) {
  vips_image_set_int(in, VIPS_META_PAGE_HEIGHT, height);
}

int get_meta_loader(const VipsImage *in, const char **out) {
  return vips_image_get_string(in, VIPS_META_LOADER, out);
}

int get_background(VipsImage *in, double **out, int *n) {
  return vips_image_get_array_double(in, "background", out, n);
}

int get_image_delay(VipsImage *in, int **out) {
  return vips_image_get_array_int(in, "delay", out, NULL);
}

void set_image_delay(VipsImage *in, const int *array, int n) {
  return vips_image_set_array_int(in, "delay", array, n);
}

int get_image_loop(VipsImage *in) {
  int loop = 0;
  if (vips_image_get_typeof(in, "loop") != 0) {
    vips_image_get_int(in, "loop", &loop);
  }
  return loop;
}

void set_image_loop(VipsImage *in, int loop) {
  vips_image_set_int(in, "loop", loop);
}

void image_set_double(VipsImage *in, const char *name, double i) {
  vips_image_set_double(in, name, i);
}

unsigned long image_get_double(VipsImage *in, const char *name, double *out) {
  return vips_image_get_double(in, name, out);
}

void image_set_int(VipsImage *in, const char *name, int i) {
  vips_image_set_int(in, name, i);
}

unsigned long image_get_int(VipsImage *in, const char *name, int *out) {
  return vips_image_get_int(in, name, out);
}

void image_set_blob(VipsImage *in, const char *name, const void *data,
                    size_t dataLength) {
  vips_image_set_blob_copy(in, name, data, dataLength);
}

unsigned long image_get_blob(VipsImage *in, const char *name, const void **data,
                             size_t *dataLength) {
  if (vips_image_get_typeof(in, name) == 0) {
    return 0;
  }

  if (vips_image_get_blob(in, name, data, dataLength)) {
    return -1;
  }

  return 0;
}

// Label

int label(VipsImage *in, VipsImage **out, LabelOptions *o) {
  double ones[3] = {1, 1, 1};
  VipsImage *base = vips_image_new();
  VipsImage **t = (VipsImage **)vips_object_local_array(VIPS_OBJECT(base), 9);
  if (vips_text(&t[0], o->Text, "font", o->Font, "width", o->Width, "height",
                o->Height, "align", o->Align, NULL) ||
      vips_linear1(t[0], &t[1], o->Opacity, 0.0, NULL) ||
      vips_cast(t[1], &t[2], VIPS_FORMAT_UCHAR, NULL) ||
      vips_embed(t[2], &t[3], o->OffsetX, o->OffsetY, t[2]->Xsize + o->OffsetX,
                 t[2]->Ysize + o->OffsetY, NULL)) {
    g_object_unref(base);
    return 1;
  }
  if (vips_black(&t[4], 1, 1, NULL) ||
      vips_linear(t[4], &t[5], ones, o->Color, 3, NULL) ||
      vips_cast(t[5], &t[6], VIPS_FORMAT_UCHAR, NULL) ||
      vips_copy(t[6], &t[7], "interpretation", in->Type, NULL) ||
      vips_embed(t[7], &t[8], 0, 0, in->Xsize, in->Ysize, "extend",
                 VIPS_EXTEND_COPY, NULL)) {
    g_object_unref(base);
    return 1;
  }
  if (vips_ifthenelse(t[3], t[8], in, out, "blend", TRUE, NULL)) {
    g_object_unref(base);
    return 1;
  }
  g_object_unref(base);
  return 0;
}

// Resample

int resize_image(VipsImage *in, VipsImage **out, double scale, gdouble vscale,
                 int kernel) {
  if (vscale > 0) {
    return vips_resize(in, out, scale, "vscale", vscale, "kernel", kernel,
                       NULL);
  }

  return vips_resize(in, out, scale, "kernel", kernel, NULL);
}

int thumbnail(const char *filename, VipsImage **out,
                    int width, int height, int crop, int size) {
  return vips_thumbnail(filename, out, width, "height", height,
                              "crop", crop, "size", size, NULL);
}

int thumbnail_image(VipsImage *in, VipsImage **out, int width, int height,
                    int crop, int size) {
  return vips_thumbnail_image(in, out, width, "height", height, "crop", crop,
                              "size", size, NULL);
}

int thumbnail_buffer_with_option(void *buf, size_t len, VipsImage **out,
                    int width, int height, int crop, int size,
                    const char *option_string) {
  return vips_thumbnail_buffer(buf, len, out, width, "height", height,
                              "crop", crop, "size", size,
                              "option_string", option_string, NULL);
}

int thumbnail_buffer(void *buf, size_t len, VipsImage **out,
                    int width, int height, int crop, int size) {
  return vips_thumbnail_buffer(buf, len, out, width, "height", height,
                              "crop", crop, "size", size, NULL);
}
