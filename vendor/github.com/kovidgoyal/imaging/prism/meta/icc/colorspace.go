package icc

import "fmt"

type ColorSpace uint32

const (
	ColorSpaceXYZ     ColorSpace = 0x58595A20 // 'XYZ '
	ColorSpaceLab     ColorSpace = 0x4C616220 // 'Lab '
	ColorSpaceLuv     ColorSpace = 0x4C757620 // 'Luv '
	ColorSpaceYCbCr   ColorSpace = 0x59436272 // 'YCbr'
	ColorSpaceYxy     ColorSpace = 0x59787920 // 'Yxy '
	ColorSpaceRGB     ColorSpace = 0x52474220 // 'RGB '
	ColorSpaceGray    ColorSpace = 0x47524159 // 'Gray'
	ColorSpaceHSV     ColorSpace = 0x48535620 // 'HSV '
	ColorSpaceHLS     ColorSpace = 0x484C5320 // 'HLS '
	ColorSpaceCMYK    ColorSpace = 0x434D594B // 'CMYK'
	ColorSpaceCMY     ColorSpace = 0x434D5920 // 'CMY '
	ColorSpace2Color  ColorSpace = 0x32434C52 // '2CLR'
	ColorSpace3Color  ColorSpace = 0x33434C52 // '3CLR'
	ColorSpace4Color  ColorSpace = 0x34434C52 // '4CLR'
	ColorSpace5Color  ColorSpace = 0x35434C52 // '5CLR'
	ColorSpace6Color  ColorSpace = 0x36434C52 // '6CLR'
	ColorSpace7Color  ColorSpace = 0x37434C52 // '7CLR'
	ColorSpace8Color  ColorSpace = 0x38434C52 // '8CLR'
	ColorSpace9Color  ColorSpace = 0x39434C52 // '9CLR'
	ColorSpace10Color ColorSpace = 0x41434C52 // 'ACLR'
	ColorSpace11Color ColorSpace = 0x42434C52 // 'BCLR'
	ColorSpace12Color ColorSpace = 0x43434C52 // 'CCLR'
	ColorSpace13Color ColorSpace = 0x44434C52 // 'DCLR'
	ColorSpace14Color ColorSpace = 0x45434C52 // 'ECLR'
	ColorSpace15Color ColorSpace = 0x46434C52 // 'FCLR'
)

func (cs ColorSpace) String() string {
	switch cs {
	case ColorSpaceXYZ:
		return "XYZ"
	case ColorSpaceLab:
		return "Lab"
	case ColorSpaceLuv:
		return "Luv"
	case ColorSpaceYCbCr:
		return "YCbCr"
	case ColorSpaceYxy:
		return "Yxy"
	case ColorSpaceRGB:
		return "RGB"
	case ColorSpaceGray:
		return "Gray"
	case ColorSpaceHSV:
		return "HSV"
	case ColorSpaceHLS:
		return "HLS"
	case ColorSpaceCMYK:
		return "CMYK"
	case ColorSpaceCMY:
		return "CMY"
	case ColorSpace2Color:
		return "2 color"
	case ColorSpace3Color:
		return "3 color"
	case ColorSpace4Color:
		return "4 color"
	case ColorSpace5Color:
		return "5 color"
	case ColorSpace6Color:
		return "6 color"
	case ColorSpace7Color:
		return "7 color"
	case ColorSpace8Color:
		return "8 color"
	case ColorSpace9Color:
		return "9 color"
	case ColorSpace10Color:
		return "10 color"
	case ColorSpace11Color:
		return "11 color"
	case ColorSpace12Color:
		return "12 color"
	case ColorSpace13Color:
		return "13 color"
	case ColorSpace14Color:
		return "14 color"
	case ColorSpace15Color:
		return "15 color"
	default:
		return fmt.Sprintf("Unknown (%s)", Signature(cs))
	}
}
