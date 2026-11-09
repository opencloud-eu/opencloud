package parity

func transparencyGroup() parityGroup {
	return parityGroup{
		name: "transparency",
		cases: []parityCase{
			{
				title:  "a transparent png keeps its transparency",
				path:   "/unsafe/fit-in/32x32/filters:format(png)/",
				source: alphaPNGSource(64), want: "200 32x32 png alpha",
			},
			// jpeg has no alpha channel
			{
				title:  "a transparent png loses its transparency as a jpeg",
				path:   "/unsafe/fit-in/32x32/filters:format(jpeg)/",
				source: alphaPNGSource(64), want: "200 32x32 jpeg",
			},
		},
	}
}
