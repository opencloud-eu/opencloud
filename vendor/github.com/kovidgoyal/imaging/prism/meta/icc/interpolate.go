package icc

import (
	"fmt"
	"math"
)

var _ = fmt.Print

type interpolation_data struct {
	num_inputs, num_outputs  int
	samples                  []unit_float
	grid_points              []int
	max_grid_points          []int
	tetrahedral_index_lookup []int
}

func make_interpolation_data(num_inputs, num_outputs int, grid_points []int, samples []unit_float) *interpolation_data {
	var tetrahedral_index_lookup [4]int
	max_grid_points := make([]int, len(grid_points))
	for i, g := range grid_points {
		max_grid_points[i] = g - 1
	}
	if num_inputs >= 3 {
		tetrahedral_index_lookup[0] = num_outputs
		for i := 1; i < num_inputs; i++ {
			tetrahedral_index_lookup[i] = tetrahedral_index_lookup[i-1] * grid_points[num_inputs-1]
		}
	}
	return &interpolation_data{
		num_inputs: num_inputs, num_outputs: num_outputs, grid_points: grid_points, max_grid_points: max_grid_points,
		tetrahedral_index_lookup: tetrahedral_index_lookup[:], samples: samples,
	}
}

func (c *interpolation_data) tetrahedral_interpolation(r, g, b unit_float, output []unit_float) {
	r, g, b = clamp01(r), clamp01(g), clamp01(b)
	px := r * unit_float(c.max_grid_points[0])
	py := g * unit_float(c.max_grid_points[1])
	pz := b * unit_float(c.max_grid_points[2])
	x0, y0, z0 := int(px), int(py), int(pz)
	rx, ry, rz := px-unit_float(x0), py-unit_float(y0), pz-unit_float(z0)

	X0 := c.tetrahedral_index_lookup[2] * x0
	X1 := X0
	if r < 1 {
		X1 += c.tetrahedral_index_lookup[2]
	}
	Y0 := c.tetrahedral_index_lookup[1] * y0
	Y1 := Y0
	if g < 1 {
		Y1 += c.tetrahedral_index_lookup[1]
	}
	Z0 := c.tetrahedral_index_lookup[0] * z0
	Z1 := Z0
	if b < 1 {
		Z1 += c.tetrahedral_index_lookup[0]
	}
	type w struct{ a, b int }
	var c1, c2, c3 w
	c0 := X0 + Y0 + Z0
	// The six tetrahedra
	switch {
	case rx >= ry && ry >= rz:
		c1 = w{X1 + Y0 + Z0, c0}
		c2 = w{X1 + Y1 + Z0, X1 + Y0 + Z0}
		c3 = w{X1 + Y1 + Z1, X1 + Y1 + Z0}
	case rx >= rz && rz >= ry:
		c1 = w{X1 + Y0 + Z0, c0}
		c2 = w{X1 + Y1 + Z1, X1 + Y0 + Z1}
		c3 = w{X1 + Y0 + Z1, X1 + Y0 + Z0}
	case rz >= rx && rx >= ry:
		c1 = w{X1 + Y0 + Z1, X0 + Y0 + Z1}
		c2 = w{X1 + Y1 + Z1, X1 + Y0 + Z1}
		c3 = w{X0 + Y0 + Z1, c0}
	case ry >= rx && rx >= rz:
		c1 = w{X1 + Y1 + Z0, X0 + Y1 + Z0}
		c2 = w{X0 + Y1 + Z0, c0}
		c3 = w{X1 + Y1 + Z1, X1 + Y1 + Z0}
	case ry >= rz && rz >= rx:
		c1 = w{X1 + Y1 + Z1, X0 + Y1 + Z1}
		c2 = w{X0 + Y1 + Z0, c0}
		c3 = w{X0 + Y1 + Z1, X0 + Y1 + Z0}
	case rz >= ry && ry >= rx:
		c1 = w{X1 + Y1 + Z1, X0 + Y1 + Z1}
		c2 = w{X0 + Y1 + Z1, X0 + Y0 + Z1}
		c3 = w{X0 + Y0 + Z1, c0}
	}
	for o := range c.num_outputs {
		s := c.samples[o:]
		output[o] = s[c0] + (s[c1.a]-s[c1.b])*rx + (s[c2.a]-s[c2.b])*ry + (s[c3.a]-s[c3.b])*rz
	}
}

