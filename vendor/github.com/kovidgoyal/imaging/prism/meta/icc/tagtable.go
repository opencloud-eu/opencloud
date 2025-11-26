package icc

import (
	"errors"
	"fmt"
	"sync"
)

type not_found struct {
	sig Signature
}

func (e *not_found) Error() string {
	return fmt.Sprintf("no tag for signature: %s found in this ICC profile", e.sig)
}

type unsupported struct {
	sig Signature
}

func (e *unsupported) Error() string {
	return fmt.Sprintf("the tag: %s (0x%x) is not supported", e.sig, uint32(e.sig))
}

type XYZType struct{ X, Y, Z unit_float }

func xyz_type(data []byte) XYZType {
	return XYZType{readS15Fixed16BE(data[:4]), readS15Fixed16BE(data[4:8]), readS15Fixed16BE(data[8:12])}
}

func f(t unit_float) unit_float {
	const Limit = (24.0 / 116.0) * (24.0 / 116.0) * (24.0 / 116.0)

	if t <= Limit {
		return (841.0/108.0)*t + (16.0 / 116.0)
	}
	return pow(t, 1.0/3.0)
}

func f_1(t unit_float) unit_float {
	const Limit = (24.0 / 116.0)

	if t <= Limit {
		return (108.0 / 841.0) * (t - (16.0 / 116.0))
	}

	return t * t * t
}

func (wt *XYZType) Lab_to_XYZ(l, a, b unit_float) (x, y, z unit_float) {
	y = (l + 16.0) / 116.0
	x = y + 0.002*a
	z = y - 0.005*b

	x = f_1(x) * wt.X
	y = f_1(y) * wt.Y
	z = f_1(z) * wt.Z
	return
}

func (wt *XYZType) XYZ_to_Lab(x, y, z unit_float) (l, a, b unit_float) {
	fx := f(x / wt.X)
	fy := f(y / wt.Y)
	fz := f(z / wt.Z)

	l = 116.0*fy - 16.0
	a = 500.0 * (fx - fy)
	b = 200.0 * (fy - fz)
	return
}

func decode_xyz(data []byte) (ans any, err error) {
	if len(data) < 20 {
		return nil, fmt.Errorf("xyz tag too short")
	}
	a := xyz_type(data[8:])
	return &a, nil
}

func decode_array(data []byte) (ans any, err error) {
	data = data[8:]
	a := make([]unit_float, len(data)/4)
	for i := range a {
		a[i] = readS15Fixed16BE(data[:4:4])
		data = data[4:]
	}
	return a, nil
}

func parse_tag(sig Signature, data []byte, input_colorspace, output_colorspace ColorSpace) (result any, err error) {
	if len(data) == 0 {
		return nil, &not_found{sig}
	}
	if len(data) < 4 {
		return nil, &unsupported{sig}
	}
	s := signature(data)
	switch s {
	default:
		return nil, &unsupported{s}
	case DescSignature, DeviceManufacturerDescriptionSignature, DeviceModelDescriptionSignature, MultiLocalisedUnicodeSignature, TextTagSignature:
		return parse_text_tag(data)
	case SignateTagSignature:
		return sigDecoder(data)
	case MatrixElemTypeSignature:
		return matrixDecoder(data)
	case LutAtoBTypeSignature, LutBtoATypeSignature:
		return modularDecoder(data, input_colorspace, output_colorspace)
	case Lut16TypeSignature:
		return decode_mft16(data, input_colorspace, output_colorspace)
	case Lut8TypeSignature:
		return decode_mft8(data, input_colorspace, output_colorspace)
	case XYZTypeSignature:
		return decode_xyz(data)
	case S15Fixed16ArrayTypeSignature:
		return decode_array(data)
	case CurveTypeSignature:
		return curveDecoder(data)
	case ParametricCurveTypeSignature:
		return parametricCurveDecoder(data)
	}
}

type parsed_tag struct {
	tag any
	err error
}

type raw_tag_entry struct {
	offset int
	data   []byte
}

type parse_cache_key struct {
	offset, size int
}

type TagTable struct {
	entries     map[Signature]raw_tag_entry
	lock        sync.Mutex
	parsed      map[Signature]parsed_tag
	parse_cache map[parse_cache_key]parsed_tag
}

func (t *TagTable) Has(sig Signature) bool {
	return t.entries[sig].data != nil
}

func (t *TagTable) add(sig Signature, offset int, data []byte) {
	t.entries[sig] = raw_tag_entry{offset, data}
}

