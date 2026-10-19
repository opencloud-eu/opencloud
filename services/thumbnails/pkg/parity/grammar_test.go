package parity

// imagor's path is "/unsafe[/{operation}]/{W}x{H}[/filters:{list}]/{image}": the
// filters may be empty, the image key is the last segment and an upload loader
// has no use for it, and a side given as 0 is computed from the other one.
func grammarGroup() parityGroup {
	src := func() source { return pngSource(300, 200) }

	return parityGroup{
		name: "grammar",
		cases: []parityCase{
			{
				title: "an empty filters segment is still a filters segment", path: "/unsafe/100x100/filters:",
				source: src(), want: "200 100x100 png",
			},
			{
				title: "an empty filters segment behind an operation", path: "/unsafe/fit-in/100x100/filters:",
				source: src(), want: "200 100x67 png",
			},
			{
				title: "a filters segment without a trailing slash", path: "/unsafe/100x100/filters:format(png)",
				source: src(), want: "200 100x100 png",
			},
			{
				title: "an image key behind the box is ignored", path: "/unsafe/100x100/foo.png",
				source: src(), want: "200 100x100 png",
			},
			{
				title: "an image key behind the filters is ignored", path: "/unsafe/100x100/filters:format(png)/foo.png",
				source: src(), want: "200 100x100 png",
			},
			{
				title: "a zero width follows the height", path: "/unsafe/0x100/filters:format(png)/",
				source: src(), want: "200 150x100 png",
			},
			{
				title: "a zero height follows the width", path: "/unsafe/100x0/filters:format(png)/",
				source: src(), want: "200 100x67 png",
			},
			{
				title: "a free axis enlarges", path: "/unsafe/0x400/filters:format(png)/",
				source: src(), want: "200 600x400 png",
			},
			{
				title: "a free axis under fit-in is the same, there is nothing to crop to", path: "/unsafe/fit-in/0x100/filters:format(png)/",
				source: src(), want: "200 150x100 png",
			},
			{
				title: "a free axis under stretch is the same", path: "/unsafe/stretch/100x0/filters:format(png)/",
				source: src(), want: "200 100x67 png",
				generatorOverrides: map[string]override{
					"imagor": {answers: "200 100x66 png", because: "imagor truncates the computed side under stretch and rounds it everywhere else"},
				},
			},
			{
				title: "a zero box leaves the source alone", path: "/unsafe/0x0/filters:format(png)/",
				source: src(), want: "200 300x200 png",
			},
			{
				title: "a bare box with no tail at all", path: "/unsafe/100x100",
				source: src(), want: "200 100x100 png",
				generatorOverrides: map[string]override{
					"imagor": {answers: "200 300x200 png", because: "the last segment is imagor's image key, so a bare box is a key and nothing is resized"},
				},
			},
		},
	}
}
