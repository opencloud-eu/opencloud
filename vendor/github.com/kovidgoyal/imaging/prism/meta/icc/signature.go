package icc

type Signature uint32

const (
	UnknownSignature     Signature = 0
	ProfileFileSignature Signature = 0x61637370 // 'acsp'
	TextTagSignature     Signature = 0x74657874 // 'text'
	SignateTagSignature  Signature = 0x73696720 // 'sig '

	DescSignature                          Signature = 0x64657363 // 'desc'
	MultiLocalisedUnicodeSignature         Signature = 0x6D6C7563 // 'mluc'
	DeviceManufacturerDescriptionSignature Signature = 0x646d6e64 // 'dmnd'
	DeviceModelDescriptionSignature        Signature = 0x646d6464 // 'dmdd'

	AdobeManufacturerSignature      Signature = 0x41444245 // 'ADBE'
	AppleManufacturerSignature      Signature = 0x6170706c // 'appl'
	AppleUpperManufacturerSignature Signature = 0x4150504c // 'APPL'
	IECManufacturerSignature        Signature = 0x49454320 // 'IEC '

	AdobeRGBModelSignature  Signature = 0x52474220 // 'RGB '
	SRGBModelSignature      Signature = 0x73524742 // 'sRGB'
	PhotoProModelSignature  Signature = 0x50525452 // 'PTPR'
	DisplayP3ModelSignature Signature = 0x70332020 // 'p3  '

	ChromaticityTypeSignature          Signature = 0x6368726D /* 'chrm' */
	ColorantOrderTypeSignature         Signature = 0x636C726F /* 'clro' */
	ColorantTableTypeSignature         Signature = 0x636C7274 /* 'clrt' */
	CrdInfoTypeSignature               Signature = 0x63726469 /* 'crdi' Removed in V4 */
	CurveTypeSignature                 Signature = 0x63757276 /* 'curv' */
	DataTypeSignature                  Signature = 0x64617461 /* 'data' */
	DictTypeSignature                  Signature = 0x64696374 /* 'dict' */
	DateTimeTypeSignature              Signature = 0x6474696D /* 'dtim' */
	DeviceSettingsTypeSignature        Signature = 0x64657673 /* 'devs' Removed in V4 */
	Lut16TypeSignature                 Signature = 0x6d667432 /* 'mft2' */
	Lut8TypeSignature                  Signature = 0x6d667431 /* 'mft1' */
	LutAtoBTypeSignature               Signature = 0x6d414220 /* 'mAB ' */
	LutBtoATypeSignature               Signature = 0x6d424120 /* 'mBA ' */
	MeasurementTypeSignature           Signature = 0x6D656173 /* 'meas' */
	MultiLocalizedUnicodeTypeSignature Signature = 0x6D6C7563 /* 'mluc' */
	MultiProcessElementTypeSignature   Signature = 0x6D706574 /* 'mpet' */
	NamedColorTypeSignature            Signature = 0x6E636f6C /* 'ncol' OBSOLETE use ncl2 */
	NamedColor2TypeSignature           Signature = 0x6E636C32 /* 'ncl2' */
	ParametricCurveTypeSignature       Signature = 0x70617261 /* 'para' */
	ProfileSequenceDescTypeSignature   Signature = 0x70736571 /* 'pseq' */
	ProfileSequceIdTypeSignature       Signature = 0x70736964 /* 'psid' */
	ResponseCurveSet16TypeSignature    Signature = 0x72637332 /* 'rcs2' */
	S15Fixed16ArrayTypeSignature       Signature = 0x73663332 /* 'sf32' */
	ScreeningTypeSignature             Signature = 0x7363726E /* 'scrn' Removed in V4 */
	SignatureTypeSignature             Signature = 0x73696720 /* 'sig ' */
	TextTypeSignature                  Signature = 0x74657874 /* 'text' */
	TextDescriptionTypeSignature       Signature = 0x64657363 /* 'desc' Removed in V4 */
	U16Fixed16ArrayTypeSignature       Signature = 0x75663332 /* 'uf32' */
	UcrBgTypeSignature                 Signature = 0x62666420 /* 'bfd ' Removed in V4 */
	UInt16ArrayTypeSignature           Signature = 0x75693136 /* 'ui16' */
	UInt32ArrayTypeSignature           Signature = 0x75693332 /* 'ui32' */
	UInt64ArrayTypeSignature           Signature = 0x75693634 /* 'ui64' */
	UInt8ArrayTypeSignature            Signature = 0x75693038 /* 'ui08' */
	ViewingConditionsTypeSignature     Signature = 0x76696577 /* 'view' */
	XYZTypeSignature                   Signature = 0x58595A20 /* 'XYZ ' */
	XYZArrayTypeSignature              Signature = 0x58595A20 /* 'XYZ ' */

	XYZSignature  Signature = 0x58595A20 /* 'XYZ ' */
	LabSignature  Signature = 0x4C616220 /* 'Lab ' */
	LUVSignature  Signature = 0x4C757620 /* 'Luv ' */
	YCbrSignature Signature = 0x59436272 /* 'YCbr' */
	YxySignature  Signature = 0x59787920 /* 'Yxy ' */
	RGBSignature  Signature = 0x52474220 /* 'RGB ' */
	GraySignature Signature = 0x47524159 /* 'GRAY' */
	HSVSignature  Signature = 0x48535620 /* 'HSV ' */
	HLSSignature  Signature = 0x484C5320 /* 'HLS ' */
	CMYKSignature Signature = 0x434D594B /* 'CMYK' */
	CMYSignature  Signature = 0x434D5920 /* 'CMY ' */

	MCH2Signature  Signature = 0x32434C52 /* '2CLR' */
	MCH3Signature  Signature = 0x33434C52 /* '3CLR' */
	MCH4Signature  Signature = 0x34434C52 /* '4CLR' */
	MCH5Signature  Signature = 0x35434C52 /* '5CLR' */
	MCH6Signature  Signature = 0x36434C52 /* '6CLR' */
	MCH7Signature  Signature = 0x37434C52 /* '7CLR' */
	MCH8Signature  Signature = 0x38434C52 /* '8CLR' */
	MCH9Signature  Signature = 0x39434C52 /* '9CLR' */
	MCHASignature  Signature = 0x41434C52 /* 'ACLR' */
	MCHBSignature  Signature = 0x42434C52 /* 'BCLR' */
	MCHCSignature  Signature = 0x43434C52 /* 'CCLR' */
	MCHDSignature  Signature = 0x44434C52 /* 'DCLR' */
	MCHESignature  Signature = 0x45434C52 /* 'ECLR' */
	MCHFSignature  Signature = 0x46434C52 /* 'FCLR' */
	NamedSignature Signature = 0x6e6d636c /* 'nmcl' */

	Color2Signature  Signature = 0x32434C52 /* '2CLR' */
	Color3Signature  Signature = 0x33434C52 /* '3CLR' */
	Color4Signature  Signature = 0x34434C52 /* '4CLR' */
	Color5Signature  Signature = 0x35434C52 /* '5CLR' */
	Color6Signature  Signature = 0x36434C52 /* '6CLR' */
	Color7Signature  Signature = 0x37434C52 /* '7CLR' */
	Color8Signature  Signature = 0x38434C52 /* '8CLR' */
	Color9Signature  Signature = 0x39434C52 /* '9CLR' */
	Color10Signature Signature = 0x41434C52 /* 'ACLR' */
	Color11Signature Signature = 0x42434C52 /* 'BCLR' */
	Color12Signature Signature = 0x43434C52 /* 'CCLR' */
	Color13Signature Signature = 0x44434C52 /* 'DCLR' */
	Color14Signature Signature = 0x45434C52 /* 'ECLR' */
	Color15Signature Signature = 0x46434C52 /* 'FCLR' */

	AToB0TagSignature                          Signature = 0x41324230 /* 'A2B0' */
	AToB1TagSignature                          Signature = 0x41324231 /* 'A2B1' */
	AToB2TagSignature                          Signature = 0x41324232 /* 'A2B2' */
	AToB3TagSignature                          Signature = 0x41324233 /* 'A2B3' */
	BlueColorantTagSignature                   Signature = 0x6258595A /* 'bXYZ' */
	BlueMatrixColumnTagSignature               Signature = 0x6258595A /* 'bXYZ' */
	BlueTRCTagSignature                        Signature = 0x62545243 /* 'bTRC' */
	BToA0TagSignature                          Signature = 0x42324130 /* 'B2A0' */
	BToA1TagSignature                          Signature = 0x42324131 /* 'B2A1' */
	BToA2TagSignature                          Signature = 0x42324132 /* 'B2A2' */
	BToA3TagSignature                          Signature = 0x42324133 /* 'B2A3' */
	CalibrationDateTimeTagSignature            Signature = 0x63616C74 /* 'calt' */
	CharTargetTagSignature                     Signature = 0x74617267 /* 'targ' */
	ChromaticAdaptationTagSignature            Signature = 0x63686164 /* 'chad' */
	ChromaticityTagSignature                   Signature = 0x6368726D /* 'chrm' */
	ColorantOrderTagSignature                  Signature = 0x636C726F /* 'clro' */
	ColorantTableTagSignature                  Signature = 0x636C7274 /* 'clrt' */
	ColorantTableOutTagSignature               Signature = 0x636C6F74 /* 'clot' */
	ColorimetricIntentImageStateTagSignature   Signature = 0x63696973 /* 'ciis' */
	CopyrightTagSignature                      Signature = 0x63707274 /* 'cprt' */
	CrdInfoTagSignature                        Signature = 0x63726469 /* 'crdi' Removed in V4 */
	DataTagSignature                           Signature = 0x64617461 /* 'data' Removed in V4 */
	DateTimeTagSignature                       Signature = 0x6474696D /* 'dtim' Removed in V4 */
	DeviceMfgDescTagSignature                  Signature = 0x646D6E64 /* 'dmnd' */
	DeviceModelDescTagSignature                Signature = 0x646D6464 /* 'dmdd' */
	DeviceSettingsTagSignature                 Signature = 0x64657673 /* 'devs' Removed in V4 */
	DToB0TagSignature                          Signature = 0x44324230 /* 'D2B0' */
	DToB1TagSignature                          Signature = 0x44324231 /* 'D2B1' */
	DToB2TagSignature                          Signature = 0x44324232 /* 'D2B2' */
	DToB3TagSignature                          Signature = 0x44324233 /* 'D2B3' */
	BToD0TagSignature                          Signature = 0x42324430 /* 'B2D0' */
	BToD1TagSignature                          Signature = 0x42324431 /* 'B2D1' */
	BToD2TagSignature                          Signature = 0x42324432 /* 'B2D2' */
	BToD3TagSignature                          Signature = 0x42324433 /* 'B2D3' */
	GamutTagSignature                          Signature = 0x67616D74 /* 'gamt' */
	GrayTRCTagSignature                        Signature = 0x6b545243 /* 'kTRC' */
	GreenColorantTagSignature                  Signature = 0x6758595A /* 'gXYZ' */
	GreenMatrixColumnTagSignature              Signature = 0x6758595A /* 'gXYZ' */
	GreenTRCTagSignature                       Signature = 0x67545243 /* 'gTRC' */
	LuminanceTagSignature                      Signature = 0x6C756d69 /* 'lumi' */
	MeasurementTagSignature                    Signature = 0x6D656173 /* 'meas' */
	MediaBlackPointTagSignature                Signature = 0x626B7074 /* 'bkpt' */
	MediaWhitePointTagSignature                Signature = 0x77747074 /* 'wtpt' */
	MetaDataTagSignature                       Signature = 0x6D657461 /* 'meta' */
	NamedColorTagSignature                     Signature = 0x6E636f6C /* 'ncol' OBSOLETE use ncl2 */
	NamedColor2TagSignature                    Signature = 0x6E636C32 /* 'ncl2' */
	OutputResponseTagSignature                 Signature = 0x72657370 /* 'resp' */
	PerceptualRenderingIntentGamutTagSignature Signature = 0x72696730 /* 'rig0' */
	Preview0TagSignature                       Signature = 0x70726530 /* 'pre0' */
	Preview1TagSignature                       Signature = 0x70726531 /* 'pre1' */
	Preview2TagSignature                       Signature = 0x70726532 /* 'pre2' */
	PrintConditionTagSignature                 Signature = 0x7074636e /* 'ptcn' */
	ProfileDescriptionTagSignature             Signature = 0x64657363 /* 'desc' */
	ProfileSequenceDescTagSignature            Signature = 0x70736571 /* 'pseq' */
	ProfileSequceIdTagSignature                Signature = 0x70736964 /* 'psid' */
	Ps2CRD0TagSignature                        Signature = 0x70736430 /* 'psd0' Removed in V4 */
	Ps2CRD1TagSignature                        Signature = 0x70736431 /* 'psd1' Removed in V4 */
	Ps2CRD2TagSignature                        Signature = 0x70736432 /* 'psd2' Removed in V4 */
	Ps2CRD3TagSignature                        Signature = 0x70736433 /* 'psd3' Removed in V4 */
	Ps2CSATagSignature                         Signature = 0x70733273 /* 'ps2s' Removed in V4 */
	Ps2RenderingIntentTagSignature             Signature = 0x70733269 /* 'ps2i' Removed in V4 */
	RedColorantTagSignature                    Signature = 0x7258595A /* 'rXYZ' */
	RedMatrixColumnTagSignature                Signature = 0x7258595A /* 'rXYZ' */
	RedTRCTagSignature                         Signature = 0x72545243 /* 'rTRC' */
	SaturationRenderingIntentGamutTagSignature Signature = 0x72696732 /* 'rig2' */
	ScreeningDescTagSignature                  Signature = 0x73637264 /* 'scrd' Removed in V4 */
	ScreeningTagSignature                      Signature = 0x7363726E /* 'scrn' Removed in V4 */
	TechnologyTagSignature                     Signature = 0x74656368 /* 'tech' */
	UcrBgTagSignature                          Signature = 0x62666420 /* 'bfd ' Removed in V4 */
	ViewingCondDescTagSignature                Signature = 0x76756564 /* 'vued' */
	ViewingConditionsTagSignature              Signature = 0x76696577 /* 'view' */

	CurveSetElemTypeSignature Signature = 0x63767374 /* 'cvst' */
	MatrixElemTypeSignature   Signature = 0x6D617466 /* 'matf' */
	CLutElemTypeSignature     Signature = 0x636C7574 /* 'clut' */
	BAcsElemTypeSignature     Signature = 0x62414353 /* 'bACS' */
	EAcsElemTypeSignature     Signature = 0x65414353 /* 'eACS' */
)

func maskNull(b byte) byte {
	switch b {
	case 0:
		return ' '
	default:
		return b
	}
}

func signature(b []byte) Signature {
	return Signature(uint32(b[0])<<24 | uint32(b[1])<<16 | uint32(b[2])<<8 | uint32(b[3]))
}

func SignatureFromString(sig string) Signature {
	var b []byte = []byte{0x20, 0x20, 0x20, 0x20}
	copy(b, sig)
	return signature(b)
}

func (s Signature) String() string {
	v := []byte{
		(maskNull(byte((s >> 24) & 0xff))),
		(maskNull(byte((s >> 16) & 0xff))),
		(maskNull(byte((s >> 8) & 0xff))),
		(maskNull(byte(s & 0xff))),
	}
	return "'" + string(v) + "'"
}
