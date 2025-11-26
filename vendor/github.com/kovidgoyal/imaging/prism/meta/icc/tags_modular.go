package icc

import (
	"encoding/binary"
	"errors"
	"fmt"
	"slices"
)

// ModularTag represents a modular tag section 10.12 and 10.13 of ICC.1-2202-05.pdf
type ModularTag struct {
	num_input_channels, num_output_channels int
	a_curves, m_curves, b_curves            []Curve1D
	clut, matrix                            ChannelTransformer
	transform_objects                       []ChannelTransformer
	is_a_to_b                               bool
}

func (m ModularTag) String() string {
	return fmt.Sprintf("%s{ %s }", IfElse(m.is_a_to_b, "mAB", "mBA"), transformers_as_string(m.transform_objects...))
}

var _ ChannelTransformer = (*ModularTag)(nil)

func (m *ModularTag) Iter(f func(ChannelTransformer) bool) {
	for _, c := range m.transform_objects {
		if !f(c) {
			break
		}
	}
}

func (m *ModularTag) IOSig() (i int, o int) {
	i, _ = m.transform_objects[0].IOSig()
	_, o = m.transform_objects[len(m.transform_objects)-1].IOSig()
	return
}

func (m *ModularTag) IsSuitableFor(num_input_channels, num_output_channels int) bool {
	return m.num_input_channels == num_input_channels && m.num_output_channels == num_output_channels
}
func (m *ModularTag) Transform(r, g, b unit_float) (unit_float, unit_float, unit_float) {
	for _, t := range m.transform_objects {
		r, g, b = t.Transform(r, g, b)
	}
	return r, g, b
}

func (m *ModularTag) TransformGeneral(o, i []unit_float) {
	for _, t := range m.transform_objects {
		t.TransformGeneral(o, i)
	}
}

func IfElse[T any](condition bool, if_val T, else_val T) T {
	if condition {
		return if_val
	}
	return else_val
}

func modularDecoder(raw []byte, _, output_colorspace ColorSpace) (ans any, err error) {
	if len(raw) < 40 {
		return nil, errors.New("modular (mAB/mBA) tag too short")
	}
	var s Signature
	_, _ = binary.Decode(raw[:4], binary.BigEndian, &s)
	is_a_to_b := false
	switch s {
	case LutAtoBTypeSignature:
		is_a_to_b = true
	case LutBtoATypeSignature:
		is_a_to_b = false
	default:
		return nil, fmt.Errorf("modular tag has unknown signature: %s", s)
	}
	inputCh, outputCh := int(raw[8]), int(raw[9])
	var offsets [5]uint32
	if _, err := binary.Decode(raw[12:], binary.BigEndian, offsets[:]); err != nil {
		return nil, err
	}
	b, matrix, m, clut, a := offsets[0], offsets[1], offsets[2], offsets[3], offsets[4]
	mt := &ModularTag{num_input_channels: inputCh, num_output_channels: outputCh, is_a_to_b: is_a_to_b}
	read_curves := func(offset uint32, num_curves_reqd int) (ans []Curve1D, err error) {
		if offset == 0 {
			return nil, nil
		}
		if int(offset)+8 > len(raw) {
			return nil, errors.New("modular (mAB/mBA) tag too short")
		}
		block := raw[offset:]
		var c any
		var consumed int
		for range inputCh {
			if len(block) < 4 {
				return nil, errors.New("modular (mAB/mBA) tag too short")
			}
			sig := Signature(binary.BigEndian.Uint32(block[:4]))
			switch sig {
			case CurveTypeSignature:
				c, consumed, err = embeddedCurveDecoder(block)
			case ParametricCurveTypeSignature:
				c, consumed, err = embeddedParametricCurveDecoder(block)
			default:
				return nil, fmt.Errorf("unknown curve type: %s in modularDecoder", sig)
			}
			if err != nil {
				return nil, err
			}
			block = block[consumed:]
			ans = append(ans, c.(Curve1D))
		}
		if len(ans) != num_curves_reqd {
			return nil, fmt.Errorf("number of curves in modular tag: %d does not match the number of channels: %d", len(ans), num_curves_reqd)
		}
		return
	}
	if mt.b_curves, err = read_curves(b, IfElse(is_a_to_b, outputCh, inputCh)); err != nil {
		return nil, err
	}
	if mt.a_curves, err = read_curves(a, IfElse(is_a_to_b, inputCh, outputCh)); err != nil {
		return nil, err
	}
	if mt.m_curves, err = read_curves(m, outputCh); err != nil {
		return nil, err
	}
	var temp any
	if clut > 0 {
		if temp, err = embeddedClutDecoder(raw[clut:], inputCh, outputCh, output_colorspace, false); err != nil {
			return nil, err
		}
		mt.clut = temp.(ChannelTransformer)
	}
	if matrix > 0 {
		if temp, err = embeddedMatrixDecoder(raw[matrix:]); err != nil {
			return nil, err
		}
		if _, is_identity_matrix := temp.(*IdentityMatrix); !is_identity_matrix {
			mt.matrix = temp.(ChannelTransformer)
		}
	}
	ans = mt
	add_curves := func(name string, c []Curve1D) {
		if len(c) > 0 {
			has_non_identity := false
			for _, x := range c {
				if _, ok := x.(*IdentityCurve); !ok {
					has_non_identity = true
					break
				}
			}
			if has_non_identity {
				nc := NewCurveTransformer(name, c...)
				mt.transform_objects = append(mt.transform_objects, nc)
			}
		}
	}
	add_curves("A", mt.a_curves)
	if mt.clut != nil {
		mt.transform_objects = append(mt.transform_objects, mt.clut)
	}
	add_curves("M", mt.m_curves)
	if mt.matrix != nil {
		if mo, ok := mt.matrix.(*MatrixWithOffset); ok {
			if _, ok := mo.m.(*IdentityMatrix); !ok {
				mt.transform_objects = append(mt.transform_objects, mo.m)
			}
			if tt := mo.Translation(); tt != nil {
				mt.transform_objects = append(mt.transform_objects, tt)
			}
		} else {
			mt.transform_objects = append(mt.transform_objects, mt.matrix)
		}
	}
	add_curves("B", mt.b_curves)
	if !is_a_to_b {
		slices.Reverse(mt.transform_objects)
	}
	return
}
