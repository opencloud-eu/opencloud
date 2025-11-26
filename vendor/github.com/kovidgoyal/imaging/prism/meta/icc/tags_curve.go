package icc

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"strings"
	"sync"
)

type IdentityCurve int
type GammaCurve struct {
	gamma, inv_gamma unit_float
	is_one           bool
}
type PointsCurve struct {
	points, reverse_lookup   []unit_float
	max_idx, reverse_max_idx unit_float
}
type ConditionalZeroCurve struct{ g, a, b, threshold, inv_gamma, inv_a unit_float }
type ConditionalCCurve struct{ g, a, b, c, threshold, inv_gamma, inv_a unit_float }
type SplitCurve struct{ g, a, b, c, d, inv_g, inv_a, inv_c, threshold unit_float }
type ComplexCurve struct{ g, a, b, c, d, e, f, inv_g, inv_a, inv_c, threshold unit_float }
type Curve1D interface {
	Transform(x unit_float) unit_float
	InverseTransform(x unit_float) unit_float
	Prepare() error
	String() string
}

var _ Curve1D = (*IdentityCurve)(nil)
var _ Curve1D = (*GammaCurve)(nil)
var _ Curve1D = (*PointsCurve)(nil)
var _ Curve1D = (*ConditionalZeroCurve)(nil)
var _ Curve1D = (*ConditionalCCurve)(nil)
var _ Curve1D = (*SplitCurve)(nil)
var _ Curve1D = (*ComplexCurve)(nil)

type CurveTransformer struct {
	curves []Curve1D
	name   string
}
type InverseCurveTransformer struct {
	curves []Curve1D
	name   string
}

func (c CurveTransformer) IOSig() (int, int) {
	return len(c.curves), len(c.curves)
}

func (c CurveTransformer) Transform(r, g, b unit_float) (unit_float, unit_float, unit_float) {
	return c.curves[0].Transform(r), c.curves[1].Transform(g), c.curves[2].Transform(b)
}
func (c CurveTransformer) TransformGeneral(o, i []unit_float) {
	for n, c := range c.curves {
		o[n] = c.Transform(i[n])
	}
}

func (c InverseCurveTransformer) IOSig() (int, int) {
	return len(c.curves), len(c.curves)
}
func (c InverseCurveTransformer) Transform(r, g, b unit_float) (unit_float, unit_float, unit_float) {
	// we need to clamp as per spec section F.3 of ICC.1-2202-05.pdf
	return c.curves[0].InverseTransform(clamp01(r)), c.curves[1].InverseTransform(clamp01(g)), c.curves[2].InverseTransform(clamp01(b))
}
func (c InverseCurveTransformer) TransformGeneral(o, i []unit_float) {
	for n, c := range c.curves {
		o[n] = c.InverseTransform(i[n])
	}
}

type CurveTransformer3 struct {
	r, g, b Curve1D
	name    string
}

func (c CurveTransformer3) IOSig() (int, int) { return 3, 3 }
func (c CurveTransformer3) Transform(r, g, b unit_float) (unit_float, unit_float, unit_float) {
	return c.r.Transform(r), c.g.Transform(g), c.b.Transform(b)
}
func (m CurveTransformer3) TransformGeneral(o, i []unit_float) { tg33(m.Transform, o, i) }

type InverseCurveTransformer3 struct {
	r, g, b Curve1D
	name    string
}

func (c *CurveTransformer) Iter(f func(ChannelTransformer) bool)         { f(c) }
func (c *CurveTransformer3) Iter(f func(ChannelTransformer) bool)        { f(c) }
func (c *InverseCurveTransformer) Iter(f func(ChannelTransformer) bool)  { f(c) }
func (c *InverseCurveTransformer3) Iter(f func(ChannelTransformer) bool) { f(c) }
func (c *CurveTransformer) Curves() []Curve1D                            { return c.curves }
func (c *InverseCurveTransformer) Curves() []Curve1D                     { return c.curves }
func (c *CurveTransformer3) Curves() []Curve1D                           { return []Curve1D{c.r, c.g, c.b} }
func (c *InverseCurveTransformer3) Curves() []Curve1D                    { return []Curve1D{c.r, c.g, c.b} }

