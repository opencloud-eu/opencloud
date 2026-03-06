// operations.h - Hand-written C bridge functions for libvips operations

#ifndef OPERATIONS_H
#define OPERATIONS_H

#include <stdlib.h>
#include <vips/vips.h>
#include <vips/foreign.h>

// Arithmetic
// https://libvips.github.io/libvips/API/current/libvips-arithmetic.html

int find_trim(VipsImage *in, int *left, int *top, int *width, int *height,
              double threshold, double r, double g, double b);
int getpoint(VipsImage *in, double **vector, int n, int x, int y);
int minOp(VipsImage *in, double *out, int *x, int *y, int size);

// Color
// https://libvips.github.io/libvips/API/current/libvips-colour.html

int is_colorspace_supported(VipsImage *in);
int to_colorspace(VipsImage *in, VipsImage **out, VipsInterpretation space);
int icc_transform(VipsImage *in, VipsImage **out, const char *output_profile,
	const char *input_profile, VipsIntent intent, int depth,
	gboolean embedded);

// Conversion
// https://libvips.github.io/libvips/API/current/libvips-conversion.html

int embed_image(VipsImage *in, VipsImage **out, int left, int top, int width,
                int height, int extend);
int embed_image_background(VipsImage *in, VipsImage **out, int left, int top,
                int width, int height, double r, double g, double b, double a);
int embed_multi_page_image(VipsImage *in, VipsImage **out, int left, int top,
                int width, int height, int extend);
int embed_multi_page_image_background(VipsImage *in, VipsImage **out, int left,
                int top, int width, int height, double r, double g, double b,
                double a);
int crop(VipsImage *in, VipsImage **out, int left, int top, int width,
         int height);
int similarity(VipsImage *in, VipsImage **out, double scale, double angle,
               double r, double g, double b, double a, double idx, double idy,
               double odx, double ody);
int composite_image(VipsImage **in, VipsImage **out, int n, int *mode, int *x,
                    int *y);
int join(VipsImage *in1, VipsImage *in2, VipsImage **out, int direction);
int add_alpha(VipsImage *in, VipsImage **out);

// Create
// https://libvips.github.io/libvips/API/current/libvips-create.html

typedef struct {
  const char *Text;
  const char *Font;
  int Width;
  int Height;
  int DPI;
  gboolean RGBA;
  gboolean Justify;
  int Spacing;
  VipsAlign Align;
  VipsTextWrap Wrap;
} TextOptions;

int text(VipsImage **out, TextOptions *o);

// Draw
// https://libvips.github.io/libvips/API/current/libvips-draw.html

int draw_rect(VipsImage *in, double r, double g, double b, double a, int left,
              int top, int width, int height, int fill);

// Header
// https://libvips.github.io/libvips/API/current/libvips-header.html

unsigned long has_icc_profile(VipsImage *in);
unsigned long get_icc_profile(VipsImage *in, const void **data,
                              size_t *dataLength);
int remove_icc_profile(VipsImage *in);

unsigned long has_iptc(VipsImage *in);
char **image_get_fields(VipsImage *in);

void image_set_string(VipsImage *in, const char *name, const char *str);
unsigned long image_get_string(VipsImage *in, const char *name,
                               const char **out);
unsigned long image_get_as_string(VipsImage *in, const char *name, char **out);

void remove_field(VipsImage *in, char *field);

int get_meta_orientation(VipsImage *in);
void remove_meta_orientation(VipsImage *in);
void set_meta_orientation(VipsImage *in, int orientation);
int get_image_n_pages(VipsImage *in);
void set_image_n_pages(VipsImage *in, int n_pages);
int get_page_height(VipsImage *in);
void set_page_height(VipsImage *in, int height);
int get_meta_loader(const VipsImage *in, const char **out);
int get_image_delay(VipsImage *in, int **out);
void set_image_delay(VipsImage *in, const int *array, int n);
int get_image_loop(VipsImage *in);
void set_image_loop(VipsImage *in, int loop);
int get_background(VipsImage *in, double **out, int *n);

void image_set_blob(VipsImage *in, const char *name, const void *data,
                    size_t dataLength);
unsigned long image_get_blob(VipsImage *in, const char *name, const void **data,
                             size_t *dataLength);

void image_set_double(VipsImage *in, const char *name, double i);
unsigned long image_get_double(VipsImage *in, const char *name, double *out);

void image_set_int(VipsImage *in, const char *name, int i);
unsigned long image_get_int(VipsImage *in, const char *name, int *out);

// Label

typedef struct {
  const char *Text;
  const char *Font;
  int Width;
  int Height;
  int OffsetX;
  int OffsetY;
  VipsAlign Align;
  int DPI;
  int Margin;
  float Opacity;
  double Color[3];
} LabelOptions;

int label(VipsImage *in, VipsImage **out, LabelOptions *o);

// Resample
// https://libvips.github.io/libvips/API/current/libvips-resample.html

int resize_image(VipsImage *in, VipsImage **out, double scale, gdouble vscale,
                 int kernel);
int thumbnail(const char *filename, VipsImage **out, int width, int height,
                    int crop, int size);
int thumbnail_image(VipsImage *in, VipsImage **out, int width, int height,
                    int crop, int size);
int thumbnail_buffer(void *buf, size_t len, VipsImage **out, int width,
                    int height, int crop, int size);
int thumbnail_buffer_with_option(void *buf, size_t len, VipsImage **out,
                    int width, int height, int crop, int size,
                    const char *option_string);

#endif // OPERATIONS_H
