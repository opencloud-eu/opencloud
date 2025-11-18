package icc

import (
	"errors"
	"fmt"
	"math"
)

type Translation [3]unit_float
type Matrix3 [3][3]unit_float
type IdentityMatrix int

type MatrixWithOffset struct {
	m                         ChannelTransformer
	offset1, offset2, offset3 unit_float
}

func (m MatrixWithOffset) String() string {
	return fmt.Sprintf("MatrixWithOffset{ %.6v %.6v }", m.m, []unit_float{m.offset1, m.offset2, m.offset3})
}

func is_identity_matrix(m *Matrix3) bool {
	if m == nil {
		return true
	}
	for r := range 3 {
		for c := range 3 {
			q := IfElse(r == c, unit_float(1), unit_float(0))
			if math.Abs(float64(m[r][c]-q)) > FLOAT_EQUALITY_THRESHOLD {
				return false
			}
		}
	}
	return true
}

func (c Translation) String() string                             { return fmt.Sprintf("Translation{%.6v}", [3]unit_float(c)) }
func (c *Translation) IOSig() (int, int)                         { return 3, 3 }
func (c *Translation) Empty() bool                               { return c[0] == 0 && c[1] == 0 && c[2] == 0 }
func (c *Translation) Iter(f func(ChannelTransformer) bool)      { f(c) }
func (c *IdentityMatrix) String() string                         { return "IdentityMatrix" }
func (c *IdentityMatrix) IOSig() (int, int)                      { return 3, 3 }
func (c *MatrixWithOffset) IOSig() (int, int)                    { return 3, 3 }
func (c *Matrix3) IOSig() (int, int)                             { return 3, 3 }
func (c *IdentityMatrix) Iter(f func(ChannelTransformer) bool)   { f(c) }
func (c *MatrixWithOffset) Iter(f func(ChannelTransformer) bool) { f(c) }
func (c *Matrix3) Iter(f func(ChannelTransformer) bool)          { f(c) }

var _ ChannelTransformer = (*MatrixWithOffset)(nil)

func embeddedMatrixDecoder(body []byte) (any, error) {
	result := Matrix3{}
	if len(body) < 36 {
		return nil, fmt.Errorf("embedded matrix tag too short: %d < 36", len(body))
	}
	var m ChannelTransformer = &result
	for i := range 9 {
		result[i/3][i%3] = readS15Fixed16BE(body[:4])
		body = body[4:]
	}
	if is_identity_matrix(&result) {
		t := IdentityMatrix(0)
		m = &t
	}
	if len(body) < 3*4 {
		return m, nil
	}
	r2 := &MatrixWithOffset{m: m}
	r2.offset1 = readS15Fixed16BE(body[:4])
	r2.offset2 = readS15Fixed16BE(body[4:8])
	r2.offset3 = readS15Fixed16BE(body[8:12])
	if r2.offset1 == 0 && r2.offset2 == 0 && r2.offset3 == 0 {
		return m, nil
	}
	return r2, nil

}

func matrixDecoder(raw []byte) (any, error) {
	if len(raw) < 8+36 {
		return nil, errors.New("mtx tag too short")
	}
	return embeddedMatrixDecoder(raw[8:])
}

func (m *Matrix3) Transform(r, g, b unit_float) (unit_float, unit_float, unit_float) {
	return m[0][0]*r + m[0][1]*g + m[0][2]*b, m[1][0]*r + m[1][1]*g + m[1][2]*b, m[2][0]*r + m[2][1]*g + m[2][2]*b
}
func (m *Matrix3) TransformGeneral(o, i []unit_float) { tg33(m.Transform, o, i) }

func (m *Matrix3) Transpose() Matrix3 {
	return Matrix3{
		{m[0][0], m[1][0], m[2][0]},
		{m[0][1], m[1][1], m[2][1]},
		{m[0][2], m[1][2], m[2][2]},
	}
}

func (m *Matrix3) Scale(s unit_float) {
	m[0][0] *= s
	m[0][1] *= s
	m[0][2] *= s
	m[1][0] *= s
	m[1][1] *= s
	m[1][2] *= s
	m[2][0] *= s
	m[2][1] *= s
	m[2][2] *= s
}