func curve_string(name string, is_inverse bool, curves ...Curve1D) string {
	var b strings.Builder
	if is_inverse {
		name += "Inverted"
	}
	b.WriteString(name + "{")
	for i, c := range curves {
		b.WriteString(fmt.Sprintf("[%d]%s ", i, c.String()))
	}
	b.WriteString("}")
	return b.String()
}

func (c CurveTransformer3) String() string        { return curve_string(c.name, false, c.r, c.g, c.b) }
func (c CurveTransformer) String() string         { return curve_string(c.name, false, c.curves...) }
func (c InverseCurveTransformer3) String() string { return curve_string(c.name, true, c.r, c.g, c.b) }
func (c InverseCurveTransformer) String() string  { return curve_string(c.name, true, c.curves...) }

func (c InverseCurveTransformer3) IOSig() (int, int) { return 3, 3 }
func (c InverseCurveTransformer3) Transform(r, g, b unit_float) (unit_float, unit_float, unit_float) {
	// we need to clamp as per spec section F.3 of ICC.1-2202-05.pdf
	return c.r.InverseTransform(clamp01(r)), c.g.InverseTransform(clamp01(g)), c.b.InverseTransform(clamp01(b))
}
func (m InverseCurveTransformer3) TransformGeneral(o, i []unit_float) { tg33(m.Transform, o, i) }

type Curves interface {
	ChannelTransformer
	Curves() []Curve1D
}

func NewCurveTransformer(name string, curves ...Curve1D) Curves {
	all_identity := true
	for _, c := range curves {
		if c == nil {
			ident := IdentityCurve(0)
			c = &ident
		}
		if _, is_ident := c.(*IdentityCurve); !is_ident {
			all_identity = false
		}
	}
	if all_identity {
		return nil
	}
	switch len(curves) {
	case 3:
		return &CurveTransformer3{curves[0], curves[1], curves[2], name}
	default:
		return &CurveTransformer{curves, name}
	}
}
func NewInverseCurveTransformer(name string, curves ...Curve1D) Curves {
	all_identity := true
	for _, c := range curves {
		if c == nil {
			ident := IdentityCurve(0)
			c = &ident
		}
		if _, is_ident := c.(*IdentityCurve); !is_ident {
			all_identity = false
		}
	}
	if all_identity {
		return nil
	}
	switch len(curves) {
	case 3:
		return &InverseCurveTransformer3{curves[0], curves[1], curves[2], name}
	default:
		return &InverseCurveTransformer{curves, name}
	}
}

type ParametricCurveFunction uint16

const (
	SimpleGammaFunction     ParametricCurveFunction = 0 // Y = X^g
	ConditionalZeroFunction ParametricCurveFunction = 1 // Y = (aX+b)^g for X >= d, else 0
	ConditionalCFunction    ParametricCurveFunction = 2 // Y = (aX+b)^g for X >= d, else c
	SplitFunction           ParametricCurveFunction = 3 // Two different functions split at d
	ComplexFunction         ParametricCurveFunction = 4 // More complex piecewise function
)

func align_to_4(x int) int {
	if extra := x % 4; extra > 0 {
		x += 4 - extra
	}
	return x
}

func fixed88ToFloat(raw []byte) unit_float {
	return unit_float(uint16(raw[0])<<8|uint16(raw[1])) / 256
}

