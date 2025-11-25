package meta

import (
	"fmt"
	"math"

	"github.com/kovidgoyal/imaging/prism/meta/icc"
)

var _ = fmt.Print

type CodingIndependentCodePoints struct {
	ColorPrimaries, TransferCharacteristics, MatrixCoefficients, VideoFullRange uint8
	IsSet                                                                       bool
}

var SRGB = CodingIndependentCodePoints{1, 13, 0, 1, true}
var DISPLAY_P3 = CodingIndependentCodePoints{12, 13, 0, 1, true}

func (c CodingIndependentCodePoints) String() string {
	return fmt.Sprintf("CodingIndependentCodePoints{ColorPrimaries: %d, TransferCharacteristics: %d, MatrixCoefficients: %d, VideoFullRange: %d}", c.ColorPrimaries, c.TransferCharacteristics, c.MatrixCoefficients, c.VideoFullRange)
}

func (c CodingIndependentCodePoints) IsSRGB() bool {
	return c == SRGB
}

func (c CodingIndependentCodePoints) VideoFullRangeIsValid() bool {
	return c.VideoFullRange == 0 || c.VideoFullRange == 1
}

// See https://www.w3.org/TR/png-3/#cICP-chunk for why we do this
func extend_over_full_range(f func(float64) float64) func(float64) float64 {
	return func(x float64) float64 {
		return math.Copysign(1, x) * f(math.Abs(x))
	}
}

func (src CodingIndependentCodePoints) PipelineTo(dest CodingIndependentCodePoints) *icc.Pipeline {
	if src == dest {
		return nil
	}
	if src.MatrixCoefficients != 0 || dest.MatrixCoefficients != 0 {
		return nil // TODO: Add support for these
	}
	if !src.VideoFullRangeIsValid() || !dest.VideoFullRangeIsValid() {
		return nil
	}
	p := primaries[int(src.ColorPrimaries)]
	if p.Name == "" {
		return nil
	}
	tc := transfer_functions[int(src.TransferCharacteristics)]
	if tc.Name == "" {
		return nil
	}
	to_linear := icc.NewUniformFunctionTransformer(tc.Name, icc.IfElse(src.VideoFullRange == SRGB.VideoFullRange, tc.EOTF, extend_over_full_range(tc.EOTF)))
	if tc.Name == "Identity" {
		to_linear = nil
	}
	linear_to_xyz := p.CalculateRGBtoXYZMatrix()
	p = primaries[int(dest.ColorPrimaries)]
	if p.Name == "" {
		return nil
	}
	tc = transfer_functions[int(dest.TransferCharacteristics)]
	if tc.Name == "" {
		return nil
	}
	xyz_to_linear := p.CalculateRGBtoXYZMatrix()
	xyz_to_linear, err := xyz_to_linear.Inverted()
	if err != nil {
		panic(err)
	}
	f := icc.IfElse(dest.VideoFullRange == SRGB.VideoFullRange, tc.OETF, extend_over_full_range(tc.OETF))
	from_linear := icc.NewUniformFunctionTransformer(tc.Name, func(x float64) float64 {
		// TODO: Gamut mapping for white point of dest, re-use code from colorconv
		return max(0, min(f(x), 1))
	})
	if tc.Name == "Identity" {
		from_linear = nil
	}
	ans := &icc.Pipeline{}
	ans.Append(to_linear, &linear_to_xyz, &xyz_to_linear, from_linear)
	ans.Finalize(true)
	return ans
}

func (c CodingIndependentCodePoints) PipelineToSRGB() *icc.Pipeline {
	return c.PipelineTo(SRGB)
}

// XY holds CIE xy chromaticity coordinates.
type XY struct {
	X, Y float64
}

// ColorSpace defines the primaries and white point of a color space.
type Primaries struct {
	Name  string
	Red   XY
	Green XY
	Blue  XY
	White XY
}

// xyToXYZ converts xy chromaticity to XYZ coordinates, assuming Y=1.
func xyToXYZ(p XY) [3]float64 {
	if p.Y == 0 {
		return [3]float64{0, 0, 0}
	}
	return [3]float64{
		p.X / p.Y,
		1.0,
		(1.0 - p.X - p.Y) / p.Y,
	}
}

