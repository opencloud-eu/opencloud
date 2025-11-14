package colorconv

import (
	"fmt"
	"math"
)

// This package converts CIE L*a*b* colors defined relative to the D50 white point
// into sRGB values relative to D65. It performs chromatic
// adaptation (Bradford), fuses linear matrix transforms where possible for speed,
// and does a simple perceptually-minded gamut mapping by scaling chroma (a,b)
// down towards zero until the resulting sRGB is inside the [0,1] cube.
//
// Notes:
// - Input L,a,b are the usual CIELAB values (L in [0,100], a,b around -/+).
// - Returned sRGB values are in [0,1]. If gamut mapping fails (rare), values
//   will be clipped to [0,1] as a fallback.
//
// The code fuses the chromatic adaptation and XYZ->linear-sRGB matrices into a
// single 3x3 matrix so that the only linear operation after Lab->XYZ is a single
// matrix multiply.

type Vec3 [3]float64
type Mat3 [3][3]float64

var whiteD65 = Vec3{0.95047, 1.00000, 1.08883}

func (m *Mat3) String() string {
	return fmt.Sprintf("Matrix3{ %.6v %.6v %.6v }", m[0], m[1], m[2])
}

// Standard reference whites (CIE XYZ) normalized so Y = 1.0
// Note that WhiteD50 uses Z value from ICC spec rather that CIE spec.
var WhiteD50 = Vec3{0.96422, 1.00000, 0.82491}

type ConvertColor struct {
	whitepoint Vec3
	// Precomputed combined matrix from XYZ(whitepoint) directly to linear sRGB (D65).
	// Combined = srgbFromXYZ * adaptMatrix (where adaptMatrix adapts XYZ D50 -> XYZ D65).
	combined_XYZ_to_linear_SRGB Mat3
}

func (c ConvertColor) String() string {
	return fmt.Sprintf("{whitepoint:%.6v matrix:%.6v}", c.whitepoint, c.combined_XYZ_to_linear_SRGB)
}

func (cc *ConvertColor) AddPreviousMatrix(a, b, c [3]float64) {
	prev := Mat3{a, b, c}
	cc.combined_XYZ_to_linear_SRGB = mulMat3(cc.combined_XYZ_to_linear_SRGB, prev)
}

func NewConvertColor(whitepoint_x, whitepoint_y, whitepoint_z, scale float64) (ans *ConvertColor) {
	ans = &ConvertColor{whitepoint: Vec3{whitepoint_x, whitepoint_y, whitepoint_z}}
	adapt := chromaticAdaptationMatrix(ans.whitepoint, whiteD65)
	// sRGB (linear) transform matrix from CIE XYZ (D65)
	var srgbFromXYZ = Mat3{
		{3.2406 * scale, -1.5372 * scale, -0.4986 * scale},
		{-0.9689 * scale, 1.8758 * scale, 0.0415 * scale},
		{0.0557 * scale, -0.2040 * scale, 1.0570 * scale},
	}
	ans.combined_XYZ_to_linear_SRGB = mulMat3(srgbFromXYZ, adapt)
	return
}

func NewStandardConvertColor() (ans *ConvertColor) {
	return NewConvertColor(WhiteD50[0], WhiteD50[1], WhiteD50[2], 1)
}

// LabToSRGB converts a Lab color (at the specified whitepoint) into sRGB (D65) with gamut mapping.
// Returned components are in [0,1].
func (c *ConvertColor) LabToSRGB(L, a, b float64) (r, g, bl float64) {
	// fast path: try direct conversion and only do gamut mapping if out of gamut
	r0, g0, b0 := c.LabToSRGBNoGamutMap(L, a, b)
	if inGamut(r0, g0, b0) {
		return r0, g0, b0
	}
	// gamut map by scaling chroma (a,b) toward 0 while keeping L constant.
	rm, gm, bm := c.gamutMapChromaScale(L, a, b)
	return rm, gm, bm
}

