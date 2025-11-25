package icc

import (
	"fmt"

	"github.com/kovidgoyal/imaging/colorconv"
)

var _ = fmt.Println

const MAX_ENCODEABLE_XYZ = 1.0 + 32767.0/32768.0
const MAX_ENCODEABLE_XYZ_INVERSE = 1.0 / (MAX_ENCODEABLE_XYZ)
const LAB_MFT2_ENCODING_CORRECTION = 65535.0 / 65280.0
const LAB_MFT2_ENCODING_CORRECTION_INVERSE = 65280.0 / 65535.0

func tg33(t func(r, g, b unit_float) (x, y, z unit_float), o, i []unit_float) {
	o[0], o[1], o[2] = t(i[0], i[1], i[2])
}

type Scaling struct {
	name string
	s    unit_float
}

func (n *Scaling) String() string                       { return fmt.Sprintf("%s{%.6v}", n.name, n.s) }
func (n *Scaling) IOSig() (int, int)                    { return 3, 3 }
func (n *Scaling) Iter(f func(ChannelTransformer) bool) { f(n) }
func (m *Scaling) Transform(x, y, z unit_float) (unit_float, unit_float, unit_float) {
	return x * m.s, y * m.s, z * m.s
}
func (m *Scaling) AsMatrix3() *Matrix3 { return NewScalingMatrix3(m.s) }

func (m *Scaling) TransformGeneral(o, i []unit_float) { tg33(m.Transform, o, i) }

func NewScaling(name string, s unit_float) *Scaling { return &Scaling{name, s} }

type Scaling4 struct {
	name string
	s    unit_float
}

func (n *Scaling4) String() string                       { return fmt.Sprintf("%s{%.6v}", n.name, n.s) }
func (n *Scaling4) IOSig() (int, int)                    { return 4, 4 }
func (n *Scaling4) Iter(f func(ChannelTransformer) bool) { f(n) }
func (m *Scaling4) Transform(x, y, z unit_float) (unit_float, unit_float, unit_float) {
	return x * m.s, y * m.s, z * m.s
}
func (m *Scaling4) TransformGeneral(o, i []unit_float) {
	for x := range 4 {
		o[x] = m.s * i[x]
	}
}

// A transformer to convert normalized [0,1] values to the [0,1.99997]
// (u1Fixed15Number) values used by ICC XYZ PCS space
func NewNormalizedToXYZ() *Scaling { return &Scaling{"NormalizedToXYZ", MAX_ENCODEABLE_XYZ} }
func NewXYZToNormalized() *Scaling { return &Scaling{"XYZToNormalized", MAX_ENCODEABLE_XYZ_INVERSE} }

// A transformer that converts from the legacy LAB encoding used in the obsolete lut16type (mft2) tags
func NewLABFromMFT2() *Scaling { return &Scaling{"LABFromMFT2", LAB_MFT2_ENCODING_CORRECTION} }
func NewLABToMFT2() *Scaling   { return &Scaling{"LABToMFT2", LAB_MFT2_ENCODING_CORRECTION_INVERSE} }

// A transformer to convert normalized [0,1] to the LAB co-ordinate system
// used by ICC PCS LAB profiles [0-100], [-128, 127]
type NormalizedToLAB int

func (n NormalizedToLAB) String() string                        { return "NormalizedToLAB" }
func (n NormalizedToLAB) IOSig() (int, int)                     { return 3, 3 }
func (n *NormalizedToLAB) Iter(f func(ChannelTransformer) bool) { f(n) }
func (m *NormalizedToLAB) Transform(x, y, z unit_float) (unit_float, unit_float, unit_float) {
	// See PackLabDoubleFromFloat in lcms source code
	return x * 100, (y*255 - 128), (z*255 - 128)
}

func (m *NormalizedToLAB) TransformGeneral(o, i []unit_float) { tg33(m.Transform, o, i) }

func NewNormalizedToLAB() *NormalizedToLAB {
	x := NormalizedToLAB(0)
	return &x
}

type LABToNormalized int