func samples_to_analytic(points []unit_float) Curve1D {
	threshold := 1e-3
	switch {
	case len(points) < 2:
		return nil
	case len(points) > 400:
		threshold = FLOAT_EQUALITY_THRESHOLD
	case len(points) > 100:
		threshold = 2 * FLOAT_EQUALITY_THRESHOLD
	case len(points) > 40:
		threshold = 16 * FLOAT_EQUALITY_THRESHOLD
	}
	if len(points) < 2 {
		return nil
	}
	n := 1 / unit_float(len(points)-1)
	srgb := SRGBCurve().Transform
	is_srgb, is_identity := true, true
	for i, y := range points {
		x := unit_float(i) * n
		if is_srgb {
			is_srgb = math.Abs(float64(y-srgb(x))) <= threshold
		}
		if is_identity {
			is_identity = math.Abs(float64(y-x)) <= threshold
		}
		if !is_identity && !is_srgb {
			break
		}
	}
	if is_identity {
		ans := IdentityCurve(0)
		return &ans
	}
	if is_srgb {
		return SRGBCurve()
	}
	return nil
}

func load_points_curve(fp []unit_float) (Curve1D, error) {
	analytic := samples_to_analytic(fp)
	if analytic != nil {
		return analytic, nil
	}
	c := &PointsCurve{points: fp}
	if err := c.Prepare(); err != nil {
		return nil, err
	}
	return c, nil
}

func embeddedCurveDecoder(raw []byte) (any, int, error) {
	if len(raw) < 12 {
		return nil, 0, errors.New("curv tag too short")
	}
	count := int(binary.BigEndian.Uint32(raw[8:12]))
	consumed := align_to_4(12 + count*2)
	switch count {
	case 0:
		c := IdentityCurve(0)
		return &c, consumed, nil
	case 1:
		if len(raw) < 14 {
			return nil, 0, errors.New("curv tag missing gamma value")
		}
		g := &GammaCurve{gamma: fixed88ToFloat(raw[12:14])}
		if err := g.Prepare(); err != nil {
			return nil, 0, err
		}
		var c Curve1D = g
		if g.is_one {
			ic := IdentityCurve(0)
			c = &ic
		}
		return c, consumed, nil
	default:
		points := make([]uint16, count)
		_, err := binary.Decode(raw[12:], binary.BigEndian, points)
		if err != nil {
			return nil, 0, errors.New("curv tag truncated")
		}
		fp := make([]unit_float, len(points))
		for i, p := range points {
			fp[i] = unit_float(p) / math.MaxUint16
		}
		c, err := load_points_curve(fp)
		return c, consumed, err
	}
}

func curveDecoder(raw []byte) (any, error) {
	ans, _, err := embeddedCurveDecoder(raw)
	return ans, err
}

func readS15Fixed16BE(raw []byte) unit_float {
	msb := int16(raw[0])<<8 | int16(raw[1])
	lsb := uint16(raw[2])<<8 | uint16(raw[3])
	return unit_float(msb) + unit_float(lsb)/(1<<16)
}

func embeddedParametricCurveDecoder(raw []byte) (ans any, consumed int, err error) {
	block_len := len(raw)
	if block_len < 16 {
		return nil, 0, errors.New("para tag too short")
	}
	funcType := ParametricCurveFunction(binary.BigEndian.Uint16(raw[8:10]))
	const header_len = 12
	raw = raw[header_len:]
	p := func() unit_float {
		ans := readS15Fixed16BE(raw[:4])
		raw = raw[4:]
		return ans
	}
	defer func() { consumed = align_to_4(consumed) }()
	var c Curve1D

	switch funcType {
	case SimpleGammaFunction:
		if consumed = header_len + 4; block_len < consumed {
			return nil, 0, errors.New("para tag too short")
		}
		g := &GammaCurve{gamma: p()}
		if abs(g.gamma-1) < FLOAT_EQUALITY_THRESHOLD {
			ic := IdentityCurve(0)
			c = &ic
		} else {
			c = g
		}
	case ConditionalZeroFunction:
		if consumed = header_len + 3*4; block_len < consumed {
			return nil, 0, errors.New("para tag too short")
		}
		c = &ConditionalZeroCurve{g: p(), a: p(), b: p()}
	case ConditionalCFunction:
		if consumed = header_len + 4*4; block_len < consumed {
			return nil, 0, errors.New("para tag too short")
		}
		c = &ConditionalCCurve{g: p(), a: p(), b: p(), c: p()}
	case SplitFunction:
		if consumed = header_len + 5*4; block_len < consumed {
			return nil, 0, errors.New("para tag too short")
		}
		c = &SplitCurve{g: p(), a: p(), b: p(), c: p(), d: p()}
	case ComplexFunction:
		if consumed = header_len + 7*4; block_len < consumed {
			return nil, 0, errors.New("para tag too short")
		}
		c = &ComplexCurve{g: p(), a: p(), b: p(), c: p(), d: p(), e: p(), f: p()}
	default:
		return nil, 0, fmt.Errorf("unknown parametric function type: %d", funcType)
	}
	if err = c.Prepare(); err != nil {
		return nil, 0, err
	}
	return c, consumed, nil

}