// CalculateRGBtoXYZMatrix computes the matrix to convert from a linear RGB color space to CIE XYZ.
func (cs *Primaries) CalculateRGBtoXYZMatrix() icc.Matrix3 {
	// Convert primaries to XYZ space (normalized to Y=1)
	r := xyToXYZ(cs.Red)
	g := xyToXYZ(cs.Green)
	b := xyToXYZ(cs.Blue)

	// Form the matrix of primaries
	M := icc.Matrix3{
		{r[0], g[0], b[0]},
		{r[1], g[1], b[1]},
		{r[2], g[2], b[2]},
	}

	// Calculate the scaling factors (S_r, S_g, S_b)
	invM, err := M.Inverted()
	if err != nil {
		panic(err)
	}
	whiteXYZ := xyToXYZ(cs.White)

	s_r := invM[0][0]*whiteXYZ[0] + invM[0][1]*whiteXYZ[1] + invM[0][2]*whiteXYZ[2]
	s_g := invM[1][0]*whiteXYZ[0] + invM[1][1]*whiteXYZ[1] + invM[1][2]*whiteXYZ[2]
	s_b := invM[2][0]*whiteXYZ[0] + invM[2][1]*whiteXYZ[1] + invM[2][2]*whiteXYZ[2]

	// Scale the primaries matrix to get the final conversion matrix
	finalMatrix := icc.Matrix3{
		{M[0][0] * s_r, M[0][1] * s_g, M[0][2] * s_b},
		{M[1][0] * s_r, M[1][1] * s_g, M[1][2] * s_b},
		{M[2][0] * s_r, M[2][1] * s_g, M[2][2] * s_b},
	}
	return finalMatrix
}

type WellKnownPrimaries int

// These come from ITU-T H.273 spec
var primaries = map[int]Primaries{
	1: {
		Name:  "sRGB",
		Green: XY{X: 0.30, Y: 0.60},
		Blue:  XY{X: 0.15, Y: 0.06},
		Red:   XY{X: 0.64, Y: 0.33},
		White: XY{X: 0.3127, Y: 0.3290}, // D65
	},
	4: {
		Name:  "BT-470M",
		Green: XY{X: 0.21, Y: 0.71},
		Blue:  XY{X: 0.14, Y: 0.08},
		Red:   XY{X: 0.67, Y: 0.33},
		White: XY{X: 0.310, Y: 0.316},
	},
	5: {
		Name:  "BT-470B",
		Green: XY{X: 0.29, Y: 0.69},
		Blue:  XY{X: 0.15, Y: 0.06},
		Red:   XY{X: 0.64, Y: 0.33},
		White: XY{X: 0.310, Y: 0.316},
	},
	6: {
		Name:  "BT-601",
		Green: XY{0.310, 0.595},
		Blue:  XY{0.155, 0.070},
		Red:   XY{0.630, 0.340},
		White: XY{0.3127, 0.3290},
	},
	7: {
		Name:  "BT-601",
		Green: XY{0.310, 0.595},
		Blue:  XY{0.155, 0.070},
		Red:   XY{0.630, 0.340},
		White: XY{0.3127, 0.3290},
	},
	8: {
		Name:  "Generic film",
		Green: XY{0.243, 0.692},
		Blue:  XY{0.145, 0.049},
		Red:   XY{0.681, 0.319},
		White: XY{0.310, 0.316},
	},
	9: {
		Name:  "BT-2020",
		Green: XY{0.170, 0.797},
		Blue:  XY{0.131, 0.046},
		Red:   XY{0.708, 0.292},
		White: XY{0.3127, 0.3290},
	},
	10: { // 10
		Name:  "SMPTE ST 428-1",
		Green: XY{0.0, 1.0},
		Blue:  XY{0.0, 0.0},
		Red:   XY{1.0, 0.0},
		White: XY{1 / 3., 1 / 3.},
	},
	11: { // 11
		Name:  "DCI-P3",
		Green: XY{0.265, 0.690},
		Blue:  XY{0.150, 0.060},
		Red:   XY{0.680, 0.320},
		White: XY{0.314, 0.351}, // DCI White
	},
	12: { // 12
		Name:  "Diplay P3",
		Green: XY{0.265, 0.690},
		Blue:  XY{0.150, 0.060},
		Red:   XY{0.680, 0.320},
		White: XY{0.3127, 0.3290}, // D65
	},
	22: { // 22
		Name:  "Unnamed",
		Green: XY{0.295, 0.605},
		Blue:  XY{0.155, 0.077},
		Red:   XY{0.630, 0.340},
		White: XY{0.3127, 0.3290}, // D65
	},
}

