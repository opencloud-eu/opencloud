package icc

import "fmt"

const (
	PerceptualRenderingIntent           RenderingIntent = 0
	RelativeColorimetricRenderingIntent RenderingIntent = 1
	SaturationRenderingIntent           RenderingIntent = 2
	AbsoluteColorimetricRenderingIntent RenderingIntent = 3
)

type RenderingIntent uint32

func (ri RenderingIntent) String() string {
	switch ri {
	case PerceptualRenderingIntent:
		return "Perceptual"
	case RelativeColorimetricRenderingIntent:
		return "Relative colorimetric"
	case SaturationRenderingIntent:
		return "Saturation"
	case AbsoluteColorimetricRenderingIntent:
		return "Absolute colorimetric"
	default:
		return fmt.Sprintf("Unknown (%d)", ri)
	}
}