func parametricCurveDecoder(raw []byte) (any, error) {
	ans, _, err := embeddedParametricCurveDecoder(raw)
	return ans, err
}

func (c IdentityCurve) Transform(x unit_float) unit_float {
	return x
}

func (c IdentityCurve) InverseTransform(x unit_float) unit_float {
	return x
}

func (c IdentityCurve) Prepare() error { return nil }
func (c IdentityCurve) String() string { return "IdentityCurve" }

func (c GammaCurve) Transform(x unit_float) unit_float {
	if x < 0 {
		if c.is_one {
			return x
		}
		return 0
	}
	return pow(x, c.gamma)
}

func (c GammaCurve) InverseTransform(x unit_float) unit_float {
	if x < 0 {
		if c.is_one {
			return x
		}
		return 0
	}
	return pow(x, c.inv_gamma)
}

func (c *GammaCurve) Prepare() error {
	if c.gamma == 0 {
		return fmt.Errorf("gamma curve has zero gamma value")
	}
	c.inv_gamma = 1 / c.gamma
	c.is_one = abs(c.gamma-1) < FLOAT_EQUALITY_THRESHOLD
	return nil
}
func (c GammaCurve) String() string { return fmt.Sprintf("GammaCurve{%f}", c.gamma) }

func calculate_reverse_for_well_behaved_sampled_curve(points []unit_float) []unit_float {
	n := len(points) - 1
	if n < 1 {
		return nil
	}
	var prev, maxy unit_float
	var miny unit_float = math.MaxFloat32
	for _, y := range points {
		if y < prev || y < 0 || y > 1 {
			return nil // not monotonic or range not in [0, 1]
		}
		prev = y
		miny = min(y, miny)
		maxy = max(y, maxy)
	}
	y_to_x := make([]unit_float, n+1)
	points_y_idx := 0
	n_inv := 1.0 / unit_float(n)
	for i := range y_to_x {
		if points_y_idx > n {
			// we are between maxy and 1
			y_to_x[i] = 1
			continue
		}
		if int(points[points_y_idx]*unit_float(n)) == i {
			y_to_x[i] = unit_float(points_y_idx) * n_inv
			for {
				points_y_idx++
				if points_y_idx > n || int(points[points_y_idx]*unit_float(n)) != i {
					break
				}
			}
		} else {
			if points_y_idx == 0 {
				// we are between 0 and miny
				y_to_x[i] = 0
				continue
			}
			// we are between points_y_idx-1 and points_y_idx
			y1, y2 := points[points_y_idx-1], points[points_y_idx]
			if y1 == 1 {
				y_to_x[i] = 1
				continue
			}
			for y1 == y2 {
				points_y_idx++
				y2 = 1
				if points_y_idx <= n {
					y2 = points[points_y_idx]
				}
			}
			y := unit_float(i) * n_inv
			frac := (y - y1) / (y2 - y1)
			x1 := unit_float(points_y_idx-1) * n_inv
			// x = x1 + frac * (x2 - x1)
			y_to_x[i] = x1 + frac*n_inv
		}
	}
	return y_to_x
}