// LabToSRGBNoGamutMap converts Lab(whitepoint) to sRGB(D65) without doing any gamut mapping.
// Values may be out of [0,1].
func (c *ConvertColor) LabToSRGBNoGamutMap(L, a, b float64) (r, g, bl float64) {
	rLin, gLin, bLin := c.LabToLinearRGB(L, a, b)
	r = linearToSRGBComp(rLin)
	g = linearToSRGBComp(gLin)
	bl = linearToSRGBComp(bLin)
	return
}

// LabToSRGBClamp converts Lab(whitepoint) to sRGB(D65) without doing any gamut mapping.
func (c *ConvertColor) LabToSRGBClamp(L, a, b float64) (r, g, bl float64) {
	rLin, gLin, bLin := c.LabToLinearRGB(L, a, b)
	r = clamp01(linearToSRGBComp(rLin))
	g = clamp01(linearToSRGBComp(gLin))
	bl = clamp01(linearToSRGBComp(bLin))
	return
}

// LabToLinearRGB converts Lab to linear RGB (not gamma-corrected), but still
// with chromatic adaptation to D65 fused into the matrix. Output is linear sRGB.
func (c *ConvertColor) LabToLinearRGB(L, a, b float64) (r, g, bl float64) {
	X, Y, Z := c.LabToXYZ(L, a, b)
	rv, gv, bv := mulMat3Vec(c.combined_XYZ_to_linear_SRGB, Vec3{X, Y, Z})
	return rv, gv, bv
}

// XYZToLinearRGB converts XYZ expressed relative to the specified whitepoint
// directly to linear sRGB values (D65) using the precomputed fused matrix.
// The output is linear RGB and may be outside the [0,1] range.
func (c *ConvertColor) XYZToLinearRGB(X, Y, Z float64) (r, g, b float64) {
	r, g, b = mulMat3Vec(c.combined_XYZ_to_linear_SRGB, Vec3{X, Y, Z})
	return
}

func (c *ConvertColor) Matrix() Mat3 {
	return c.combined_XYZ_to_linear_SRGB
}

// XYZToSRGBNoGamutMap converts XYZ expressed relative to the whitepoint directly to
// gamma-corrected sRGB values (D65). The outputs are clamped to [0,1].
// This function re-uses the precomputed combined matrix and the existing companding function.
func (c *ConvertColor) XYZToSRGBNoGamutMap(X, Y, Z float64) (r, g, b float64) {
	rl, gl, bl := c.XYZToLinearRGB(X, Y, Z)
	// Apply sRGB companding and clamp
	r = clamp01(linearToSRGBComp(rl))
	g = clamp01(linearToSRGBComp(gl))
	b = clamp01(linearToSRGBComp(bl))
	return
}

// If you need the non-clamped gamma-corrected values (for checking out-of-gamut)
// you can use this helper which only compands but doesn't clamp.
func (c *ConvertColor) XYZToSRGBNoClamp(X, Y, Z float64) (r, g, b float64) {
	rl, gl, bl := c.XYZToLinearRGB(X, Y, Z)
	r = linearToSRGBComp(rl)
	g = linearToSRGBComp(gl)
	b = linearToSRGBComp(bl)
	return
}

// XYZToSRGB converts XYZ (whitepoint) to sRGB (D65) using the Lab-projection
// + chroma-scaling gamut mapping. It projects XYZ into CIELAB (whitepoint), reuses the
// existing LabToSRGB (which performs chroma-scaling if needed), and returns final sRGB.
func (c *ConvertColor) XYZToSRGB(X, Y, Z float64) (r, g, b float64) {
	r, g, b = c.XYZToSRGBNoClamp(X, Y, Z)
	if inGamut(r, g, b) {
		return
	}
	L, a, bb := c.XYZToLab(X, Y, Z)
	return c.LabToSRGB(L, a, bb)
}

// Helpers: core conversions

