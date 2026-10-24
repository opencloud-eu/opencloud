package parity

// 406 instead of 400: imagor runs without an http loader here.
func errorsGroup() parityGroup {
	noImageBody, noImageType := multipartWithoutImage()

	return parityGroup{
		name: "errors",
		cases: []parityCase{
			{
				title: "an untyped upload part", path: "/unsafe/50x50/filters:format(png)/",
				source: pngSource(300, 200), partType: "application/octet-stream",
				want: "406",
			},
			{
				title: "non-numeric dimensions", path: "/unsafe/axb/filters:format(png)/",
				source: pngSource(300, 200), want: "400",
				generatorOverrides: map[string]override{
					"imagor": {answers: "200 300x200 png", because: "imagor ignores dimensions it cannot read"},
				},
			},
			{
				title: "an unknown operation", path: "/unsafe/shrink/50x50/filters:format(png)/",
				source: pngSource(300, 200), want: "400",
				generatorOverrides: map[string]override{
					"imagor": {answers: "200 300x200 png", because: "imagor reads an unknown segment as a filename and returns the source"},
				},
			},
			{
				title: "a negative box", path: "/unsafe/-50x50/filters:format(png)/",
				source: pngSource(300, 200), want: "400",
				generatorOverrides: map[string]override{
					"imagor": {answers: "200 50x50 png", because: "a leading minus is imagor's flip marker, so it reads a 50x50 box and flips it; this service has no flip and refuses the box"},
				},
			},
			{
				title: "bytes that are not an image", path: "/unsafe/50x50/filters:format(png)/",
				source: source{name: "garbage", filename: "x.png", partType: "image/png", data: []byte("not an image")},
				want:   "406",
			},
			{
				title: "a body that is not multipart", path: "/unsafe/50x50/filters:format(png)/",
				source: pngSource(300, 200), rawBody: []byte("plain"), rawContentType: "text/plain",
				want: "400",
				generatorOverrides: map[string]override{
					"imagor": {answers: "406", because: "no loader to fall back to"},
				},
			},
			{
				title: "a dimension bomb in a jpeg header", path: "/unsafe/64x64/filters:format(png)/",
				source: jpegDimensionBomb(20000, 20000), want: "422",
				generatorOverrides: map[string]override{
					"imagor": {answers: "200 64x64 png", because: "imagor decodes the file instead of trusting the header, so the declared size never reaches its resolution guard"},
				},
			},
			{
				title: "a dimension bomb in a gif header", path: "/unsafe/64x64/filters:format(gif)/",
				source: gifDimensionBomb(), want: "422",
				generatorOverrides: map[string]override{
					"imagor": {answers: "406", because: "a header without a single frame is not an image libvips reads at all, so it never gets to the size"},
				},
			},
			{
				title: "a multipart body without the image field", path: "/unsafe/50x50/filters:format(png)/",
				source: pngSource(300, 200), rawBody: noImageBody, rawContentType: noImageType,
				want: "400",
			},
			{
				title: "a GET on a processing path", path: "/unsafe/50x50/filters:format(png)/",
				source: pngSource(300, 200), method: "GET", rawContentType: "text/plain",
				want: "405",
				generatorOverrides: map[string]override{
					"imagor": {answers: "200 non-image", because: "imagor serves an upload form there"},
				},
			},
		},
	}
}
