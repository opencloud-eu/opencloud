// https://libvips.github.io/libvips/API/current/VipsImage.html

#include <stdlib.h>
#include <vips/vips.h>

int has_alpha_channel(VipsImage *image);

void clear_image(VipsImage **image);

VipsImage *create_image_from_memory_copy(const void *data, size_t size,
                                          int width, int height, int bands,
                                          VipsBandFormat format);
