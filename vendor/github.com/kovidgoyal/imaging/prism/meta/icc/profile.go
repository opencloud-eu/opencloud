package icc

import (
	"bytes"
	_ "embed"
	"fmt"
	"io"
	"math"
	"os"
	"sync"
)

var _ = fmt.Println

type WellKnownProfile int

//go:embed test-profiles/sRGB-v4.icc
var Srgb_xyz_profile_data []byte

const (
	UnknownProfile WellKnownProfile = iota
	SRGBProfile
	AdobeRGBProfile
	PhotoProProfile
	DisplayP3Profile
)

type Profile struct {
	Header        Header
	TagTable      TagTable
	PCSIlluminant XYZType
	blackpoints   map[RenderingIntent]*XYZType

	blackpoints_lock sync.Mutex
}

func (p *Profile) Description() (string, error) {
	return p.TagTable.getProfileDescription()
}

func (p *Profile) DeviceManufacturerDescription() (string, error) {
	return p.TagTable.getDeviceManufacturerDescription()
}

func (p *Profile) DeviceModelDescription() (string, error) {
	return p.TagTable.getDeviceModelDescription()
}

// See _cmsReadMediaWhitePoint() in cmsio1.c
func (p *Profile) media_white_point() XYZType {
	// V2 display profiles should give D50
	if p.Header.Version.Major < 4 && p.Header.DeviceClass == DeviceClassDisplay {
		return lcms_d50
	}
	x, err := p.TagTable.get_parsed(MediaWhitePointTagSignature, p.Header.DataColorSpace, p.Header.ProfileConnectionSpace)
	if err != nil {
		return lcms_d50
	}
	if wtpt, ok := x.(*XYZType); ok {
		return *wtpt
	}
	return lcms_d50
}

// The transformers to convert relative colorimetric normalized PCS values to
// absolute colorimetric ones (or the reverse when forward is false). See
// ComputeAbsoluteIntent() in cmscnvrt.c, we use an adaptation state of 1 (the
// lcms default) and D50 as the white point of the other side of the
// conversion, which is the PCS or an sRGB profile.
func (p *Profile) absolute_colorimetric_adaptation(forward bool, intent RenderingIntent) []ChannelTransformer {
	if intent != AbsoluteColorimetricRenderingIntent {
		return nil
	}
	wp := p.media_white_point()
	// v4 profiles have a media white point of D50 up to the precision of s15Fixed16Number
	const threshold = 2. / 65536
	if math.Abs(wp.X-lcms_d50.X) < threshold && math.Abs(wp.Y-lcms_d50.Y) < threshold && math.Abs(wp.Z-lcms_d50.Z) < threshold {
		return nil
	}
	m := &Matrix3{{wp.X / lcms_d50.X, 0, 0}, {0, wp.Y / lcms_d50.Y, 0}, {0, 0, wp.Z / lcms_d50.Z}}
	if !forward {
		m = &Matrix3{{lcms_d50.X / wp.X, 0, 0}, {0, lcms_d50.Y / wp.Y, 0}, {0, 0, lcms_d50.Z / wp.Z}}
	}
	if p.Header.ProfileConnectionSpace == ColorSpaceLab {
		// the scaling must be done in XYZ space
		return []ChannelTransformer{NewNormalizedToLAB(), NewLABtoXYZ(p.PCSIlluminant), m, NewXYZtoLAB(p.PCSIlluminant), NewLABToNormalized()}
	}
	return []ChannelTransformer{m}
}

func (p *Profile) create_matrix_trc_transformer(forward bool, pipeline *Pipeline) (err error) {
	if p.Header.DataColorSpace == ColorSpaceGray {
		return p.create_gray_trc_transformer(forward, pipeline)
	}
	if p.Header.ProfileConnectionSpace != ColorSpaceXYZ {
		return fmt.Errorf("matrix/TRC based profile using non XYZ PCS color space: %v", p.Header.ProfileConnectionSpace)
	}
	// See section F.3 of ICC.1-2202-5.pdf for how these transforms are composed
	var rc, gc, bc Curve1D
	if rc, err = p.TagTable.load_curve_tag(RedTRCTagSignature); err != nil {
		return err
	}
	if gc, err = p.TagTable.load_curve_tag(GreenTRCTagSignature); err != nil {
		return err
	}
	if bc, err = p.TagTable.load_curve_tag(BlueTRCTagSignature); err != nil {
		return err
	}
	m, err := p.TagTable.load_rgb_matrix(forward)
	if err != nil {
		return err
	}
	if forward {
		pipeline.Append(NewCurveTransformer("TRC", rc, gc, bc), m)
	} else {
		pipeline.Append(m, NewInverseCurveTransformer("TRC", rc, gc, bc))
	}
	return nil
}

