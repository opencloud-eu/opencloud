package parity

// 300x200 is 3:2 and the boxes are not, so cropping and inscribing cannot be
// confused.
func resizeGroup() parityGroup {
	src := func() source { return pngSource(300, 200) }

	return parityGroup{
		name: "resize",
		cases: []parityCase{
			{
				title: "fill downscales to the exact box", path: "/unsafe/100x100/filters:format(png)/",
				source: src(), want: "200 100x100 png",
			},
			{
				title: "fill upscales a small source to the exact box", path: "/unsafe/600x600/filters:format(png)/",
				source: src(), want: "200 600x600 png",
			},
			{
				title: "fill with no_upscale leaves a small source alone", path: "/unsafe/600x600/filters:no_upscale():format(png)/",
				source: src(), want: "200 300x200 png",
			},
			{
				title: "fit-in inscribes into the box", path: "/unsafe/fit-in/100x100/filters:format(png)/",
				source: src(), want: "200 100x67 png",
			},
			// imagor only upscales fit-in with the upscale() filter
			{
				title: "fit-in never upscales", path: "/unsafe/fit-in/600x600/filters:format(png)/",
				source: src(), want: "200 300x200 png",
			},
			{
				title: "fit-in with no_upscale leaves a small source alone", path: "/unsafe/fit-in/600x600/filters:no_upscale():format(png)/",
				source: src(), want: "200 300x200 png",
			},
			{
				title: "stretch hits the box exactly", path: "/unsafe/stretch/100x400/filters:format(png)/",
				source: src(), want: "200 100x400 png",
			},
		},
	}
}