func (c *PointsCurve) Prepare() error {
	c.max_idx = unit_float(len(c.points) - 1)
	reverse_lookup := calculate_reverse_for_well_behaved_sampled_curve(c.points)
	if reverse_lookup == nil {
		reverse_lookup = make([]unit_float, len(c.points))
		for i := range len(reverse_lookup) {
			y := unit_float(i) / unit_float(len(reverse_lookup)-1)
			idx := get_interval(c.points, y)
			if idx < 0 {
				reverse_lookup[i] = 0
			} else {
				y1, y2 := c.points[idx], c.points[idx+1]
				if y2 < y1 {
					y1, y2 = y2, y1
				}
				x1, x2 := unit_float(idx)/c.max_idx, unit_float(idx+1)/c.max_idx
				frac := (y - y1) / (y2 - y1)
				reverse_lookup[i] = x1 + frac*(x2-x1)
			}
		}
	}
	c.reverse_lookup = reverse_lookup
	c.reverse_max_idx = unit_float(len(reverse_lookup) - 1)
	return nil
}

func (c PointsCurve) Transform(v unit_float) unit_float {
	return sampled_value(c.points, c.max_idx, v)
}

func (c PointsCurve) InverseTransform(v unit_float) unit_float {
	return sampled_value(c.reverse_lookup, c.reverse_max_idx, v)
}
func (c PointsCurve) String() string { return fmt.Sprintf("PointsCurve{%d}", len(c.points)) }

func get_interval(lookup []unit_float, y unit_float) int {
	if len(lookup) < 2 {
		return -1
	}
	for i := range len(lookup) - 1 {
		y0, y1 := lookup[i], lookup[i+1]
		if y1 < y0 {
			y0, y1 = y1, y0
		}
		if y0 <= y && y <= y1 {
			return i
		}
	}
	return -1
}

func safe_inverse(x, fallback unit_float) unit_float {
	if x == 0 {
		return fallback
	}
	return 1 / x
}

func (c *ConditionalZeroCurve) Prepare() error {
	c.inv_a = safe_inverse(c.a, 1)
	c.threshold, c.inv_gamma = -c.b*c.inv_a, safe_inverse(c.g, 0)
	return nil
}

func (c *ConditionalZeroCurve) String() string {
	return fmt.Sprintf("ConditionalZeroCurve{a: %v b: %v g: %v}", c.a, c.b, c.g)
}

func (c *ConditionalZeroCurve) Transform(x unit_float) unit_float {
	// Y = (aX+b)^g if X ≥ -b/a else 0
	if x >= c.threshold {
		if e := c.a*x + c.b; e > 0 {
			return pow(e, c.g)
		}
	}
	return 0
}

func (c *ConditionalZeroCurve) InverseTransform(y unit_float) unit_float {
	// X = (Y^(1/g) - b) / a if Y >= 0 else X = -b/a
	// the below doesnt match the actual spec but matches lcms2 implementation
	return max(0, (pow(y, c.inv_gamma)-c.b)*c.inv_a)
}

func (c *ConditionalCCurve) Prepare() error {
	c.inv_a = safe_inverse(c.a, 1)
	c.threshold, c.inv_gamma = -c.b*c.inv_a, safe_inverse(c.g, 0)
	return nil
}

func (c *ConditionalCCurve) String() string {
	return fmt.Sprintf("ConditionalCCurve{a: %v b: %v c: %v g: %v}", c.a, c.b, c.c, c.g)
}

func (c *ConditionalCCurve) Transform(x unit_float) unit_float {
	// Y = (aX+b)^g + c if X ≥ -b/a else c
	if x >= c.threshold {
		if e := c.a*x + c.b; e > 0 {
			return pow(e, c.g) + c.c
		}
		return 0
	}
	return c.c
}

func (c *ConditionalCCurve) InverseTransform(y unit_float) unit_float {
	// X = ((Y-c)^(1/g) - b) / a if Y >= c else X = -b/a
	if e := y - c.c; e >= 0 {
		if e == 0 {
			return 0
		}
		return (pow(e, c.inv_gamma) - c.b) * c.inv_a
	}
	return c.threshold
}

