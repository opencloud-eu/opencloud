package icc

import (
	"encoding/binary"
	"errors"
	"fmt"
)

// CLUTTag represents a color lookup table tag (TagColorLookupTable)
type CLUTTag struct {
	GridPoints     []uint8 // e.g., [17,17,17] for 3D CLUT
	InputChannels  int
	OutputChannels int
	Values         []float64 // flattened [in1, in2, ..., out1, out2, ...]
}

var _ ChannelTransformer = (*CLUTTag)(nil)

// section 10.12.3 (CLUT) in ICC.1-2202-05.pdf
func embeddedClutDecoder(raw []byte, InputChannels, OutputChannels int) (any, error) {
	if len(raw) < 20 {
		return nil, errors.New("clut tag too short")
	}
	gridPoints := make([]uint8, InputChannels)
	copy(gridPoints, raw[:InputChannels])
	bytes_per_channel := raw[16]
	raw = raw[20:]
	// expected size: (product of grid points) * output channels * bytes_per_channel
	expected_num_of_values := expectedValues(gridPoints, OutputChannels)
	values := make([]float64, expected_num_of_values)
	if len(values)*int(bytes_per_channel) > len(raw) {
		return nil, fmt.Errorf("CLUT unexpected body length: expected %d, got %d", expected_num_of_values*int(bytes_per_channel), len(raw))
	}

	switch bytes_per_channel {
	case 1:
		for i, b := range raw[:len(values)] {
			values[i] = float64(b) / 255
		}
	case 2:
		for i := range len(values) {
			values[i] = float64(binary.BigEndian.Uint16(raw[i*2:i*2+2])) / 65535
		}
	}
	ans := &CLUTTag{
		GridPoints:     gridPoints,
		InputChannels:  InputChannels,
		OutputChannels: OutputChannels,
		Values:         values,
	}
	if ans.InputChannels > 6 {
		return nil, fmt.Errorf("unsupported num of CLUT input channels: %d", ans.InputChannels)
	}
	return ans, nil
}

func expectedValues(gridPoints []uint8, outputChannels int) int {
	expectedPoints := 1
	for _, g := range gridPoints {
		expectedPoints *= int(g)
	}
	return expectedPoints * outputChannels
}

func (c *CLUTTag) WorkspaceSize() int { return 16 }

func (c *CLUTTag) IsSuitableFor(num_input_channels, num_output_channels int) bool {
	return num_input_channels == int(c.InputChannels) && num_output_channels == c.OutputChannels
}

func (c *CLUTTag) Transform(output, workspace []float64, inputs ...float64) error {
	return c.Lookup(output, workspace, inputs)
}

func (c *CLUTTag) Lookup(output, workspace, inputs []float64) error {
	// clamp input values to 0-1...
	clamped := workspace[:len(inputs)]
	for i, v := range inputs {
		clamped[i] = clamp01(v)
	}
	// find the grid positions and interpolation factors...
	gridFrac := workspace[len(clamped) : 2*len(clamped)]
	var buf [4]int
	gridPos := buf[:]
	for i, v := range clamped {
		nPoints := int(c.GridPoints[i])
		if nPoints < 2 {
			return fmt.Errorf("CLUT input channel %d has invalid grid points: %d", i, nPoints)
		}
		pos := v * float64(nPoints-1)
		gridPos[i] = int(pos)
		if gridPos[i] >= nPoints-1 {
			gridPos[i] = nPoints - 2 // clamp
			gridFrac[i] = 1.0
		} else {
			gridFrac[i] = pos - float64(gridPos[i])
		}
	}
	// perform multi-dimensional interpolation (recursive)...
	return c.triLinearInterpolate(output[:c.OutputChannels], gridPos, gridFrac)
}

func (c *CLUTTag) triLinearInterpolate(out []float64, gridPos []int, gridFrac []float64) error {
	numCorners := 1 << c.InputChannels // 2^inputs
	for o := range c.OutputChannels {
		out[o] = 0
	}
	// walk all corners of the hypercube
	for corner := range numCorners {
		weight := 1.0
		idx := 0
		stride := 1
		for dim := c.InputChannels - 1; dim >= 0; dim-- {
			bit := (corner >> dim) & 1
			pos := gridPos[dim] + bit
			if pos >= int(c.GridPoints[dim]) {
				return fmt.Errorf("CLUT corner position out of bounds at dimension %d", dim)
			}
			idx += pos * stride
			stride *= int(c.GridPoints[dim])
			if bit == 0 {
				weight *= 1 - gridFrac[dim]
			} else {
				weight *= gridFrac[dim]
			}
		}
		base := idx * c.OutputChannels
		if base+c.OutputChannels > len(c.Values) {
			return errors.New("CLUT value index out of bounds")
		}
		for o := range c.OutputChannels {
			out[o] += weight * c.Values[base+o]
		}
	}
	return nil
}

func clamp01(v float64) float64 {
	return max(0, min(v, 1))
}
