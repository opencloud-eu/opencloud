package parity

// both sides are configured with the same edge limit, see imagortest.Args.
//
// The sources are PNGs on purpose. libvips loads a JPEG at a scale close to the
// requested box, so the limit never sees the full image and a 20000 pixel wide
// JPEG passes a 2048 limit in imagor, where this service refuses it on the
// header. PNG has no such loader option, so both sides see the same picture.
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
