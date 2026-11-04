package icc

import (
	"fmt"
)

var _ = fmt.Print

// A transformer that expands a single achromatic channel into the three
// channels of the normalized PCS
type GrayToPCS struct {
	name          string
	scale, offset [3]unit_float
}

func (c *GrayToPCS) IOSig() (int, int)                    { return 1, 3 }
func (c *GrayToPCS) Iter(f func(ChannelTransformer) bool) { f(c) }
func (c *GrayToPCS) String() string {
	return fmt.Sprintf("%s{scale: %.6v offset: %.6v}", c.name, c.scale, c.offset)
}
func (c *GrayToPCS) Transform(v, _, _ unit_float) (unit_float, unit_float, unit_float) {
	return v*c.scale[0] + c.offset[0], v*c.scale[1] + c.offset[1], v*c.scale[2] + c.offset[2]
}
func (c *GrayToPCS) TransformGeneral(o, i []unit_float) {
	o[0], o[1], o[2] = c.Transform(i[0], 0, 0)
}

// The gray value is the luminance, Y relative to D50 in normalized XYZ
func NewGrayToXYZ() *GrayToPCS {
	const s = MAX_ENCODEABLE_XYZ_INVERSE
	return &GrayToPCS{name: "GrayToXYZ", scale: [3]unit_float{lcms_d50.X * s, lcms_d50.Y * s, lcms_d50.Z * s}}
}

// The gray value is L* with a* = b* = 0 in normalized LAB
func NewGrayToLAB() *GrayToPCS {
	return &GrayToPCS{name: "GrayToLAB", scale: [3]unit_float{1, 0, 0}, offset: [3]unit_float{0, 128. / 255, 128. / 255}}
}

// A transformer that extracts a single achromatic channel from the three
// channels of the normalized PCS
type PCSToGray struct {
	name    string
	weights [3]unit_float
}

func (c *PCSToGray) IOSig() (int, int)                    { return 3, 1 }
func (c *PCSToGray) Iter(f func(ChannelTransformer) bool) { f(c) }
func (c *PCSToGray) String() string                       { return fmt.Sprintf("%s{%.6v}", c.name, c.weights) }
func (c *PCSToGray) Transform(r, g, b unit_float) (unit_float, unit_float, unit_float) {
	return c.weights[0]*r + c.weights[1]*g + c.weights[2]*b, 0, 0
}
func (c *PCSToGray) TransformGeneral(o, i []unit_float) {
	o[0], _, _ = c.Transform(i[0], i[1], i[2])
}

// See BuildGrayOutputPipeline() in cmsio1.c
func NewXYZToGray() *PCSToGray {
	return &PCSToGray{name: "XYZToGray", weights: [3]unit_float{0, MAX_ENCODEABLE_XYZ * lcms_d50.Y, 0}}
}

func NewLABToGray() *PCSToGray {
	return &PCSToGray{name: "LABToGray", weights: [3]unit_float{1, 0, 0}}
}

// See section F.2 of ICC.1-2022-05.pdf and BuildGrayInputMatrixPipeline() and
// BuildGrayOutputPipeline() in cmsio1.c of lcms
// Note that the PCS illuminant, not the media white point, is used as per
// section F.2, adaptation to the media white point is done only for the
// absolute colorimetric intent, as for all other profiles.
func (p *Profile) create_gray_trc_transformer(forward bool, pipeline *Pipeline) (err error) {
	c, err := p.TagTable.load_curve_tag(GrayTRCTagSignature)
	if err != nil {
		return err
	}
	var to_pcs, from_pcs ChannelTransformer
	switch p.Header.ProfileConnectionSpace {
	case ColorSpaceXYZ:
		to_pcs, from_pcs = NewGrayToXYZ(), NewXYZToGray()
	case ColorSpaceLab:
		to_pcs, from_pcs = NewGrayToLAB(), NewLABToGray()
	default:
		return fmt.Errorf("monochrome matrix/TRC based profile using unsupported PCS color space: %v", p.Header.ProfileConnectionSpace)
	}
	if forward {
		pipeline.Append(NewCurveTransformer("GrayTRC", c), to_pcs)
	} else {
		pipeline.Append(from_pcs, NewInverseCurveTransformer("GrayTRC", c))
	}
	return nil
}
