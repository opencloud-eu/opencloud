package icc

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
)

// TrilinearInterpolate represents a color lookup table tag (TagColorLookupTable)
type TrilinearInterpolate struct {
	d      *interpolation_data
	legacy bool
}

type TetrahedralInterpolate struct {
	d      *interpolation_data
	legacy bool
}

type CLUT interface {
	ChannelTransformer
	Samples() []unit_float
}

func (c *TrilinearInterpolate) Samples() []unit_float   { return c.d.samples }
func (c *TetrahedralInterpolate) Samples() []unit_float { return c.d.samples }

func (c TetrahedralInterpolate) String() string {
	return fmt.Sprintf("TetrahedralInterpolate{ inp:%v outp:%v grid:%v values[:9]:%v }", c.d.num_inputs, c.d.num_outputs, c.d.grid_points, c.d.samples[:min(9, len(c.d.samples))])
}

func (c TrilinearInterpolate) String() string {
	return fmt.Sprintf("TrilinearInterpolate{ inp:%v outp:%v grid:%v values[:9]:%v }", c.d.num_inputs, c.d.num_outputs, c.d.grid_points, c.d.samples[:min(9, len(c.d.samples))])
}

var _ CLUT = (*TrilinearInterpolate)(nil)
var _ CLUT = (*TetrahedralInterpolate)(nil)

func decode_clut_table8(raw []byte, ans []unit_float) {
	for i, x := range raw {
		ans[i] = unit_float(x) / math.MaxUint8
	}
}

func decode_clut_table16(raw []byte, ans []unit_float) {
	raw = raw[:2*len(ans)]
	const inv = 1. / math.MaxUint16
	for i := range ans {
		val := binary.BigEndian.Uint16(raw)
		ans[i] = unit_float(val) * inv
		raw = raw[2:]
	}
}

func decode_clut_table(raw []byte, bytes_per_channel, OutputChannels int, grid_points []int, output_colorspace ColorSpace) (ans []unit_float, consumed int, err error) {
	expected_num_of_output_channels := 3
	switch output_colorspace {
	case ColorSpaceCMYK:
		expected_num_of_output_channels = 4
	}
	if expected_num_of_output_channels != OutputChannels {
		return nil, 0, fmt.Errorf("CLUT table number of output channels %d inappropriate for output_colorspace: %s", OutputChannels, output_colorspace)
	}
	expected_num_of_values := expectedValues(grid_points, OutputChannels)
	consumed = bytes_per_channel * expected_num_of_values
	if len(raw) < consumed {
		return nil, 0, fmt.Errorf("CLUT table too short %d < %d", len(raw), bytes_per_channel*expected_num_of_values)
	}
	ans = make([]unit_float, expected_num_of_values)
	if bytes_per_channel == 1 {
		decode_clut_table8(raw[:consumed], ans)
	} else {
		decode_clut_table16(raw[:consumed], ans)
	}
	return
}

func make_clut(grid_points []int, num_inputs, num_outputs int, samples []unit_float, legacy, prefer_trilinear bool) CLUT {
	if num_inputs >= 3 && !prefer_trilinear {
		return &TetrahedralInterpolate{make_interpolation_data(num_inputs, num_outputs, grid_points, samples), legacy}
	}
	return &TrilinearInterpolate{make_interpolation_data(num_inputs, num_outputs, grid_points, samples), legacy}
}

// section 10.12.3 (CLUT) in ICC.1-2202-05.pdf
func embeddedClutDecoder(raw []byte, InputChannels, OutputChannels int, output_colorspace ColorSpace, prefer_trilinear bool) (any, error) {
	if len(raw) < 20 {
		return nil, errors.New("clut tag too short")
	}
	if InputChannels > 4 {
		return nil, fmt.Errorf("clut supports at most 4 input channels not: %d", InputChannels)
	}
	gridPoints := make([]int, InputChannels)
	for i, b := range raw[:InputChannels] {
		gridPoints[i] = int(b)
	}
	for i, nPoints := range gridPoints {
		if nPoints < 2 {
			return nil, fmt.Errorf("CLUT input channel %d has invalid grid points: %d", i, nPoints)
		}
	}
	bytes_per_channel := raw[16]
	raw = raw[20:]
	values, _, err := decode_clut_table(raw, int(bytes_per_channel), OutputChannels, gridPoints, output_colorspace)
	if err != nil {
		return nil, err
	}
	return make_clut(gridPoints, InputChannels, OutputChannels, values, false, prefer_trilinear), nil
}

func expectedValues(gridPoints []int, outputChannels int) int {
	expectedPoints := 1
	for _, g := range gridPoints {
		expectedPoints *= int(g)
	}
	return expectedPoints * outputChannels
}

func (c *TrilinearInterpolate) IOSig() (int, int)                      { return c.d.num_inputs, c.d.num_outputs }
func (c *TetrahedralInterpolate) IOSig() (int, int)                    { return c.d.num_inputs, c.d.num_outputs }
func (c *TrilinearInterpolate) Iter(f func(ChannelTransformer) bool)   { f(c) }
func (c *TetrahedralInterpolate) Iter(f func(ChannelTransformer) bool) { f(c) }

func (c *TrilinearInterpolate) Transform(r, g, b unit_float) (unit_float, unit_float, unit_float) {
	var obuf [3]unit_float
	var ibuf = [3]unit_float{r, g, b}
	c.d.trilinear_interpolate(ibuf[:], obuf[:])
	return obuf[0], obuf[1], obuf[2]
}
func (m *TrilinearInterpolate) TransformGeneral(o, i []unit_float) {
	o = o[0:m.d.num_outputs:m.d.num_outputs]
	for i := range o {
		o[i] = 0
	}
	m.d.trilinear_interpolate(i[0:m.d.num_inputs:m.d.num_inputs], o)
}

func (c *TetrahedralInterpolate) Tetrahedral_interpolate(r, g, b unit_float) (unit_float, unit_float, unit_float) {
	var obuf [3]unit_float
	c.d.tetrahedral_interpolation(r, g, b, obuf[:])
	return obuf[0], obuf[1], obuf[2]
}

func (c *TetrahedralInterpolate) Transform(r, g, b unit_float) (unit_float, unit_float, unit_float) {
	return c.Tetrahedral_interpolate(r, g, b)
}

func (m *TetrahedralInterpolate) TransformGeneral(o, i []unit_float) {
	m.d.tetrahedral_interpolation4(i[0], i[1], i[2], i[3], o[:m.d.num_outputs:m.d.num_outputs])
}

func clamp01(v unit_float) unit_float {
	return max(0, min(v, 1))
}
