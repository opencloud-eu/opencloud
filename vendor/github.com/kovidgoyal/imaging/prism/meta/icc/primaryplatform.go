package icc

import "fmt"

type PrimaryPlatform uint32

const (
	PrimaryPlatformNone      PrimaryPlatform = 0x00000000
	PrimaryPlatformApple     PrimaryPlatform = 0x4150504C // 'AAPL'
	PrimaryPlatformMicrosoft PrimaryPlatform = 0x4D534654 // 'MSFT'
	PrimaryPlatformSGI       PrimaryPlatform = 0x53474920 // 'SGI '
	PrimaryPlatformSun       PrimaryPlatform = 0x53554E57 // 'SUNW'
)

func (pp PrimaryPlatform) String() string {
	switch pp {
	case PrimaryPlatformNone:
		return "None"
	case PrimaryPlatformApple:
		return "Apple Computer, Inc."
	case PrimaryPlatformMicrosoft:
		return "Microsoft Corporation"
	case PrimaryPlatformSGI:
		return "Silicon Graphics, Inc."
	case PrimaryPlatformSun:
		return "Sun Microsystems, Inc."
	default:
		return fmt.Sprintf("Unknown (%d)", Signature(pp))
	}
}
