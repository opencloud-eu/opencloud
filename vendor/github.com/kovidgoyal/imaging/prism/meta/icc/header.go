package icc

import (
	"fmt"
	"time"
)

type Header struct {
	ProfileSize            uint32
	PreferredCMM           Signature
	Version                Version
	DeviceClass            DeviceClass
	DataColorSpace         ColorSpace
	ProfileConnectionSpace ColorSpace
	CreatedAtRaw           [6]uint16
	FileSignature          Signature
	PrimaryPlatform        PrimaryPlatform
	Flags                  uint32
	DeviceManufacturer     Signature
	DeviceModel            Signature
	DeviceAttributes       uint64
	RenderingIntent        RenderingIntent
	PCSIlluminant          [3]uint32
	ProfileCreator         Signature
	ProfileID              [16]byte
	Reserved               [28]byte
}

func (h Header) CreatedAt() time.Time {
	b := h.CreatedAtRaw
	return time.Date(int(b[0]), time.Month(b[1]), int(b[2]), int(b[3]), int(b[4]), int(b[5]), 0, time.UTC)
}

func (h Header) Embedded() bool {
	return (h.Flags >> 31) != 0
}

func (h Header) DependsOnEmbeddedData() bool {
	return (h.Flags>>30)&1 != 0
}

func (h Header) String() string {
	return fmt.Sprintf("Header{PreferredCMM: %s, Version: %s, DeviceManufacturer: %s, DeviceModel: %s, ProfileCreator: %s, RenderingIntent: %s, CreatedAt: %v}", h.PreferredCMM, h.Version, h.DeviceManufacturer, h.DeviceModel, h.ProfileCreator, h.RenderingIntent, h.CreatedAt())
}