func (c *SplitCurve) Prepare() error {
	c.threshold, c.inv_g, c.inv_a, c.inv_c = pow(c.a*c.d+c.b, c.g), safe_inverse(c.g, 0), safe_inverse(c.a, 1), safe_inverse(c.c, 1)
	return nil
}

func eq(a, b unit_float) bool { return abs(a-b) <= FLOAT_EQUALITY_THRESHOLD }

func (c *SplitCurve) IsSRGB() bool {
	s := SRGBCurve()
	return eq(s.a, c.a) && eq(s.b, c.b) && eq(s.c, c.c) && eq(s.d, c.d)
}

func (c *SplitCurve) String() string {
	if c.IsSRGB() {
		return "SRGBCurve"
	}
	return fmt.Sprintf("SplitCurve{a: %v b: %v c: %v d: %v g: %v}", c.a, c.b, c.c, c.d, c.g)
}

func (c *SplitCurve) Transform(x unit_float) unit_float {
	// Y = (aX+b)^g if X ≥ d else cX
	if x >= c.d {
		if e := c.a*x + c.b; e > 0 {
			return pow(e, c.g)
		}
		return 0
	}
	return c.c * x
}

func (c *SplitCurve) InverseTransform(y unit_float) unit_float {
	// X=((Y^1/g-b)/a)    | Y >= (ad+b)^g
	// X=Y/c              | Y< (ad+b)^g
	if y < c.threshold {
		return y * c.inv_c
	}
	return (pow(y, c.inv_g) - c.b) * c.inv_a
}

func (c *ComplexCurve) IsSRGB() bool {
	s := SRGBCurve()
	return eq(s.a, c.a) && eq(s.b, c.b) && eq(s.c, c.c) && eq(s.d, c.d) && eq(c.e, 0) && eq(c.f, 0)
}

type IsSRGB interface {
	IsSRGB() bool
}

func (c *ComplexCurve) Prepare() error {
	c.threshold, c.inv_g, c.inv_a, c.inv_c = pow(c.a*c.d+c.b, c.g)+c.e, safe_inverse(c.g, 0), safe_inverse(c.a, 1), safe_inverse(c.c, 1)
	return nil
}

func (c *ComplexCurve) String() string {
	return fmt.Sprintf("ComplexCurve{a: %v b: %v c: %v d: %v e: %v f: %v g: %v}", c.a, c.b, c.c, c.d, c.e, c.f, c.g)
}

func (c *ComplexCurve) Transform(x unit_float) unit_float {
	// Y = (aX+b)^g + e if X ≥ d else cX+f
	if x >= c.d {
		if e := c.a*x + c.b; e > 0 {
			return pow(e, c.g) + c.e
		}
		return c.e
	}
	return c.c*x + c.f
}

func (c *ComplexCurve) InverseTransform(y unit_float) unit_float {
	// X=((Y-e)1/g-b)/a   | Y >=(ad+b)^g+e), cd+f
	// X=(Y-f)/c          | else
	if y < c.threshold {
		return (y - c.f) * c.inv_c
	}
	if e := y - c.e; e > 0 {
		return (pow(e, c.inv_g) - c.b) * c.inv_a
	}
	return 0
}

var SRGBCurve = sync.OnceValue(func() *SplitCurve {
	ans := &SplitCurve{g: 2.4, a: 1 / 1.055, b: 0.055 / 1.055, c: 1 / 12.92, d: 0.0031308 * 12.92}
	ans.Prepare()
	return ans
})

var SRGBCurveTransformer = sync.OnceValue(func() Curves {
	return NewCurveTransformer("sRGB curve", SRGBCurve(), SRGBCurve(), SRGBCurve())
})
var SRGBCurveInverseTransformer = sync.OnceValue(func() Curves {
	return NewInverseCurveTransformer("TRC", SRGBCurve(), SRGBCurve(), SRGBCurve())
})