func (n LABToNormalized) String() string                        { return "LABToNormalized" }
func (n LABToNormalized) IOSig() (int, int)                     { return 3, 3 }
func (n *LABToNormalized) Iter(f func(ChannelTransformer) bool) { f(n) }
func (m *LABToNormalized) Transform(x, y, z unit_float) (unit_float, unit_float, unit_float) {
	// See PackLabDoubleFromFloat in lcms source code
	return x * (1. / 100), (y*(1./255) + 128./255), (z*(1./255) + 128./255)
}

func (m *LABToNormalized) TransformGeneral(o, i []unit_float) { tg33(m.Transform, o, i) }

func NewLABToNormalized() *LABToNormalized {
	x := LABToNormalized(0)
	return &x
}

type BlackPointCorrection struct {
	scale, offset XYZType
}

func (n BlackPointCorrection) IOSig() (int, int) { return 3, 3 }
func (n *BlackPointCorrection) Iter(f func(ChannelTransformer) bool) {
	m := &Matrix3{{n.scale.X, 0, 0}, {0, n.scale.Y, 0}, {0, 0, n.scale.Z}}
	if !is_identity_matrix(m) {
		if !f(m) {
			return
		}
	}
	t := &Translation{n.offset.X, n.offset.Y, n.offset.Z}
	if !t.Empty() {
		f(t)
	}
}

func NewBlackPointCorrection(in_whitepoint, in_blackpoint, out_blackpoint XYZType) *BlackPointCorrection {
	tx := in_blackpoint.X - in_whitepoint.X
	ty := in_blackpoint.Y - in_whitepoint.Y
	tz := in_blackpoint.Z - in_whitepoint.Z
	ans := BlackPointCorrection{}

	ans.scale.X = (out_blackpoint.X - in_whitepoint.X) / tx
	ans.scale.Y = (out_blackpoint.Y - in_whitepoint.Y) / ty
	ans.scale.Z = (out_blackpoint.Z - in_whitepoint.Z) / tz

	ans.offset.X = -in_whitepoint.X * (out_blackpoint.X - in_blackpoint.X) / tx
	ans.offset.Y = -in_whitepoint.Y * (out_blackpoint.Y - in_blackpoint.Y) / ty
	ans.offset.Z = -in_whitepoint.Z * (out_blackpoint.Z - in_blackpoint.Z) / tz
	ans.offset.X *= MAX_ENCODEABLE_XYZ_INVERSE
	ans.offset.Y *= MAX_ENCODEABLE_XYZ_INVERSE
	ans.offset.Z *= MAX_ENCODEABLE_XYZ_INVERSE

	return &ans
}

func (c *BlackPointCorrection) String() string {
	return fmt.Sprintf("BlackPointCorrection{scale: %v offset: %v}", c.scale, c.offset)
}

func (c *BlackPointCorrection) Transform(r, g, b unit_float) (unit_float, unit_float, unit_float) {
	return c.scale.X*r + c.offset.X, c.scale.Y*g + c.offset.Y, c.scale.Z*b + c.offset.Z
}
func (m *BlackPointCorrection) TransformGeneral(o, i []unit_float) { tg33(m.Transform, o, i) }

type LABtosRGB struct {
	c *colorconv.ConvertColor
	t func(l, a, b unit_float) (x, y, z unit_float)
}

func NewLABtosRGB(whitepoint XYZType, clamp, map_gamut bool) *LABtosRGB {
	c := colorconv.NewConvertColor(whitepoint.X, whitepoint.Y, whitepoint.Z, 1)
	if clamp {
		if map_gamut {
			return &LABtosRGB{c, c.LabToSRGB}
		}
		return &LABtosRGB{c, c.LabToSRGBClamp}
	}
	return &LABtosRGB{c, c.LabToSRGBNoGamutMap}
}

func (c LABtosRGB) Transform(l, a, b unit_float) (unit_float, unit_float, unit_float) {
	return c.t(l, a, b)
}
func (m LABtosRGB) TransformGeneral(o, i []unit_float)   { tg33(m.Transform, o, i) }
func (n LABtosRGB) IOSig() (int, int)                    { return 3, 3 }
func (n LABtosRGB) String() string                       { return fmt.Sprintf("%T%s", n, n.c.String()) }
func (n LABtosRGB) Iter(f func(ChannelTransformer) bool) { f(n) }