// TransferFunction defines an Opto-Electronic Transfer Function (OETF)
// and its inverse Electro-Optical Transfer Function (EOTF).
type TransferFunction struct {
	ID   int
	Name string
	OETF func(float64) float64 // To non-linear
	EOTF func(float64) float64 // To linear
}

// Constants from various specifications used in the transfer functions.
const (
	// BT.709, BT.2020, BT.601
	alpha709 = 1.099
	beta709  = 0.018
	gamma709 = 0.45
	delta709 = 4.5

	// SMPTE ST 240M
	alpha240M = 1.1115
	beta240M  = 0.0228
	gamma240M = 0.45
	delta240M = 4.0

	// SMPTE ST 428-1
	gamma428 = 1.0 / 2.6

	// PQ (Perceptual Quantizer) - SMPTE ST 2084
	m1PQ = 2610.0 / 16384.0 // (2610 / 4096) * (1/4)
	m2PQ = 2523.0 / 32.0    // (2523 / 4096) * 128
	c1PQ = 3424.0 / 4096.0
	c2PQ = 2413.0 / 4096.0 * 32.0
	c3PQ = 2392.0 / 4096.0 * 32.0

	// HLG (Hybrid Log-Gamma) - ARIB STD-B67
	aHLG = 0.17883277
	bHLG = 1.0 - 4.0*aHLG // 0.28466892
	cHLG = 0.55991073     // 0.5 - aHLG*math.Log(4.0*aHLG)
)

// holds all the H.273 transfer characteristics.
var transfer_functions = make(map[int]TransferFunction)

