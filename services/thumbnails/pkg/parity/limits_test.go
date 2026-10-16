package parity

// both sides are configured with the same edge limit, see imagortest.Args.
func limitsGroup() parityGroup {
	return parityGroup{
		name: "limits",
		cases: []parityCase{
			{
				title: "a source wider than the configured maximum", path: "/unsafe/50x50/filters:format(png)/",
				source: pngSource(7681, 8), want: "422",
			},
			{
				title: "a source at the configured maximum", path: "/unsafe/50x50/filters:format(png)/",
				source: pngSource(7680, 8), want: "200 50x50 png",
			},
		},
	}
}
