package enmime

type EncoderOption interface {
	apply(p *Encoder)
}

// Encoder implements MIME part encoding options
type Encoder struct {
	forceQuotedPrintableCteOption bool
}

// ForceQuotedPrintableCte forces "quoted-printable" transfer encoding when selecting Content Transfer Encoding, preventing the use of base64.
func ForceQuotedPrintableCte(b bool) EncoderOption {
	return forceQuotedPrintableCteOption(b)
}

type forceQuotedPrintableCteOption bool

func (o forceQuotedPrintableCteOption) apply(p *Encoder) {
	p.forceQuotedPrintableCteOption = bool(o)
}

func NewEncoder(ops ...EncoderOption) *Encoder {
	e := Encoder{
		forceQuotedPrintableCteOption: false,
	}

	for _, o := range ops {
		o.apply(&e)
	}

	return &e
}