func init() {
	tf1 := TransferFunction{
		ID: 1, Name: "BT.709",
		OETF: func(L float64) float64 {
			if L < beta709 {
				return delta709 * L
			}
			return alpha709*math.Pow(L, gamma709) - (alpha709 - 1)
		},
		EOTF: func(V float64) float64 {
			if V < delta709*beta709 {
				return V / delta709
			}
			return math.Pow((V+(alpha709-1))/alpha709, 1.0/gamma709)
		},
	}
	transfer_functions[1] = tf1
	transfer_functions[6] = tf1  // BT.601, BT.2020 share this with BT.709
	transfer_functions[14] = tf1 // BT.2020 10-bit
	transfer_functions[15] = tf1 // BT.2020 12-bit

	// 2: Identity
	transfer_functions[2] = TransferFunction{
		ID: 2, Name: "Identity",
		OETF: func(v float64) float64 { return v },
		EOTF: func(v float64) float64 { return v },
	}
	transfer_functions[8] = transfer_functions[2]

	// 4: Gamma 2.2
	tf4 := TransferFunction{
		ID: 4, Name: "Gamma 2.2",
		OETF: func(L float64) float64 { return math.Pow(L, 1.0/2.2) },
		EOTF: func(V float64) float64 { return math.Pow(V, 2.2) },
	}
	transfer_functions[4] = tf4
	transfer_functions[5] = tf4 // BT.470BG also uses Gamma 2.2 approx.

	// 5: Gamma 2.8
	transfer_functions[5] = TransferFunction{
		ID: 5, Name: "Gamma 2.8",
		OETF: func(L float64) float64 { return math.Pow(L, 1.0/2.8) },
		EOTF: func(V float64) float64 { return math.Pow(V, 2.8) },
	}

	// 7: SMPTE 240M
	tf7 := TransferFunction{
		ID: 7, Name: "SMPTE 240M",
		OETF: func(L float64) float64 {
			if L < beta240M {
				return delta240M * L
			}
			return alpha240M*math.Pow(L, gamma240M) - (alpha240M - 1)
		},
		EOTF: func(V float64) float64 {
			if V < delta240M*beta240M {
				return V / delta240M
			}
			return math.Pow((V+(alpha240M-1))/alpha240M, 1.0/gamma240M)
		},
	}
	transfer_functions[7] = tf7
	// 9: Logarithmic (100:1)
	transfer_functions[9] = TransferFunction{
		ID: 9, Name: "Logarithmic (100:1)",
		OETF: func(L float64) float64 {
			return 1.0 - math.Log10(1.0-L*(1.0-math.Pow(10.0, -2.0)))/2.0
		},
		EOTF: func(V float64) float64 {
			return (1.0 - math.Pow(10.0, -2.0*V)) / (1.0 - math.Pow(10.0, -2.0))
		},
	}

	// 10: Logarithmic (100 * sqrt(10):1)
	transfer_functions[10] = TransferFunction{
		ID: 10, Name: "Logarithmic (100*sqrt(10):1)",
		OETF: func(L float64) float64 {
			return 1.0 - math.Log10(1.0-L*(1.0-math.Pow(10.0, -2.5)))/2.5
		},
		EOTF: func(V float64) float64 {
			return (1.0 - math.Pow(10.0, -2.5*V)) / (1.0 - math.Pow(10.0, -2.5))
		},
	}

	// 11: IEC 61966-2-4
	transfer_functions[11] = TransferFunction{
		ID: 11, Name: "IEC 61966-2-4",
		OETF: func(L float64) float64 {
			if L < -beta709 {
				return -delta709 * -L
			}
			if L > beta709 {
				return alpha709*math.Pow(L, gamma709) - (alpha709 - 1)
			}
			return delta709 * L
		},
		EOTF: func(V float64) float64 {
			if V < -delta709*beta709 {
				return -math.Pow((-V+(alpha709-1))/alpha709, 1.0/gamma709)
			}
			if V > delta709*beta709 {
				return math.Pow((V+(alpha709-1))/alpha709, 1.0/gamma709)
			}
			return V / delta709
		},
	}

	// 12: BT.1361 extended gamut
	tf12 := tf1 // It's based on BT.709
	tf12.ID = 12
	tf12.Name = "BT.1361"
	transfer_functions[12] = tf12

	// 13: sRGB/IEC 61966-2-1
	transfer_functions[13] = TransferFunction{
		ID: 13, Name: "sRGB",
		OETF: func(L float64) float64 {
			if L <= 0.0031308 {
				return 12.92 * L
			}
			return 1.055*math.Pow(L, 1.0/2.4) - 0.055
		},
		EOTF: func(V float64) float64 {
			if V <= 0.04045 {
				return V / 12.92
			}
			return math.Pow((V+0.055)/1.055, 2.4)
		},
	}
	// 16: SMPTE ST 2084 (PQ)
	transfer_functions[16] = TransferFunction{
		ID: 16, Name: "SMPTE ST 2084 (PQ)",
		OETF: func(L float64) float64 { // EOTF^-1, L is normalized to 10000 cd/m^2
			Lp := math.Pow(L, m1PQ)
			return math.Pow((c1PQ+c2PQ*Lp)/(1.0+c3PQ*Lp), m2PQ)
		},
		EOTF: func(V float64) float64 { // V is non-linear signal
			Vp := math.Pow(V, 1.0/m2PQ)
			num := math.Max(Vp-c1PQ, 0.0)
			den := math.Max(c2PQ-c3PQ*Vp, 1e-6) // Avoid division by zero
			return math.Pow(num/den, 1.0/m1PQ)
		},
	}

	// 17: SMPTE ST 428-1
	transfer_functions[17] = TransferFunction{
		ID: 17, Name: "SMPTE ST 428-1",
		OETF: func(L float64) float64 { // OOTF^-1, from linear scene light to D-cinema
			// Input L is assumed to be scene linear (48 cd/m^2 peak)
			// The spec normalizes by 52.37
			return math.Pow((L*48.0)/52.37, gamma428)
		},
		EOTF: func(V float64) float64 { // OOTF
			// Output is linear light, normalized to 1.0 for peak white (48 cd/m^2)
			return (52.37 / 48.0) * math.Pow(V, 1.0/gamma428)
		},
	}

	// 18: ARIB STD-B67 (HLG)
	transfer_functions[18] = TransferFunction{
		ID: 18, Name: "ARIB STD-B67 (HLG)",
		OETF: func(L float64) float64 { // L is scene linear light, display-referred
			if L <= 1.0/12.0 {
				return math.Sqrt(3.0 * L)
			}
			return aHLG*math.Log(12.0*L-bHLG) + cHLG
		},
		EOTF: func(V float64) float64 { // V is the non-linear signal
			if V <= 0.5 {
				return (V * V) / 3.0
			}
			return (math.Exp((V-cHLG)/aHLG) + bHLG) / 12.0
		},
	}
}
