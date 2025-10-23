package icc

import (
	"fmt"
	"sync"
)

type not_found struct {
	sig Signature
}

func (e *not_found) Error() string {
	return fmt.Sprintf("no tag for signature: %s found in this ICC profile", e.sig)
}

type unsupported struct {
	sig Signature
}

func (e *unsupported) Error() string {
	return fmt.Sprintf("the tag: %s is not supported", e.sig)
}

func parse_tag(sig Signature, data []byte) (result any, err error) {
	if len(data) == 0 {
		return nil, &not_found{sig}
	}
	switch sig {
	default:
		return nil, &unsupported{sig}
	case DescSignature, DeviceManufacturerDescriptionSignature, DeviceModelDescriptionSignature:
		return parse_text_tag(data)
	case SignateTagSignature:
		return sigDecoder(data)
	}
}

type parsed_tag struct {
	tag any
	err error
}

type TagTable struct {
	entries map[Signature][]byte
	lock    sync.Mutex
	parsed  map[Signature]parsed_tag
}

func (t *TagTable) add(sig Signature, data []byte) {
	t.entries[sig] = data
}

func (t *TagTable) get_parsed(sig Signature) (ans any, err error) {
	t.lock.Lock()
	defer t.lock.Unlock()
	existing, found := t.parsed[sig]
	if found {
		return existing.tag, existing.err
	}
	if t.parsed == nil {
		t.parsed = make(map[Signature]parsed_tag)
	}
	defer func() {
		t.parsed[sig] = parsed_tag{ans, err}
	}()
	return parse_tag(sig, t.entries[sig])
}

func (t *TagTable) getDescription(s Signature) (string, error) {
	q, err := t.get_parsed(s)
	if err != nil {
		return "", fmt.Errorf("could not get description for %s with error: %w", s, err)
	}
	if t, ok := q.(TextTag); ok {
		return t.BestGuessValue(), nil
	} else {
		return "", fmt.Errorf("tag for %s is not a text tag", s)
	}
}

func (t *TagTable) getProfileDescription() (string, error) {
	return t.getDescription(DescSignature)
}

func (t *TagTable) getDeviceManufacturerDescription() (string, error) {
	return t.getDescription(DeviceManufacturerDescriptionSignature)
}

func (t *TagTable) getDeviceModelDescription() (string, error) {
	return t.getDescription(DeviceModelDescriptionSignature)
}

func emptyTagTable() TagTable {
	return TagTable{
		entries: make(map[Signature][]byte),
	}
}

type ChannelTransformer interface {
	Transform(output, workspace []float64, input ...float64) error
	IsSuitableFor(num_input_channels int, num_output_channels int) bool
	WorkspaceSize() int
}