// For more that 3 inputs (i.e., CMYK)
// evaluate two 3-dimensional interpolations and then linearly interpolate between them.
func (d *interpolation_data) tetrahedral_interpolation4(c, m, y, k unit_float, output []unit_float) {
	var tmp1, tmp2 [4]float64
	pk := clamp01(c) * unit_float(d.max_grid_points[0])
	k0 := int(math.Trunc(pk))
	rest := pk - unit_float(k0)

	K0 := d.tetrahedral_index_lookup[3] * k0
	K1 := K0 + IfElse(c >= 1, 0, d.tetrahedral_index_lookup[3])

	half := *d
	half.grid_points = half.grid_points[1:]
	half.max_grid_points = half.max_grid_points[1:]

	half.samples = d.samples[K0:]
	half.tetrahedral_interpolation(m, y, k, tmp1[:len(output)])

	half.samples = d.samples[K1:]
	half.tetrahedral_interpolation(m, y, k, tmp2[:len(output)])

	for i := range output {
		y0, y1 := tmp1[i], tmp2[i]
		output[i] = y0 + (y1-y0)*rest
	}
}

func sampled_value(samples []unit_float, max_idx unit_float, x unit_float) unit_float {
	idx := clamp01(x) * max_idx
	lof := unit_float(math.Trunc(float64(idx)))
	lo := int(lof)
	if lof == idx {
		return samples[lo]
	}
	p := idx - unit_float(lo)
	vhi := unit_float(samples[lo+1])
	vlo := unit_float(samples[lo])
	return vlo + p*(vhi-vlo)
}

// Performs an n-linear interpolation on the CLUT values for the given input color using an iterative method.
// Input values should be normalized between 0.0 and 1.0. Output MUST be zero initialized.
func (c *interpolation_data) trilinear_interpolate(input, output []unit_float) {
	// Pre-allocate slices for indices and weights
	var buf [4]int
	var wbuf [4]unit_float
	indices := buf[:c.num_inputs]
	weights := wbuf[:c.num_inputs]
	input = input[:c.num_inputs]
	output = output[:c.num_outputs]

	// Calculate the base indices and interpolation weights for each dimension.
	for i, val := range input {
		val = clamp01(val)
		// Scale the value to the grid dimensions
		pos := val * unit_float(c.max_grid_points[i])
		// The base index is the floor of the position.
		idx := int(pos)
		// The weight is the fractional part of the position.
		weight := pos - unit_float(idx)
		// Clamp index to be at most the second to last grid point.
		if idx >= c.max_grid_points[i] {
			idx = c.max_grid_points[i] - 1
			weight = 1 // set weight to 1 for border index
		}
		indices[i] = idx
		weights[i] = weight
	}
	// Iterate through all 2^InputChannels corners of the n-dimensional hypercube
	for i := range 1 << c.num_inputs {
		// Calculate the combined weight for this corner
		cornerWeight := unit_float(1)
		// Calculate the N-dimensional index to look up in the table
		tableIndex := 0
		multiplier := unit_float(1)

		// As per section 10.12.3 of ICC.1-2022-5.pdf spec the first input channel
		// varies least rapidly and the last varies most rapidly
		for j := c.num_inputs - 1; j >= 0; j-- {
			// Check the j-th bit of i to decide if we are at the lower or upper bound for this dimension
			if (i>>j)&1 == 1 {
				// Upper bound for this dimension
				cornerWeight *= weights[j]
				tableIndex += int(unit_float(indices[j]+1) * multiplier)
			} else {
				// Lower bound for this dimension
				cornerWeight *= (1.0 - weights[j])
				tableIndex += int(unit_float(indices[j]) * multiplier)
			}
			multiplier *= unit_float(c.grid_points[j])
		}
		// Get the color value from the table for the current corner
		offset := tableIndex * c.num_outputs
		// Add the weighted corner color to the output
		for k, v := range c.samples[offset : offset+c.num_outputs] {
			output[k] += v * cornerWeight
		}
	}
}