func finv(t float64) float64 {
	const delta = 6.0 / 29.0
	if t > delta {
		return t * t * t
	}
	// when t <= delta: 3*delta^2*(t - 4/29)
	return 3 * delta * delta * (t - 4.0/29.0)
}

// LabToXYZ converts Lab (whitepoint) to CIE XYZ values relative to the whitepoint (Y=1).
func (c *ConvertColor) LabToXYZ(L, a, b float64) (X, Y, Z float64) {
	// Inverse of the CIELAB f function
	var fy = (L + 16.0) / 116.0
	var fx = fy + (a / 500.0)
	var fz = fy - (b / 200.0)

	xr := finv(fx)
	yr := finv(fy)
	zr := finv(fz)

	X = xr * c.whitepoint[0]
	Y = yr * c.whitepoint[1]
	Z = zr * c.whitepoint[2]
	return
}

func ff(t float64) float64 {
	const delta = 6.0 / 29.0
	if t > delta*delta*delta {
		return math.Cbrt(t)
	}
	// t <= delta^3
	return t/(3*delta*delta) + 4.0/29.0
}

func xyz_to_lab(wt Vec3, X, Y, Z float64) (L, a, b float64) {
	// Normalize by white
	xr := X / wt[0]
	yr := Y / wt[1]
	zr := Z / wt[2]

	fx := ff(xr)
	fy := ff(yr)
	fz := ff(zr)

	L = 116.0*fy - 16.0
	a = 500.0 * (fx - fy)
	b = 200.0 * (fy - fz)
	return

}

// XYZToLab converts XYZ (relative to whitepoint, Y=1) into CIELAB (whitepoint).
func (c *ConvertColor) XYZToLab(X, Y, Z float64) (L, a, b float64) {
	return xyz_to_lab(c.whitepoint, X, Y, Z)
}

// linearToSRGBComp applies the sRGB (gamma) companding function to a linear component.
func linearToSRGBComp(c float64) float64 {
	switch {
	case c <= 0.0031308:
		// clip small negative values for stability
		if c < 0 && c > -1./math.MaxUint16 {
			return 0
		}
		return 12.92 * c
	default:
		return 1.055*math.Pow(c, 1.0/2.4) - 0.055
	}
}

// Convert sRGB to linear light
func srgbToLinear(c float64) float64 {
	c = clamp01(c)
	// sRGB transfer function inverse
	if c <= 0.04045 {
		return c / 12.92
	}
	return math.Pow((c+0.055)/1.055, 2.4)
}

// Converts linear RGB to CIE XYZ using sRGB D65 matrix.
// Input r,g,b must be linear-light (not gamma-encoded).
func rgbToXYZ(r, g, b float64) (x, y, z float64) {
	// sRGB (linear) to XYZ (D65), matrix from IEC 61966-2-1
	x = 0.4124564*r + 0.3575761*g + 0.1804375*b
	y = 0.2126729*r + 0.7151522*g + 0.0721750*b
	z = 0.0193339*r + 0.1191920*g + 0.9503041*b
	return
}

func SrgbToLab(r, g, b float64) (L, a, B float64) {
	// convert gamma-encoded sRGB to linear
	r = srgbToLinear(r)
	g = srgbToLinear(g)
	b = srgbToLinear(b)
	x, y, z := rgbToXYZ(r, g, b)
	return xyz_to_lab(whiteD65, x, y, z)
}

// h' (in degrees 0..360)
func hp(aPrime, b float64) float64 {
	if aPrime == 0 && b == 0 {
		return 0.0
	}
	angle := math.Atan2(b, aPrime) * (180.0 / math.Pi)
	if angle < 0 {
		angle += 360.0
	}
	return angle
}