// See section 8.10.2 of ICC.1-2202-05.pdf for tag selection algorithm
func (p *Profile) find_conversion_tag(forward bool, rendering_intent RenderingIntent) (ans ChannelTransformer, err error) {
	var ans_sig Signature = UnknownSignature
	found_tag := false
	if forward {
		switch rendering_intent {
		case PerceptualRenderingIntent:
			ans_sig = AToB0TagSignature
		case RelativeColorimetricRenderingIntent:
			ans_sig = AToB1TagSignature
		case SaturationRenderingIntent:
			ans_sig = AToB2TagSignature
		case AbsoluteColorimetricRenderingIntent:
			// absolute colorimetric uses the relative colorimetric table
			// with adaptation to the media white point, see Device2PCS16 in cmsio1.c
			ans_sig = AToB1TagSignature
		default:
			return nil, fmt.Errorf("unknown rendering intent: %v", rendering_intent)
		}
		found_tag = p.TagTable.Has(ans_sig)
		const fallback = AToB0TagSignature
		if !found_tag && p.TagTable.Has(fallback) {
			ans_sig = fallback
			found_tag = true
		}
	} else {
		switch rendering_intent {
		case PerceptualRenderingIntent:
			ans_sig = BToA0TagSignature
		case RelativeColorimetricRenderingIntent:
			ans_sig = BToA1TagSignature
		case SaturationRenderingIntent:
			ans_sig = BToA2TagSignature
		case AbsoluteColorimetricRenderingIntent:
			ans_sig = BToA1TagSignature
		default:
			return nil, fmt.Errorf("unknown rendering intent: %v", rendering_intent)
		}
		found_tag = p.TagTable.Has(ans_sig)
		const fallback = BToA0TagSignature
		if !found_tag && p.TagTable.Has(fallback) {
			ans_sig = fallback
			found_tag = true
		}
	}
	if !found_tag {
		return nil, nil
	}
	// We rely on profile reader to error out if the PCS color space is not XYZ
	// or LAB and the device colorspace is not RGB, CMYK or Gray
	input_colorspace, output_colorspace := p.Header.DataColorSpace, p.Header.ProfileConnectionSpace
	if !forward {
		input_colorspace, output_colorspace = output_colorspace, input_colorspace
	}
	c, err := p.TagTable.get_parsed(ans_sig, input_colorspace, output_colorspace)
	if err != nil {
		return nil, err
	}
	ans, ok := c.(ChannelTransformer)
	if !ok {
		return nil, fmt.Errorf("%s tag is not a ChannelTransformer: %T", ans_sig, c)
	}
	return ans, nil
}

func (p *Profile) effective_bpc(intent RenderingIntent, user_requested_bpc bool) bool {
	// See _cmsLinkProfiles() in cmscnvrt.c
	if intent == AbsoluteColorimetricRenderingIntent {
		return false
	}
	if (intent == PerceptualRenderingIntent || intent == SaturationRenderingIntent) && p.Header.Version.Major >= 4 {
		return true
	}
	return user_requested_bpc
}

