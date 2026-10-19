package parity

// a flattened gif is silent data loss that no dimension check would catch.
func animationGroup() parityGroup {
	return parityGroup{
		name: "animation",
		cases: []parityCase{
			{
				title: "an animated gif keeps its frames", path: "/unsafe/50x50/filters:format(gif)/",
				source: animatedGIFSource(300, 200, 3), want: "200 50x50 gif frames=3",
			},
			{
				title: "fit-in leaves an animated gif smaller than the box alone", path: "/unsafe/fit-in/200x200/filters:format(gif)/",
				source: animatedGIFSource(100, 80, 3), want: "200 100x80 gif frames=3",
				generatorOverrides: map[string]override{
					"imagor": {answers: "200 100x80 gif", because: "imagor drops the animation when it does not resize"},
				},
			},
			{
				title: "an animated gif flattens when the output is not a gif", path: "/unsafe/50x50/filters:format(png)/",
				source: animatedGIFSource(300, 200, 3), want: "200 50x50 png",
			},
			{
				title: "fit-in resizes every frame of a larger animation", path: "/unsafe/fit-in/50x50/filters:format(gif)/",
				source: flatGIFSource(200, 100, 4), want: "200 50x25 gif frames=4",
			},
			{
				// this used to fail the encode with "image block is out of bounds"
				title: "stretch past the source grows the logical screen with the frames", path: "/unsafe/stretch/200x160/filters:no_upscale():format(gif)/",
				source: flatGIFSource(100, 80, 3), want: "200 200x160 gif frames=3",
			},
			{
				title: "a free axis keeps an animation animated", path: "/unsafe/0x40/filters:format(gif)/",
				source: flatGIFSource(200, 100, 3), want: "200 80x40 gif frames=3",
			},
			{
				title: "an animation above the frame limit", path: "/unsafe/50x50/filters:format(gif)/",
				source: flatGIFSource(20, 20, 101), want: "422",
				generatorOverrides: map[string]override{
					"imagor": {answers: "200 50x50 gif frames=101", because: "imagor loads every frame unless -vips-max-animation-frames says otherwise; the built-in refuses a file above THUMBNAILS_MAX_ANIMATION_FRAMES instead of truncating it, because image/gif decodes all frames or none"},
				},
			},
		},
	}
}
