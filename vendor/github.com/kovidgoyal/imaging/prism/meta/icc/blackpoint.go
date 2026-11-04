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
	// The lock is not held during the computation as it can recurse via
	// CreateTransformerToDevice()
	p.blackpoints_lock.Lock()
	q := p.blackpoints[intent]
	p.blackpoints_lock.Unlock()
	if q != nil {
		return *q
	}
	defer func() {
		p.blackpoints_lock.Lock()
		p.blackpoints[intent] = &ans
		p.blackpoints_lock.Unlock()
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

// See cmsIsIntentSupported() in cmsio1.c
func (p *Profile) is_intent_supported_as_input(intent RenderingIntent) bool {
	var sig Signature
	switch intent {
	case PerceptualRenderingIntent:
		sig = AToB0TagSignature
	case RelativeColorimetricRenderingIntent, AbsoluteColorimetricRenderingIntent:
		sig = AToB1TagSignature
	case SaturationRenderingIntent:
		sig = AToB2TagSignature
	default:
		return false
	}
	return p.TagTable.Has(sig) || p.IsMatrixShaper()
}

func (p *Profile) black_point_as_darker_colorant(intent RenderingIntent, debug General_debug_callback) XYZType {
	if !p.is_intent_supported_as_input(intent) {
		return XYZType{}
	}
	bp := p.Header.DataColorSpace.BlackPoint()
	if bp == nil || (len(bp) != 1 && len(bp) != 3 && len(bp) != 4) {
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
	var out, inp [4]unit_float
	copy(inp[:], bp)
	if debug == nil {
		if len(bp) == 3 {
			l, a, b = tr.Transform(bp[0], bp[1], bp[2])
		} else {
			tr.TransformGeneral(out[:], inp[:])
			l, a, b = out[0], out[1], out[2]
		}
	} else {
		tr.TransformGeneralDebug(out[:], inp[:], debug)
		l, a, b = out[0], out[1], out[2]
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
