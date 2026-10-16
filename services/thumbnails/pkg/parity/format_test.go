package parity

func formatGroup() parityGroup {
	return parityGroup{
		name: "format",
		cases: []parityCase{
			{
				title: "png output from a jpeg source", path: "/unsafe/50x50/filters:format(png)/",
				source: jpegSource(300, 200), want: "200 50x50 png",
			},
			{
				title: "jpeg output from a png source", path: "/unsafe/50x50/filters:format(jpeg)/",
				source: pngSource(300, 200), want: "200 50x50 jpeg",
			},
			{
				title: "gif output from a png source", path: "/unsafe/50x50/filters:format(gif)/",
				source: pngSource(300, 200), want: "200 50x50 gif",
			},
			{
				title: "webp output from a png source", path: "/unsafe/50x50/filters:format(webp)/",
				source: pngSource(300, 200), want: "200 50x50 webp",
				generatorOverrides: map[string]override{
					"imaging": {answers: "406", because: "the pure Go build has no webp encoder"},
				},
			},
			{
				title: "no format filter keeps a png a png", path: "/unsafe/50x50/filters:no_upscale()/",
				source: pngSource(300, 200), want: "200 50x50 png",
			},
			{
				title: "no format filter keeps a gif a gif", path: "/unsafe/50x50/filters:no_upscale()/",
				source: animatedGIFSource(300, 200, 3), want: "200 50x50 gif frames=3",
			},
			{
				title: "no filters segment at all", path: "/unsafe/50x50/",
				source: pngSource(300, 200), want: "200 50x50 png",
			},
			{
				title: "jpg, the name webdav sends for a jpeg", path: "/unsafe/50x50/filters:format(jpg)/",
				source: pngSource(300, 200), want: "200 50x50 jpeg",
			},
			{
				title: "an unknown format name", path: "/unsafe/50x50/filters:format(bogus)/",
				source: pngSource(300, 200), want: "400",
				generatorOverrides: map[string]override{
					"imagor": {answers: "200 50x50 png", because: "imagor drops a filter it does not know, the built-in refuses it"},
				},
			},
		},
	}
}