// DeltaE2000 computes the CIEDE2000 color-difference between two Lab colors.
// Implementation follows the formula from Sharma et al., 2005.
func DeltaE2000(L1, a1, b1, L2, a2, b2 float64) float64 {
	// Weighting factors
	kL, kC, kH := 1.0, 1.0, 1.0

	// Step 1: Compute C' and h'
	C1 := math.Hypot(a1, b1)
	C2 := math.Hypot(a2, b2)
	// mean C'
	Cbar := (C1 + C2) / 2.0

	// compute G
	Cbar7 := math.Pow(Cbar, 7)
	G := 0.5 * (1 - math.Sqrt(Cbar7/(Cbar7+math.Pow(25.0, 7))))

	// a' values
	ap1 := (1 + G) * a1
	ap2 := (1 + G) * a2

	// C' recalculated
	C1p := math.Hypot(ap1, b1)
	C2p := math.Hypot(ap2, b2)

	h1p := hp(ap1, b1)
	h2p := hp(ap2, b2)

	// delta L'
	dLp := L2 - L1
	// delta C'
	dCp := C2p - C1p

	// delta h'
	var dhp float64
	if C1p*C2p == 0 {
		dhp = 0
	} else {
		diff := h2p - h1p
		if math.Abs(diff) <= 180 {
			dhp = diff
		} else if diff > 180 {
			dhp = diff - 360
		} else {
			dhp = diff + 360
		}
	}
	// convert to radians for the formula
	dHp := 2 * math.Sqrt(C1p*C2p) * math.Sin((dhp*math.Pi/180.0)/2.0)

	// average L', C', h'
	LpBar := (L1 + L2) / 2.0
	CpBar := (C1p + C2p) / 2.0

	var hpBar float64
	if C1p*C2p == 0 {
		hpBar = h1p + h2p
	} else {
		diff := math.Abs(h1p - h2p)
		if diff <= 180 {
			hpBar = (h1p + h2p) / 2.0
		} else if (h1p + h2p) < 360 {
			hpBar = (h1p + h2p + 360) / 2.0
		} else {
			hpBar = (h1p + h2p - 360) / 2.0
		}
	}

	// T
	T := 1 - 0.17*math.Cos((hpBar-30)*math.Pi/180.0) +
		0.24*math.Cos((2*hpBar)*math.Pi/180.0) +
		0.32*math.Cos((3*hpBar+6)*math.Pi/180.0) -
		0.20*math.Cos((4*hpBar-63)*math.Pi/180.0)

	// delta theta
	dTheta := 30 * math.Exp(-((hpBar-275)/25)*((hpBar-275)/25))
	// R_C
	Rc := 2 * math.Sqrt(math.Pow(CpBar, 7)/(math.Pow(CpBar, 7)+math.Pow(25.0, 7)))
	// S_L, S_C, S_H
	Sl := 1 + ((0.015 * (LpBar - 50) * (LpBar - 50)) / math.Sqrt(20+((LpBar-50)*(LpBar-50))))
	Sc := 1 + 0.045*CpBar
	Sh := 1 + 0.015*CpBar*T
	// R_T
	RT := -math.Sin(2*dTheta*math.Pi/180.0) * Rc

	// finally
	dL := dLp / (kL * Sl)
	dC := dCp / (kC * Sc)
	dH := dHp / (kH * Sh)

	return math.Sqrt(dL*dL + dC*dC + dH*dH + RT*dC*dH)
}

// DeltaEBetweenSrgb takes two sRGB colors (0..1) and returns the Delta E (CIEDE2000).
func DeltaEBetweenSrgb(r1, g1, b1, r2, g2, b2 float64) float64 {
	L1, a1, b1 := SrgbToLab(r1, g1, b1)
	L2, a2, b2 := SrgbToLab(r2, g2, b2)
	return DeltaE2000(L1, a1, b1, L2, a2, b2)
}

// inGamut checks whether r,g,b are all inside [0,1]
func inGamut(r, g, b float64) bool {
	return 0 <= r && r <= 1 && 0 <= g && g <= 1 && 0 <= b && b <= 1
}

