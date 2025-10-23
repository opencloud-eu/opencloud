package icc

import "fmt"

type DeviceClass uint32

const (
	DeviceClassInput      DeviceClass = 0x73636E72 // 'scnr'
	DeviceClassDisplay    DeviceClass = 0x6D6E7472 // 'mntr'
	DeviceClassOutput     DeviceClass = 0x70727472 // 'prtr'
	DeviceClassLink       DeviceClass = 0x6C696E6B // 'link'
	DeviceClassColorSpace DeviceClass = 0x73706163 // 'spac'
	DeviceClassAbstract   DeviceClass = 0x61627374 // 'abst'
	DeviceClassNamedColor DeviceClass = 0x6E6D636C // 'nmcl'
)

func (dc DeviceClass) String() string {
	switch dc {
	case DeviceClassInput:
		return "Input"
	case DeviceClassDisplay:
		return "Display"
	case DeviceClassOutput:
		return "Output"
	case DeviceClassLink:
		return "Device link"
	case DeviceClassColorSpace:
		return "Color space"
	case DeviceClassAbstract:
		return "Abstract"
	case DeviceClassNamedColor:
		return "Named color"
	default:
		return fmt.Sprintf("Unknown (%s)", Signature(dc))
	}
}
