package icc

import (
	"fmt"
)

var _ = fmt.Print

func (p *Profile) IsMatrixShaper() bool {
	h := p.TagTable.Has
	switch p.Header.DataColorSpace {
	case ColorSpaceGray:
		return h(GrayTRCTagSignature)
	case ColorSpaceRGB:
		return h(RedColorantTagSignature) && h(RedTRCTagSignature) && h(GreenColorantTagSignature) && h(GreenTRCTagSignature) && h(BlueColorantTagSignature) && h(BlueTRCTagSignature)
	default:
		return false
	}
}

func (p *Profile) BlackPoint(intent RenderingIntent, debug General_debug_callback) (ans XYZType) {
	if q := p.blackpoints[intent]; q != nil {
		return *q
	}
	defer func() {
		p.blackpoints[intent] = &ans
	}()
	if p.Header.DeviceClass == DeviceClassLink || p.Header.DeviceClass == DeviceClassAbstract || p.Header.DeviceClass == DeviceClassNamedColor {
		return
	}
	if !(intent == PerceptualRenderingIntent || intent == SaturationRenderingIntent || intent == RelativeColorimetricRenderingIntent) {
		return
	}
	if p.Header.Version.Major >= 4 && (intent == PerceptualRenderingIntent || intent == SaturationRenderingIntent) {
		if p.IsMatrixShaper() {
			return p.black_point_as_darker_colorant(RelativeColorimetricRenderingIntent, debug)
		}
		return XYZType{0.00336, 0.0034731, 0.00287}
	}
	if intent == RelativeColorimetricRenderingIntent && p.Header.DeviceClass == DeviceClassOutput && p.Header.DataColorSpace == ColorSpaceCMYK {
		return p.black_point_using_perceptual_black(debug)
	}
	return p.black_point_as_darker_colorant(intent, debug)
}

func (p *Profile) black_point_as_darker_colorant(intent RenderingIntent, debug General_debug_callback) XYZType {
	bp := p.Header.DataColorSpace.BlackPoint()
	if bp == nil || (len(bp) != 3 && len(bp) != 4) {
		return XYZType{}
	}
	tr, err := p.CreateTransformerToPCS(intent, len(bp), debug == nil)
	if err != nil {
		return XYZType{}
	}
	if p.Header.ProfileConnectionSpace == ColorSpaceXYZ {
		tr.Append(NewXYZtoLAB(p.PCSIlluminant))
	}
	var l, a, b unit_float
	if debug == nil {
		if len(bp) == 3 {
			l, a, b = tr.Transform(bp[0], bp[1], bp[2])
		} else {
			var x [4]unit_float
			tr.TransformGeneral(x[:], bp)
			l, a, b = x[0], x[1], x[2]
		}
	} else {
		var x [4]unit_float
		tr.TransformGeneralDebug(x[:], bp, debug)
		l, a, b = x[0], x[1], x[2]
	}
	a, b = 0, 0
	if l < 0 || l > 50 {
		l = 0
	}
	x, y, z := NewLABtoXYZ(p.PCSIlluminant).Transform(l, a, b)
	return XYZType{x, y, z}
}

func (p *Profile) black_point_using_perceptual_black(debug General_debug_callback) XYZType {
	dev, err := p.CreateTransformerToDevice(PerceptualRenderingIntent, false, debug == nil)
	if err != nil {
		return XYZType{}
	}
	tr, err := p.CreateTransformerToPCS(RelativeColorimetricRenderingIntent, 4, debug == nil)
	if err != nil {
		return XYZType{}
	}
	dev = dev.Weld(tr, debug == nil)
	if !dev.IsSuitableFor(3, 3) {
		return XYZType{}
	}
	lab := [4]unit_float{}
	if debug == nil {
		dev.TransformGeneral(lab[:], []unit_float{0, 0, 0, 0})
	} else {
		dev.TransformGeneralDebug(lab[:], []unit_float{0, 0, 0, 0}, debug)
	}
	l, a, b := lab[0], lab[1], lab[2]
	l = min(l, 50)
	a, b = 0, 0
	x, y, z := NewLABtoXYZ(p.PCSIlluminant).Transform(l, a, b)
	return XYZType{x, y, z}
}