// gamutMapChromaScale reduces chroma (a,b) by scaling factor s in [0,1] to bring the
// color into gamut. Binary search is used to find the maximum s such that the color
// is in gamut. L is preserved.
func (c *ConvertColor) gamutMapChromaScale(L, a, b float64) (r, g, bl float64) {
	// If a==0 && b==0 we can't scale; just clip after conversion
	if a == 0 && b == 0 {
		r0, g0, b0 := c.LabToSRGBNoGamutMap(L, a, b)
		return clamp01(r0), clamp01(g0), clamp01(b0)
	}
	// Binary search scale factor in [0,1]
	lo := 0.0
	hi := 1.0
	var mid float64
	var foundR, foundG, foundB float64
	// If even fully desaturated (a=b=0) is out of gamut, we'll clip
	for range 24 {
		mid = (lo + hi) / 2.0
		a2 := a * mid
		b2 := b * mid
		r0, g0, b0 := c.LabToSRGBNoGamutMap(L, a2, b2)
		if inGamut(r0, g0, b0) {
			foundR, foundG, foundB = r0, g0, b0
			// can try to keep more chroma
			lo = mid
		} else {
			hi = mid
		}
	}
	// If we never found a valid in-gamut during binary search, try a= b =0
	if !(inGamut(foundR, foundG, foundB)) {
		r0, g0, b0 := c.LabToSRGBNoGamutMap(L, 0, 0)
		// if still out-of-gamut (very unlikely), clip
		return clamp01(r0), clamp01(g0), clamp01(b0)
	}
	return clamp01(foundR), clamp01(foundG), clamp01(foundB)
}

// clamp01 clamps value to [0,1]
func clamp01(x float64) float64 {
	return max(0, min(x, 1))
}

// Matrix & vector utilities

func mulMat3(a, b Mat3) Mat3 {
	var out Mat3
	for i := range 3 {
		for j := range 3 {
			sum := 0.0
			for k := range 3 {
				sum += a[i][k] * b[k][j]
			}
			out[i][j] = sum
		}
	}
	return out
}

func mulMat3Vec(m Mat3, v Vec3) (x, y, z float64) {
	x = m[0][0]*v[0] + m[0][1]*v[1] + m[0][2]*v[2]
	y = m[1][0]*v[0] + m[1][1]*v[1] + m[1][2]*v[2]
	z = m[2][0]*v[0] + m[2][1]*v[1] + m[2][2]*v[2]
	return
}

// chromaticAdaptationMatrix constructs a 3x3 matrix that adapts XYZ values
// from sourceWhite to targetWhite using the Bradford method.
func chromaticAdaptationMatrix(sourceWhite, targetWhite Vec3) Mat3 {
	// Bradford transform matrices (forward and inverse)
	var (
		bradford = Mat3{
			{0.8951, 0.2664, -0.1614},
			{-0.7502, 1.7135, 0.0367},
			{0.0389, -0.0685, 1.0296},
		}
		bradford_inverted = Mat3{
			{0.9869929054667121, -0.1470542564209901, 0.1599626516637312},
			{0.4323052697233945, 0.5183602715367774, 0.049291228212855594},
			{-0.008528664575177326, 0.04004282165408486, 0.96848669578755},
		}
	)

	// Convert whites to LMS using Bradford
	srcL, srcM, srcS := mulMat3Vec(bradford, sourceWhite)
	tgtL, tgtM, tgtS := mulMat3Vec(bradford, targetWhite)
	// Build diag matrix in-between
	diag := Mat3{
		{tgtL / srcL, 0, 0},
		{0, tgtM / srcM, 0},
		{0, 0, tgtS / srcS},
	}
	// adapt = invBradford * diag * bradford
	tmp := mulMat3(diag, bradford)           // diag*B
	adapt := mulMat3(bradford_inverted, tmp) // invB * (diag*B)
	return adapt
}
