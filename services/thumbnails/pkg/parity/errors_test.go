package parity

// 406 instead of 400: imagor runs without an http loader here.
func errorsGroup() parityGroup {
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