func (p *Profile) CreateTransformerToDevice(rendering_intent RenderingIntent, use_blackpoint_compensation, optimize bool) (ans *Pipeline, err error) {
	num_output_channels := len(p.Header.DataColorSpace.BlackPoint())
	if num_output_channels == 0 {
		return nil, fmt.Errorf("unsupported device color space: %s", p.Header.DataColorSpace)
	}
	defer func() {
		if err == nil && !ans.IsSuitableFor(3, num_output_channels) {
			err = fmt.Errorf("transformer to PCS %s not suitable for 3 output channels", ans.String())
		}
		if err == nil {
			ans.finalize(optimize)
		}
	}()
	ans = &Pipeline{}

	pcs_normalized := false
	if p.effective_bpc(rendering_intent, use_blackpoint_compensation) {
		var PCS_blackpoint XYZType // 0, 0, 0
		output_blackpoint := p.BlackPoint(rendering_intent, nil)
		if PCS_blackpoint != output_blackpoint {
			is_lab := p.Header.ProfileConnectionSpace == ColorSpaceLab
			if is_lab {
				ans.Append(NewLABtoXYZ(p.PCSIlluminant))
			}
			// black point correction works on normalized XYZ
			ans.Append(NewXYZToNormalized())
			ans.Append(NewBlackPointCorrection(p.PCSIlluminant, PCS_blackpoint, output_blackpoint))
			if is_lab {
				ans.Append(NewNormalizedToXYZ())
				ans.Append(NewXYZtoLAB(p.PCSIlluminant))
			} else {
				pcs_normalized = true
			}
		}
	}
	if !pcs_normalized {
		ans.Append(transform_for_pcs_colorspace(p.Header.ProfileConnectionSpace, false))
	}

	const forward = false
	b2a, err := p.find_conversion_tag(forward, rendering_intent)
	if err != nil {
		return nil, err
	}
	ans.Append(p.absolute_colorimetric_adaptation(forward, rendering_intent)...)
	if b2a != nil {
		is_legacy_lab := false
		if mft, ok := b2a.(*MFT); ok && !mft.is8bit && p.Header.ProfileConnectionSpace == ColorSpaceLab {
			// Need to scale to the legacy LAB encoding used by lut16type
			// tags, see _cmsReadOutputLUT() in cmsio1.c
			is_legacy_lab = true
			ans.Append(NewLABToMFT2())
		}
		ans.Append(b2a)
		if is_legacy_lab && p.Header.DataColorSpace == ColorSpaceLab {
			ans.Append(NewLABFromMFT2())
		}
		if p.Header.ProfileConnectionSpace == ColorSpaceLab {
			// For some reason, lcms prefers trilinear over tetrahedral in this
			// case, see _cmsReadOutputLUT() in cmsio1.c
			ans.UseTrilinearInsteadOfTetrahedral()
		}
	} else {
		err = p.create_matrix_trc_transformer(forward, ans)
	}
	return
}

func (p *Profile) createTransformerToPCS(rendering_intent RenderingIntent) (ans *Pipeline, err error) {
	const forward = true
	ans = &Pipeline{}
	a2b, err := p.find_conversion_tag(forward, rendering_intent)
	if err != nil {
		return nil, err
	}
	if a2b != nil {
		ans.Append(a2b)
		if ans.has_lut16type_tag && p.Header.ProfileConnectionSpace == ColorSpaceLab {
			// Need to scale the lut16type data for legacy LAB encoding in ICC profiles
			if p.Header.DataColorSpace == ColorSpaceLab {
				ans.Insert(0, NewLABToMFT2())
			}
			ans.Append(NewLABFromMFT2())
		}
	} else {
		err = p.create_matrix_trc_transformer(forward, ans)
	}
	if err == nil {
		ans.Append(p.absolute_colorimetric_adaptation(forward, rendering_intent)...)
	}
	return
}

func (p *Profile) IsSRGB() bool {
	if p.Header.ProfileConnectionSpace == ColorSpaceXYZ {
		tr, err := p.createTransformerToPCS(p.Header.RenderingIntent)
		if err != nil {
			return false
		}
		tr.finalize(true)
		return tr.IsXYZSRGB()
	}
	return false
}

func transform_for_pcs_colorspace(cs ColorSpace, forward bool) ChannelTransformer {
	switch cs {
	case ColorSpaceXYZ:
		if forward {
			return NewNormalizedToXYZ()
		}
		return NewXYZToNormalized()
	case ColorSpaceLab:
		if forward {
			return NewNormalizedToLAB()
		}
		return NewLABToNormalized()
	default:
		panic(fmt.Sprintf("unsupported PCS colorspace in profile: %s", cs))
	}
}

func (p *Profile) CreateTransformerToPCS(rendering_intent RenderingIntent, input_channels int, optimize bool) (ans *Pipeline, err error) {
	ans, err = p.createTransformerToPCS(rendering_intent)
	if err == nil && !ans.IsSuitableFor(input_channels, 3) {
		err = fmt.Errorf("transformer to PCS %s not suitable for %d input channels", ans.String(), input_channels)
	}
	if err == nil {
		ans.Append(transform_for_pcs_colorspace(p.Header.ProfileConnectionSpace, true))
		ans.finalize(optimize)
	}
	return
}

