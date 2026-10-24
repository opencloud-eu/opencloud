package ldap

// This file contains the "Who Am I?" extended operation as specified in rfc 4532
//
// https://tools.ietf.org/html/rfc4532

// WhoAmIResult is returned by the WhoAmI() call
type WhoAmIResult struct {
	AuthzID string
}

// WhoAmI returns the authzId the server thinks we are, you may pass controls
// like a Proxied Authorization control
func (l *Conn) WhoAmI(controls []Control) (*WhoAmIResult, error) {
	extendedRequest := NewExtendedRequest(ControlTypeWhoAmI, nil)
	extendedRequest.Controls = controls
	resp, err := l.Extended(extendedRequest)
	if err != nil {
		return nil, err
	}

	// responseValue is OPTIONAL (RFC 4532); Extended leaves Value nil when the
	// server omits it. Guard the dereference and report an empty authzId.
	result := &WhoAmIResult{}
	if resp.Value != nil {
		result.AuthzID = resp.Value.Data.String()
	}
	return result, nil
}