func Dot(v1, v2 [3]unit_float) unit_float {
	return v1[0]*v2[0] + v1[1]*v2[1] + v1[2]*v2[2]
}

func (m *Matrix3) Equals(o *Matrix3, threshold unit_float) bool {
	for r := range 3 {
		ar, br := m[r], o[r]
		for c := range 3 {
			if abs(ar[c]-br[c]) > threshold {
				return false
			}
		}
	}
	return true
}

func (m *Matrix3) String() string {
	return fmt.Sprintf("Matrix3{ %.6v, %.6v, %.6v }", m[0], m[1], m[2])
}

func (m *Matrix3) AsMatrix3() *Matrix3 { return m }
func NewScalingMatrix3(scale unit_float) *Matrix3 {
	return &Matrix3{{scale, 0, 0}, {0, scale, 0}, {0, 0, scale}}
}
func (m *IdentityMatrix) AsMatrix3() *Matrix3 { return NewScalingMatrix3(1) }

// Return m * o
func (m *Matrix3) Multiply(o Matrix3) Matrix3 {
	t := o.Transpose()
	return Matrix3{
		{Dot(t[0], m[0]), Dot(t[1], m[0]), Dot(t[2], m[0])},
		{Dot(t[0], m[1]), Dot(t[1], m[1]), Dot(t[2], m[1])},
		{Dot(t[0], m[2]), Dot(t[1], m[2]), Dot(t[2], m[2])},
	}
}

func (m *Matrix3) Inverted() (ans Matrix3, err error) {
	o := Matrix3{
		{
			m[1][1]*m[2][2] - m[2][1]*m[1][2],
			-(m[0][1]*m[2][2] - m[2][1]*m[0][2]),
			m[0][1]*m[1][2] - m[1][1]*m[0][2],
		},
		{
			-(m[1][0]*m[2][2] - m[2][0]*m[1][2]),
			m[0][0]*m[2][2] - m[2][0]*m[0][2],
			-(m[0][0]*m[1][2] - m[1][0]*m[0][2]),
		},
		{
			m[1][0]*m[2][1] - m[2][0]*m[1][1],
			-(m[0][0]*m[2][1] - m[2][0]*m[0][1]),
			m[0][0]*m[1][1] - m[1][0]*m[0][1],
		},
	}

	det := m[0][0]*o[0][0] + m[1][0]*o[0][1] + m[2][0]*o[0][2]
	if abs(det) < FLOAT_EQUALITY_THRESHOLD {
		return ans, fmt.Errorf("matrix is singular and cannot be inverted, det=%v", det)
	}
	det = 1 / det

	o[0][0] *= det
	o[0][1] *= det
	o[0][2] *= det
	o[1][0] *= det
	o[1][1] *= det
	o[1][2] *= det
	o[2][0] *= det
	o[2][1] *= det
	o[2][2] *= det
	return o, nil
}

func (m *Translation) Transform(r, g, b unit_float) (unit_float, unit_float, unit_float) {
	return r + m[0], g + m[1], b + m[2]
}

func (m *Translation) TransformGeneral(o, i []unit_float) { tg33(m.Transform, o, i) }

func (m IdentityMatrix) Transform(r, g, b unit_float) (unit_float, unit_float, unit_float) {
	return r, g, b
}
func (m IdentityMatrix) TransformGeneral(o, i []unit_float) { tg33(m.Transform, o, i) }

func (m *MatrixWithOffset) Translation() *Translation {
	if m.offset1 == 0 && m.offset2 == 0 && m.offset3 == 0 {
		return nil
	}
	return &Translation{m.offset1, m.offset2, m.offset3}
}

func (m *MatrixWithOffset) Transform(r, g, b unit_float) (unit_float, unit_float, unit_float) {
	r, g, b = m.m.Transform(r, g, b)
	r += m.offset1
	g += m.offset2
	b += m.offset3
	return r, g, b
}
func (m *MatrixWithOffset) TransformGeneral(o, i []unit_float) { tg33(m.Transform, o, i) }
