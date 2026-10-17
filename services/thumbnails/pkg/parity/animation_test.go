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
		},
	}
}
