package webfinger

// Link represents a link relation object as per https://www.rfc-editor.org/rfc/rfc7033#section-4.4.4
type Link struct {
	// Rel is either a URI or a registered relation type (see RFC 5988)
	//
	// The "rel" member MUST be present in the link relation object.
	Rel string `json:"rel"`
	// Type indicates the media type of the target resource
	//
	// The "type" member is OPTIONAL in the link relation object.
	Type string `json:"type,omitempty"`
	// Href contains a URI pointing to the target resource.
	//
	// The "href" member is OPTIONAL in the link relation object.
	Href string `json:"href,omitempty"`
	// The "properties" object within the link relation object comprises
	// zero or more name/value pairs whose names are URIs (referred to as
	// "property identifiers") and whose values are strings or null.
	//
	// Properties are used to convey additional information about the link
	// relation.  As an example, consider this use of "properties":
	//
	//  "properties" : { "http://webfinger.example/mail/port" : "993" }
	//
	// The "properties" member is OPTIONAL in the link relation object.
	Properties map[string]string `json:"properties,omitempty"`
	// Titles comprises zero or more name/value pairs whose
	// names are a language tag or the string "und"
	//
	// Here is an example of the "titles" object:
	//
	//  "titles" :
	//  {
	//    "en-us" : "The Magical World of Steve",
	//    "fr" : "Le Monde Magique de Steve"
	//  }
	//
	// The "titles" member is OPTIONAL in the link relation object.
	Titles map[string]string `json:"titles,omitempty"`
}

// JSONResourceDescriptor represents a JSON Resource Descriptor (JRD) as per https://www.rfc-editor.org/rfc/rfc7033#section-4.4
type JSONResourceDescriptor struct {
	// Subject is a URI that identifies the entity that the JRD describes
	//
	// The "subject" member SHOULD be present in the JRD.
	Subject string `json:"subject,omitempty"`
	// Aliases is an array of zero or more URI strings that identify the same
	// entity as the "subject" URI.
	//
	// The "aliases" array is OPTIONAL in the JRD.
	Aliases []string `json:"aliases,omitempty"`
	// Properties is an object comprising zero or more name/value pairs whose
	// names are URIs (referred to as "property identifiers") and whose
	// values are strings or null.
	//
	// The "properties" member is OPTIONAL in the JRD.
	Properties map[string]any `json:"properties,omitempty"`
	// Links is an array of objects that contain link relation information
	//
	// The "links" array is OPTIONAL in the JRD.
	Links []Link `json:"links,omitempty"`
}
