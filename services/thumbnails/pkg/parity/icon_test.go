package parity

// Tika hands over an ICO for a windows executable, and its transparency has to
// survive the resize. Reading one needs a libvips with ImageMagick.
func iconGroup() parityGroup {
	return parityGroup{
		name: "icon",
		cases: []parityCase{
			{
				title:  "an icon resizes into a png that keeps its transparency",
				path:   "/unsafe/fit-in/32x32/filters:format(png)/",
				source: icoSource(64), want: "200 32x32 png alpha",
				generatorOverrides: map[string]override{
					"imagor-default": {answers: "406", because: "that image has no ICO loader, the magick variant has one"},
					"imaging":        {answers: "406", because: "the pure Go build reads no ICO"},
				},
			},
		},
	}
}
