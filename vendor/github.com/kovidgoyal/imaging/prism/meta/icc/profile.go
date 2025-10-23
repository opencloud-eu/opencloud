package icc

type WellKnownProfile int

const (
	UnknownProfile WellKnownProfile = iota
	SRGBProfile
	AdobeRGBProfile
	PhotoProProfile
	DisplayP3Profile
)

func WellKnownProfileFromDescription(x string) WellKnownProfile {
	switch x {
	case "sRGB IEC61966-2.1", "sRGB_ICC_v4_Appearance.icc":
		return SRGBProfile
	case "Adobe RGB (1998)":
		return AdobeRGBProfile
	case "Display P3":
		return DisplayP3Profile
	case "ProPhoto RGB":
		return PhotoProProfile
	default:
		return UnknownProfile
	}
}

func (p WellKnownProfile) String() string {
	switch p {
	case SRGBProfile:
		return "sRGB IEC61966-2.1"
	case AdobeRGBProfile:
		return "Adobe RGB (1998)"
	case PhotoProProfile:
		return "ProPhoto RGB"
	case DisplayP3Profile:
		return "Display P3"
	default:
		return "Unknown Profile"
	}
}

type Profile struct {
	Header   Header
	TagTable TagTable
}

func (p *Profile) Description() (string, error) {
	return p.TagTable.getProfileDescription()
}

func (p *Profile) DeviceManufacturerDescription() (string, error) {
	return p.TagTable.getDeviceManufacturerDescription()
}

func (p *Profile) DeviceModelDescription() (string, error) {
	return p.TagTable.getDeviceModelDescription()
}

func (p *Profile) WellKnownProfile() WellKnownProfile {
	model, err := p.DeviceModelDescription()
	if err == nil {
		switch model {
		case "IEC 61966-2-1 Default RGB Colour Space - sRGB":
			return SRGBProfile
		}
	}
	d, err := p.Description()
	if err == nil {
		if ans := WellKnownProfileFromDescription(d); ans != UnknownProfile {
			return ans
		}
	}
	switch p.Header.DeviceManufacturer {
	case IECManufacturerSignature:
		switch p.Header.DeviceModel {
		case SRGBModelSignature:
			return SRGBProfile
		}
	case AdobeManufacturerSignature:
		switch p.Header.DeviceModel {
		case AdobeRGBModelSignature:
			return AdobeRGBProfile
		case PhotoProModelSignature:
			return PhotoProProfile
		}
	case AppleManufacturerSignature, AppleUpperManufacturerSignature:
		switch p.Header.DeviceModel {
		case DisplayP3ModelSignature:
			return DisplayP3Profile
		}
	}
	return UnknownProfile
}

func newProfile() *Profile {
	return &Profile{
		TagTable: emptyTagTable(),
	}
}