func (t *TagTable) get_parsed(sig Signature, input_colorspace, output_colorspace ColorSpace) (ans any, err error) {
	t.lock.Lock()
	defer t.lock.Unlock()
	if t.parsed == nil {
		t.parsed = make(map[Signature]parsed_tag)
		t.parse_cache = make(map[parse_cache_key]parsed_tag)
	}
	existing, found := t.parsed[sig]
	if found {
		return existing.tag, existing.err
	}
	var key parse_cache_key
	defer func() {
		t.parsed[sig] = parsed_tag{ans, err}
		t.parse_cache[key] = parsed_tag{ans, err}
	}()
	re := t.entries[sig]
	if re.data == nil {
		return nil, &not_found{sig}
	}
	key = parse_cache_key{re.offset, len(re.data)}
	if cached, ok := t.parse_cache[key]; ok {
		return cached.tag, cached.err
	}
	return parse_tag(sig, re.data, input_colorspace, output_colorspace)
}

func (t *TagTable) getDescription(s Signature) (string, error) {
	q, err := t.get_parsed(s, ColorSpaceRGB, ColorSpaceXYZ)
	if err != nil {
		return "", fmt.Errorf("could not get description for %s with error: %w", s, err)
	}
	if t, ok := q.(TextTag); ok {
		return t.BestGuessValue(), nil
	} else {
		return "", fmt.Errorf("tag for %s is not a text tag", s)
	}
}

func (t *TagTable) getProfileDescription() (string, error) {
	return t.getDescription(DescSignature)
}

func (t *TagTable) getDeviceManufacturerDescription() (string, error) {
	return t.getDescription(DeviceManufacturerDescriptionSignature)
}

func (t *TagTable) getDeviceModelDescription() (string, error) {
	return t.getDescription(DeviceModelDescriptionSignature)
}

func (t *TagTable) load_curve_tag(s Signature) (Curve1D, error) {
	r, err := t.get_parsed(s, ColorSpaceRGB, ColorSpaceXYZ)
	if err != nil {
		return nil, fmt.Errorf("could not load %s tag from profile with error: %w", s, err)
	}
	if ans, ok := r.(Curve1D); !ok {
		return nil, fmt.Errorf("could not load %s tag from profile as it is of unsupported type: %T", s, r)
	} else {
		if _, ok := r.(*IdentityCurve); ok {
			return nil, nil
		}
		return ans, nil
	}
}

func (t *TagTable) load_rgb_matrix(forward bool) (ans *Matrix3, err error) {
	r, err := t.get_parsed(RedMatrixColumnTagSignature, ColorSpaceRGB, ColorSpaceXYZ)
	if err != nil {
		return nil, err
	}
	g, err := t.get_parsed(GreenMatrixColumnTagSignature, ColorSpaceRGB, ColorSpaceXYZ)
	if err != nil {
		return nil, err
	}
	b, err := t.get_parsed(BlueMatrixColumnTagSignature, ColorSpaceRGB, ColorSpaceXYZ)
	if err != nil {
		return nil, err
	}
	rc, bc, gc := r.(*XYZType), g.(*XYZType), b.(*XYZType)
	var m Matrix3
	m[0][0], m[0][1], m[0][2] = rc.X, bc.X, gc.X
	m[1][0], m[1][1], m[1][2] = rc.Y, bc.Y, gc.Y
	m[2][0], m[2][1], m[2][2] = rc.Z, bc.Z, gc.Z
	// stored in 2.15 format so need to scale, see
	// BuildRGBInputMatrixShaper in lcms
	m.Scale(MAX_ENCODEABLE_XYZ_INVERSE)

	if is_identity_matrix(&m) {
		return nil, nil
	}
	if forward {
		return &m, nil
	}
	inv, err := m.Inverted()
	if err != nil {
		return nil, fmt.Errorf("the colorspace conversion matrix is not invertible: %w", err)
	}
	return &inv, nil
}

func array_to_matrix(a []unit_float) *Matrix3 {
	_ = a[8]
	m := Matrix3{}
	copy(m[0][:], a[:3])
	copy(m[1][:], a[3:6])
	copy(m[2][:], a[6:9])
	if is_identity_matrix(&m) {
		return nil
	}
	return &m
}

func (p *TagTable) get_chromatic_adaption() (*Matrix3, error) {
	x, err := p.get_parsed(ChromaticAdaptationTagSignature, ColorSpaceRGB, ColorSpaceXYZ)
	if err != nil {
		var nf *not_found
		if errors.As(err, &nf) {
			return nil, nil
		}
		return nil, err
	}
	a, ok := x.([]unit_float)
	if !ok {
		return nil, fmt.Errorf("chad tag is not an ArrayType")
	}
	return array_to_matrix(a), nil
}

func emptyTagTable() TagTable {
	return TagTable{
		entries: make(map[Signature]raw_tag_entry),
	}
}

type Debug_callback = func(r, g, b, x, y, z unit_float, t ChannelTransformer)

type ChannelTransformer interface {
	Transform(r, g, b unit_float) (x, y, z unit_float)
	TransformGeneral(out, in []unit_float)
	IOSig() (num_inputs, num_outputs int)
	// Should yield only itself unless it is a container, in which case it should yield its contained transforms
	Iter(func(ChannelTransformer) bool)
	String() string
}