func (p *Profile) CreateTransformerToSRGB(rendering_intent RenderingIntent, use_blackpoint_compensation bool, input_channels int, clamp, map_gamut, optimize bool) (ans *Pipeline, err error) {
	if ans, err = p.createTransformerToPCS(rendering_intent); err != nil {
		return
	}
	if !ans.IsSuitableFor(input_channels, 3) {
		return nil, fmt.Errorf("transformer to PCS %s not suitable for %d input channels", ans.String(), input_channels)
	}
	input_colorspace := p.Header.ProfileConnectionSpace
	if p.effective_bpc(rendering_intent, use_blackpoint_compensation) {
		var sRGB_blackpoint XYZType // 0, 0, 0
		input_blackpoint := p.BlackPoint(rendering_intent, nil)
		if input_blackpoint != sRGB_blackpoint {
			if input_colorspace == ColorSpaceLab {
				ans.Append(transform_for_pcs_colorspace(input_colorspace, true))
				ans.Append(NewLABtoXYZ(p.PCSIlluminant))
				ans.Append(NewXYZToNormalized())
				input_colorspace = ColorSpaceXYZ
			}
			ans.Append(NewBlackPointCorrection(p.PCSIlluminant, input_blackpoint, sRGB_blackpoint))
		}
	}
	ans.Append(transform_for_pcs_colorspace(input_colorspace, true))
	switch input_colorspace {
	case ColorSpaceXYZ:
		t := NewXYZtosRGB(p.PCSIlluminant, clamp, map_gamut)
		ans.Append(t)
	case ColorSpaceLab:
		ans.Append(NewLABtosRGB(p.PCSIlluminant, clamp, map_gamut))
	default:
		return nil, fmt.Errorf("unknown PCS colorspace: %s", input_colorspace)
	}
	ans.finalize(optimize)
	return
}

func (p *Profile) CreateDefaultTransformerToDevice() (*Pipeline, error) {
	return p.CreateTransformerToDevice(p.Header.RenderingIntent, false, true)
}

func (p *Profile) CreateDefaultTransformerToPCS(input_channels int) (*Pipeline, error) {
	return p.CreateTransformerToPCS(p.Header.RenderingIntent, input_channels, true)
}

func newProfile() *Profile {
	return &Profile{
		TagTable:    emptyTagTable(),
		blackpoints: make(map[RenderingIntent]*XYZType),
	}
}

// Recursively generates all points in an m-dimensional hypercube.
// currentPoint stores the coordinates of the current point being built.
// dimension is the current dimension being processed (from 0 to m-1).
// m is the total number of dimensions.
// n is the number of points per dimension (0 to n-1).
func iterate_hypercube(currentPoint []int, dimension, m, n int, callback func([]int)) {
	// Base case: If all dimensions have been assigned, print the point.
	if dimension == m {
		callback(currentPoint)
		return
	}

	// Recursive step: Iterate through all possible values for the current dimension.
	for i := range n {
		currentPoint[dimension] = i // Assign value to the current dimension
		// Recursively call for the next dimension
		iterate_hypercube(currentPoint, dimension+1, m, n, callback)
	}
}

func points_for_transformer_comparison(input_channels, num_points_per_input_channel int) []unit_float {
	m, n := input_channels, num_points_per_input_channel
	sz := input_channels // n ** m * m
	for range m {
		sz *= n
	}
	ans := make([]unit_float, 0, sz)
	current_point := make([]int, input_channels)
	factor := 1 / unit_float(num_points_per_input_channel-1)
	iterate_hypercube(current_point, 0, m, n, func(p []int) {
		for _, x := range current_point {
			ans = append(ans, unit_float(x)*factor)
		}
	})
	if len(ans) != sz {
		panic(fmt.Sprintf("insufficient points: wanted %d, got %d", sz, len(ans)))
	}
	return ans
}

var Points_for_transformer_comparison3 = sync.OnceValue(func() []unit_float {
	return points_for_transformer_comparison(3, 16)
})
var Points_for_transformer_comparison4 = sync.OnceValue(func() []unit_float {
	return points_for_transformer_comparison(4, 16)
})

func DecodeProfile(r io.Reader) (ans *Profile, err error) {
	return NewProfileReader(r).ReadProfile()
}

func ReadProfile(path string) (ans *Profile, err error) {
	data, err := os.ReadFile(path)
	if err == nil {
		ans, err = NewProfileReader(bytes.NewReader(data)).ReadProfile()
	}
	return
}