type UniformFunctionTransformer struct {
	name string
	f    func(unit_float) unit_float
}

func (n UniformFunctionTransformer) IOSig() (int, int)                     { return 3, 3 }
func (n UniformFunctionTransformer) String() string                        { return n.name }
func (n *UniformFunctionTransformer) Iter(f func(ChannelTransformer) bool) { f(n) }
func (c *UniformFunctionTransformer) Transform(x, y, z unit_float) (unit_float, unit_float, unit_float) {
	return c.f(x), c.f(y), c.f(z)
}
func (c *UniformFunctionTransformer) TransformGeneral(o, i []unit_float) {
	for k, x := range i {
		o[k] = c.f(x)
	}
}
func NewUniformFunctionTransformer(name string, f func(unit_float) unit_float) *UniformFunctionTransformer {
	return &UniformFunctionTransformer{name, f}
}

type XYZtosRGB struct {
	c *colorconv.ConvertColor
	t func(l, a, b unit_float) (x, y, z unit_float)
}

func NewXYZtosRGB(whitepoint XYZType, clamp, map_gamut bool) *XYZtosRGB {
	c := colorconv.NewConvertColor(whitepoint.X, whitepoint.Y, whitepoint.Z, 1)
	if clamp {
		if map_gamut {
			return &XYZtosRGB{c, c.XYZToSRGB}
		}
		return &XYZtosRGB{c, c.XYZToSRGBNoGamutMap}
	}
	return &XYZtosRGB{c, c.XYZToSRGBNoClamp}
}

func (n *XYZtosRGB) AddPreviousMatrix(m Matrix3) {
	n.c.AddPreviousMatrix(m[0], m[1], m[2])
}

func (c *XYZtosRGB) Transform(l, a, b unit_float) (unit_float, unit_float, unit_float) {
	return c.t(l, a, b)
}
func (m *XYZtosRGB) TransformGeneral(o, i []unit_float)   { tg33(m.Transform, o, i) }
func (n *XYZtosRGB) IOSig() (int, int)                    { return 3, 3 }
func (n *XYZtosRGB) String() string                       { return fmt.Sprintf("%T%s", n, n.c.String()) }
func (n *XYZtosRGB) Iter(f func(ChannelTransformer) bool) { f(n) }

type LABtoXYZ struct {
	c *colorconv.ConvertColor
	t func(l, a, b unit_float) (x, y, z unit_float)
}

func NewLABtoXYZ(whitepoint XYZType) *LABtoXYZ {
	c := colorconv.NewConvertColor(whitepoint.X, whitepoint.Y, whitepoint.Z, 1)
	return &LABtoXYZ{c, c.LabToXYZ}
}

func (c *LABtoXYZ) Transform(l, a, b unit_float) (unit_float, unit_float, unit_float) {
	return c.t(l, a, b)
}
func (m *LABtoXYZ) TransformGeneral(o, i []unit_float)   { tg33(m.Transform, o, i) }
func (n *LABtoXYZ) IOSig() (int, int)                    { return 3, 3 }
func (n *LABtoXYZ) String() string                       { return fmt.Sprintf("%T%s", n, n.c.String()) }
func (n *LABtoXYZ) Iter(f func(ChannelTransformer) bool) { f(n) }

type XYZtoLAB struct {
	c *colorconv.ConvertColor
	t func(l, a, b unit_float) (x, y, z unit_float)
}

func NewXYZtoLAB(whitepoint XYZType) *XYZtoLAB {
	c := colorconv.NewConvertColor(whitepoint.X, whitepoint.Y, whitepoint.Z, 1)
	return &XYZtoLAB{c, c.XYZToLab}
}

func (c *XYZtoLAB) Transform(l, a, b unit_float) (unit_float, unit_float, unit_float) {
	return c.t(l, a, b)
}
func (m *XYZtoLAB) TransformGeneral(o, i []unit_float)   { tg33(m.Transform, o, i) }
func (n *XYZtoLAB) IOSig() (int, int)                    { return 3, 3 }
func (n *XYZtoLAB) String() string                       { return fmt.Sprintf("%T%s", n, n.c.String()) }
func (n *XYZtoLAB) Iter(f func(ChannelTransformer) bool) { f(n) }
