package jscontact

import (
	"time"
)

const (
	JSContactVersion = "1.0"

	AddressType           = "Address"
	AddressComponentType  = "AddressComponent"
	AnniversaryType       = "Anniversary"
	AuthorType            = "Author"
	ContactCardType       = "Card"
	CalendarType          = "Calendar"
	CryptoKeyType         = "CryptoKey"
	DirectoryType         = "Directory"
	EmailAddressType      = "EmailAddress"
	LanguagePrefType      = "LanguagePref"
	LinkType              = "Link"
	MediaType             = "Media"
	NameType              = "Name"
	NameComponentType     = "NameComponent"
	NicknameType          = "Nickname"
	NoteType              = "Note"
	OnlineServiceType     = "OnlineService"
	OrganizationType      = "Organization"
	OrgUnitType           = "OrgUnit"
	PartialDateType       = "PartialDate"
	PersonalInfoType      = "PersonalInfo"
	PhoneType             = "Phone"
	PronounsType          = "Pronouns"
	RelationType          = "Relation"
	SchedulingAddressType = "SchedulingAddress"
	SpeakToAsType         = "SpeakToAs"
	TimestampType         = "Timestamp"
	TitleType             = "Title"

	JSContactTypeAddress           = AddressType
	JSContactTypeAddressComponent  = AddressComponentType
	JSContactTypeAnniversary       = AnniversaryType
	JSContactTypeAuthor            = AuthorType
	JSContactTypeCard              = ContactCardType
	JSContactTypeCalendar          = CalendarType
	JSContactTypeCryptoKey         = CryptoKeyType
	JSContactTypeDirectory         = DirectoryType
	JSContactTypeEmailAddress      = EmailAddressType
	JSContactTypeLanguagePref      = LanguagePrefType
	JSContactTypeLink              = LinkType
	JSContactTypeMedia             = MediaType
	JSContactTypeName              = NameType
	JSContactTypeNameComponent     = NameComponentType
	JSContactTypeNickname          = NicknameType
	JSContactTypeNote              = NoteType
	JSContactTypeOnlineService     = OnlineServiceType
	JSContactTypeOrganization      = OrganizationType
	JSContactTypeOrgUnit           = OrgUnitType
	JSContactTypePartialDate       = PartialDateType
	JSContactTypePersonalInfo      = PersonalInfoType
	JSContactTypePhone             = PhoneType
	JSContactTypePronouns          = PronounsType
	JSContactTypeRelation          = RelationType
	JSContactTypeSchedulingAddress = SchedulingAddressType
	JSContactTypeSpeakToAs         = SpeakToAsType
	JSContactTypeTimestamp         = TimestampType
	JSContactTypeTitle             = TitleType

	ResourceTypeCalendar  = JSContactTypeCalendar
	ResourceTypeCryptoKey = JSContactTypeCryptoKey
	ResourceTypeDirectory = JSContactTypeDirectory
	ResourceTypeLink      = JSContactTypeLink
	ResourceTypeMedia     = JSContactTypeMedia

	CalendarResourceKindCalendar = "calendar"
	CalendarResourceKindFreeBusy = "freeBusy"

	LinkResourceKindContact = "contact"

	MediaResourceKindPhoto = "photo"
	MediaResourceKindSound = "sound"
	MediaResourceKindLogo  = "logo"

	ResourceKindCalendar = CalendarResourceKindCalendar
	ResourceKindFreeBusy = CalendarResourceKindFreeBusy
	ResourceKindContact  = LinkResourceKindContact
	ResourceKindPhoto    = MediaResourceKindPhoto
	ResourceKindSound    = MediaResourceKindSound
	ResourceKindLogo     = MediaResourceKindLogo

	AddressContextBilling  = "billing"
	AddressContextDelivery = "delivery"
	AddressContextPrivate  = "private"
	AddressContextWork     = "work"

	CalendarContextPrivate = "private"
	CalendarContextWork    = "work"

	CryptoKeyContextPrivate = "private"
	CryptoKeyContextWork    = "work"

	DirectoryContextPrivate = "private"
	DirectoryContextWork    = "work"

	EmailAddressContextPrivate = "private"
	EmailAddressContextWork    = "work"

	LanguagePrefContextPrivate = "private"
	LanguagePrefContextWork    = "work"

	LinkContextPrivate = "private"
	LinkContextWork    = "work"

	MediaContextPrivate = "private"
	MediaContextWork    = "work"

	NicknameContextPrivate = "private"
	NicknameContextWork    = "work"

	OnlineServiceContextPrivate = "private"
	OnlineServiceContextWork    = "work"

	OrganizationContextPrivate = "private"
	OrganizationContextWork    = "work"

	PhoneContextPrivate = "private"
	PhoneContextWork    = "work"

	PronounsContextPrivate = "private"
	PronounsContextWork    = "work"

	SchedulingAddressContextPrivate = "private"
	SchedulingAddressContextWork    = "work"

	ResourceContextPrivate         = "private"
	ResourceContextWork            = "work"
	ResourceContextAddressBilling  = "billing"
	ResourceContextAddressDelivery = "delivery"

	ContactCardKindIndividual  = "individual"
	ContactCardKindGroup       = "group"
	ContactCardKindOrg         = "org"
	ContactCardKindLocation    = "location"
	ContactCardKindDevice      = "device"
	ContactCardKindApplication = "application"

	RelationAcquaintance = "acquaintance"
	RelationAgent        = "agent"
	RelationChild        = "child"
	RelationCoResident   = "co-resident"
	RelationCoWorker     = "co-worker"
	RelationColleague    = "colleague"
	RelationContact      = "contact"
	RelationCrush        = "crush"
	RelationDate         = "date"
	RelationEmergency    = "emergency"
	RelationFriend       = "friend"
	RelationKin          = "kin"
	RelationMe           = "me"
	RelationMet          = "met"
	RelationMuse         = "muse"
	RelationNeighbor     = "neighbor"
	RelationParent       = "parent"
	RelationSibling      = "sibling"
	RelationSpouse       = "spouse"
	RelationSweetheart   = "sweetheart"

	GrammaticalGenderAnimate   = "animate"
	GrammaticalGenderCommon    = "common"
	GrammaticalGenderFeminine  = "feminine"
	GrammaticalGenderInanimate = "inanimate"
	GrammaticalGenderMasculine = "masculine"
	GrammaticalGenderNeuter    = "neuter"

	DirectoryResourceKindDirectory = "directory"
	DirectoryResourceKindEntry     = "entry"

	TitleKindTitle = "title"
	TitleKindRole  = "role"

	PhoneFeatureMobile     = "mobile"
	PhoneFeatureVoice      = "voice"
	PhoneFeatureText       = "text"
	PhoneFeatureVideo      = "video"
	PhoneFeatureMainNumber = "main-number"
	PhoneFeatureTextPhone  = "textphone"
	PhoneFeatureFax        = "fax"
	PhoneFeaturePager      = "pager"

	AddressComponentKindRoom          = "room"
	AddressComponentKindApartment     = "apartment"
	AddressComponentKindFloor         = "floor"
	AddressComponentKindBuilding      = "building"
	AddressComponentKindNumber        = "number"
	AddressComponentKindName          = "name"
	AddressComponentKindBlock         = "block"
	AddressComponentKindSubdistrict   = "subdistrict"
	AddressComponentKindDistrict      = "district"
	AddressComponentKindLocality      = "locality"
	AddressComponentKindRegion        = "region"
	AddressComponentKindPostcode      = "postcode"
	AddressComponentKindCountry       = "country"
	AddressComponentKindDirection     = "direction"
	AddressComponentKindLandmark      = "landmark"
	AddressComponentKindPostOfficeBox = "postOfficeBox"
	AddressComponentKindSeparator     = "separator"

	AnniversaryKindBirth   = "birth"
	AnniversaryKindDeath   = "death"
	AnniversaryKindWedding = "wedding"

	PersonalInfoKindExpertise = "expertise"
	PersonalInfoKindHobby     = "hobby"
	PersonalInfoKindInterest  = "interest"
	PersonalInfoLevelHigh     = "high"
	PersonalInfoLevelMedium   = "medium"
	PersonalInfoLevelLow      = "low"

	NameComponentKindTitle      = "title"
	NameComponentKindGiven      = "given"
	NameComponentKindGiven2     = "given2"
	NameComponentKindSurname    = "surname"
	NameComponentKindSurname2   = "surname2"
	NameComponentKindCredential = "credential"
	NameComponentKindGeneration = "generation"
	NameComponentKindSeparator  = "separator"
)

var (
	JSContactTypes = []string{
		JSContactTypeAddress,
		JSContactTypeAddressComponent,
		JSContactTypeAnniversary,
		JSContactTypeAuthor,
		JSContactTypeCard,
		JSContactTypeCalendar,
		JSContactTypeCryptoKey,
		JSContactTypeDirectory,
		JSContactTypeEmailAddress,
		JSContactTypeLanguagePref,
		JSContactTypeLink,
		JSContactTypeMedia,
		JSContactTypeName,
		JSContactTypeNameComponent,
		JSContactTypeNickname,
		JSContactTypeNote,
		JSContactTypeOnlineService,
		JSContactTypeOrganization,
		JSContactTypeOrgUnit,
		JSContactTypePartialDate,
		JSContactTypePersonalInfo,
		JSContactTypePhone,
		JSContactTypePronouns,
		JSContactTypeRelation,
		JSContactTypeSchedulingAddress,
		JSContactTypeSpeakToAs,
		JSContactTypeTimestamp,
		JSContactTypeTitle,
	}
	AddressContexts = []string{
		AddressContextBilling,
		AddressContextDelivery,
		AddressContextPrivate,
		AddressContextWork,
	}
	CalendarContexts = []string{
		CalendarContextPrivate,
		CalendarContextWork,
	}

	CryptoKeyContexts = []string{
		CryptoKeyContextPrivate,
		CryptoKeyContextWork,
	}

	DirectoryContexts = []string{
		DirectoryContextPrivate,
		DirectoryContextWork,
	}

	EmailAddressContexts = []string{
		EmailAddressContextPrivate,
		EmailAddressContextWork,
	}

	LanguagePrefContexts = []string{
		LanguagePrefContextPrivate,
		LanguagePrefContextWork,
	}

	LinkContexts = []string{
		LinkContextPrivate,
		LinkContextWork,
	}

	MediaContexts = []string{
		MediaContextPrivate,
		MediaContextWork,
	}

	NicknameContexts = []string{
		NicknameContextPrivate,
		NicknameContextWork,
	}

	OnlineServiceContexts = []string{
		OnlineServiceContextPrivate,
		OnlineServiceContextWork,
	}

	OrganizationContexts = []string{
		OrganizationContextPrivate,
		OrganizationContextWork,
	}

	PhoneContexts = []string{
		PhoneContextPrivate,
		PhoneContextWork,
	}

	PronounsContexts = []string{
		PronounsContextPrivate,
		PronounsContextWork,
	}

	SchedulingAddressContexts = []string{
		SchedulingAddressContextPrivate,
		SchedulingAddressContextWork,
	}

	ResourceTypes = []string{
		ResourceTypeCalendar,
		ResourceTypeCryptoKey,
		ResourceTypeDirectory,
		ResourceTypeLink,
		ResourceTypeMedia,
	}

	CalendarResourceKinds = []string{
		CalendarResourceKindCalendar,
		CalendarResourceKindFreeBusy,
	}

	ResourceKinds = []string{
		ResourceKindCalendar,
		ResourceKindFreeBusy,
		ResourceKindContact,
		ResourceKindPhoto,
		ResourceKindSound,
		ResourceKindLogo,
	}
	ResourceContexts = []string{
		ResourceContextPrivate,
		ResourceContextWork,
		ResourceContextAddressBilling,
		ResourceContextAddressDelivery,
	}
	ContactCardKinds = []string{
		ContactCardKindIndividual,
		ContactCardKindGroup,
		ContactCardKindOrg,
		ContactCardKindLocation,
		ContactCardKindDevice,
		ContactCardKindApplication,
	}
	Relations = []string{
		RelationAcquaintance,
		RelationAgent,
		RelationChild,
		RelationCoResident,
		RelationCoWorker,
		RelationColleague,
		RelationContact,
		RelationCrush,
		RelationDate,
		RelationEmergency,
		RelationFriend,
		RelationKin,
		RelationMe,
		RelationMet,
		RelationMuse,
		RelationNeighbor,
		RelationParent,
		RelationSibling,
		RelationSpouse,
		RelationSweetheart,
	}
	GrammaticalGenders = []string{
		GrammaticalGenderAnimate,
		GrammaticalGenderCommon,
		GrammaticalGenderFeminine,
		GrammaticalGenderInanimate,
		GrammaticalGenderMasculine,
		GrammaticalGenderNeuter,
	}
	TitleKinds = []string{
		TitleKindTitle,
		TitleKindRole,
	}
	PhoneFeatures = []string{
		PhoneFeatureMobile,
		PhoneFeatureVoice,
		PhoneFeatureText,
		PhoneFeatureVideo,
		PhoneFeatureMainNumber,
		PhoneFeatureTextPhone,
		PhoneFeatureFax,
		PhoneFeaturePager,
	}
	AddressComponentKinds = []string{
		AddressComponentKindRoom,
		AddressComponentKindApartment,
		AddressComponentKindFloor,
		AddressComponentKindBuilding,
		AddressComponentKindNumber,
		AddressComponentKindName,
		AddressComponentKindBlock,
		AddressComponentKindSubdistrict,
		AddressComponentKindDistrict,
		AddressComponentKindLocality,
		AddressComponentKindRegion,
		AddressComponentKindPostcode,
		AddressComponentKindCountry,
		AddressComponentKindDirection,
		AddressComponentKindLandmark,
		AddressComponentKindPostOfficeBox,
		AddressComponentKindSeparator,
	}
	AnniversaryKinds = []string{
		AnniversaryKindBirth,
		AnniversaryKindDeath,
		AnniversaryKindWedding,
	}
	PersonalInfoKinds = []string{
		PersonalInfoKindExpertise,
		PersonalInfoKindHobby,
		PersonalInfoKindInterest,
	}
	PersonalInfoLevels = []string{
		PersonalInfoLevelHigh,
		PersonalInfoLevelMedium,
		PersonalInfoLevelLow,
	}
	DirectoryResourceKinds = []string{
		DirectoryResourceKindDirectory,
		DirectoryResourceKindEntry,
	}
	NameComponentKinds = []string{
		NameComponentKindTitle,
		NameComponentKindGiven,
		NameComponentKindGiven2,
		NameComponentKindSurname,
		NameComponentKindSurname2,
		NameComponentKindCredential,
		NameComponentKindGeneration,
		NameComponentKindSeparator,
	}
	LinkResourceKinds = []string{
		LinkResourceKindContact,
	}
	MediaResourceKinds = []string{
		MediaResourceKindPhoto,
		MediaResourceKindSound,
		MediaResourceKindLogo,
	}
)

// A `PatchObject` is of type `String[*]` and represents an unordered set of patches on a JSON object.
//
// Each key is a path represented in a subset of the JSON Pointer format [RFC6901].
//
// The paths have an implicit leading `"/"`, so each key is prefixed with `"/"` before applying the
// JSON Pointer evaluation algorithm.
//
// A patch within a `PatchObject` is only valid if all the following conditions apply:
// !1. The pointer MAY reference inside an array, but if the last reference token in the pointer is an array index,
// then the patch value MUST NOT be null. The pointer MUST NOT use `"-"` as an array index in any of its reference
// tokens (i.e., you MUST NOT insert/delete from an array, but you MAY replace the contents of its existing members.
// To add or remove members, one needs to replace the complete array value).
// !2. All reference tokens prior to the last (i.e., the value after the final slash) MUST already exist as values
// in the object being patched. If the last reference token is an array index, then a member at this index MUST
// already exist in the referenced array.
// !3. There MUST NOT be two patches in the `PatchObject` where the pointer of
// one is the prefix of the pointer of the other, e.g., `"addresses/1/city"` and `"addresses"`.
// !4. The value for the patch MUST be valid for the property being set (of the correct type and obeying any
// other applicable restrictions), or if null, the property MUST be optional.
//
// The value associated with each pointerdetermines how to apply that patch:
// !- If null, remove the property from the patched object. If the key is not present in the parent, this is a no-op.
// !- If non-null, set the value given as the value for this property (this may be a replacement or addition to the
// object being patched).
//
// A `PatchObject` does not define its own `@type` property. Instead, the `@type` property in a patch MUST be handled
// as any other patched property value.
//
// Implementations MUST reject a `PatchObject` in its entirety if any of its patches are invalid.
//
// Implementations MUST NOT apply partial patches.
//
// [RFC6901]: https://www.rfc-editor.org/rfc/rfc6901.html
type PatchObject map[string]any

// The Resource data type defines a resource associated with the entity represented by the Card,
// identified by a URI [RFC3986].
//
// Several property definitions refer to the `Resource` type as the basis for their property-specific
// value types.
//
// The `Resource` type defines the properties that are common to all of them.
//
// Property definitions making use of `Resource` MAY define additional properties for their value types.
type Resource struct {
	// The JSContact type of the object.
	//
	// The value MUST NOT be "Resource"; instead, the value MUST be the name of a [concrete resource type].
	//
	// [concrete resource type]: https://www.rfc-editor.org/rfc/rfc9553.html#resource-properties
	Type string `json:"@type,omitempty"`

	// The kind of the resource.
	//
	// The allowed values are defined in the property definition that makes use of the Resource type.
	//
	// Some property definitions may change this property from being optional to mandatory.
	//
	// A contact card with a `kind` property equal to `group` represents a group of contacts.
	//
	// Clients often present these separately from other contact cards.
	//
	// The `members` property, as defined in [RFC 9553, Section 2.1.6], contains a set of UIDs for other
	// contacts that are the members of this group.
	//
	// Clients should consider the group to contain any `ContactCard` with a matching UID, from
	// any account they have access to with support for the `urn:ietf:params:jmap:contacts` capability.
	//
	// UIDs that cannot be found SHOULD be ignored but preserved.
	//
	// For example, suppose a user adds contacts from a shared address book to their private group, then
	// temporarily loses access to this address book. The UIDs cannot be resolved so the contacts will
	// disappear from the group. However, if they are given permission to access the data again the UIDs
	// will be found and the contacts will reappear.
	//
	// [RFC 9553, Section 2.1.8]: https://www.rfc-editor.org/rfc/rfc9553#members
	Kind string `json:"kind,omitempty"`

	// The resource value.
	//
	// This MUST be a URI as defined in Section 3 of [RFC3986-section3].
	//
	// [RFC3986-section3]: https://www.rfc-editor.org/rfc/rfc3986.html#section-3
	Uri string `json:"uri,omitempty"`

	// The [RFC2046 media type] of the resource identified by the uri property value.
	//
	// [RFC2046 media type]: https://www.rfc-editor.org/rfc/rfc2046.html
	MediaType string `json:"mediaType,omitempty"`

	// The contexts in which to use this resource.
	//
	// The contexts in which to use the contact information.
	//
	// For example, someone might have distinct phone numbers for work and private contexts and may set the
	// desired context on the respective phone number in the phones (Section 2.3.3) property.
	//
	// This section defines common contexts.
	//
	// Additional contexts may be defined in the properties or data types that make use of this property.
	//
	// The enumerated common context values are:
	// !- `private`: the contact information that may be used in a private context.
	// !- `work`: the contact information that may be used in a professional context.
	Contexts map[string]bool `json:"contexts,omitempty"`

	// The [preference] of the resource in relation to other resources.
	//
	// A preference order for contact information.
	//
	// For example, a person may have two email addresses and prefer to be contacted with one of them.
	//
	// The value MUST be in the range of 1 to 100. Lower values correspond to a higher level of preference, with 1
	// being most preferred.
	//
	// If no preference is set, then the contact information MUST be interpreted as being least preferred.
	//
	// Note that the preference is only defined in relation to contact information of the same type.
	//
	// For example, the preference orders within emails and phone numbers are independent of each other.
	//
	// [preference]: https://www.rfc-editor.org/rfc/rfc9553.html#prop-pref
	Pref uint `json:"pref,omitzero"`

	// A [custom label] for the value.
	//
	// The labels associated with the contact data.
	//
	// Such labels may be set for phone numbers, email addresses, and other resources.
	//
	// Typically, these labels are displayed along with their associated contact data in graphical user interfaces.
	//
	// Note that succinct labels are best for proper display on small graphical interfaces and screens.
	//
	// [custom label]: https://www.rfc-editor.org/rfc/rfc9553.html#prop-label
	Label string `json:"label,omitempty"`
}

type DirectoryResource struct {
	// The JSContact type of the object.
	//
	// The value MUST NOT be "Resource"; instead, the value MUST be the name of a [concrete resource type].
	//
	// [concrete resource type]: https://www.rfc-editor.org/rfc/rfc9553.html#resource-properties
	Type string `json:"@type,omitempty"`

	// The kind of the resource.
	//
	// The allowed values are defined in the property definition that makes use of the Resource type.
	//
	// Some property definitions may change this property from being optional to mandatory.
	//
	// A contact card with a `kind` property equal to `group` represents a group of contacts.
	//
	// Clients often present these separately from other contact cards.
	//
	// The `members` property, as defined in [RFC 9553, Section 2.1.6], contains a set of UIDs for other
	// contacts that are the members of this group.
	//
	// Clients should consider the group to contain any `ContactCard` with a matching UID, from
	// any account they have access to with support for the `urn:ietf:params:jmap:contacts` capability.
	//
	// UIDs that cannot be found SHOULD be ignored but preserved.
	//
	// For example, suppose a user adds contacts from a shared address book to their private group, then
	// temporarily loses access to this address book. The UIDs cannot be resolved so the contacts will
	// disappear from the group. However, if they are given permission to access the data again the UIDs
	// will be found and the contacts will reappear.
	//
	// [RFC 9553, Section 2.1.8]: https://www.rfc-editor.org/rfc/rfc9553#members
	Kind string `json:"kind,omitempty"`

	// The resource value.
	//
	// This MUST be a URI as defined in Section 3 of [RFC3986-section3].
	//
	// [RFC3986-section3]: https://www.rfc-editor.org/rfc/rfc3986.html#section-3
	Uri string `json:"uri,omitempty"`

	// The [RFC2046 media type] of the resource identified by the uri property value.
	//
	// [RFC2046 media type]: https://www.rfc-editor.org/rfc/rfc2046.html
	MediaType string `json:"mediaType,omitempty"`

	// The contexts in which to use this resource.
	//
	// The contexts in which to use the contact information.
	//
	// For example, someone might have distinct phone numbers for work and private contexts and may set the
	// desired context on the respective phone number in the phones (Section 2.3.3) property.
	//
	// This section defines common contexts.
	//
	// Additional contexts may be defined in the properties or data types that make use of this property.
	//
	// The enumerated common context values are:
	// !- `private`: the contact information that may be used in a private context.
	// !- `work`: the contact information that may be used in a professional context.
	Contexts map[string]bool `json:"contexts,omitempty"`

	// The [preference] of the resource in relation to other resources.
	//
	// A preference order for contact information.
	//
	// For example, a person may have two email addresses and prefer to be contacted with one of them.
	//
	// The value MUST be in the range of 1 to 100. Lower values correspond to a higher level of preference, with 1
	// being most preferred.
	//
	// If no preference is set, then the contact information MUST be interpreted as being least preferred.
	//
	// Note that the preference is only defined in relation to contact information of the same type.
	//
	// For example, the preference orders within emails and phone numbers are independent of each other.
	//
	// [preference]: https://www.rfc-editor.org/rfc/rfc9553.html#prop-pref
	Pref uint `json:"pref,omitzero"`

	// A [custom label] for the value.
	//
	// The labels associated with the contact data.
	//
	// Such labels may be set for phone numbers, email addresses, and other resources.
	//
	// Typically, these labels are displayed along with their associated contact data in graphical user interfaces.
	//
	// Note that succinct labels are best for proper display on small graphical interfaces and screens.
	//
	// [custom label]: https://www.rfc-editor.org/rfc/rfc9553.html#prop-label
	Label string `json:"label,omitempty"`

	// The position of the directory resource in the list of all `Directory` objects having the same kind property
	// value in the Card.
	//
	// Only in `Directory` `Resource` types.
	//
	// If set, the `listAs` value MUST be higher than zero.
	//
	// Multiple directory resources MAY have the same `listAs` property value or none.
	//
	// Sorting such same-valued entries is implementation-specific.
	ListAs uint `json:"listAs,omitzero"`
}

type MediaResource struct {
	// The JSContact type of the object.
	//
	// The value MUST NOT be "Resource"; instead, the value MUST be the name of a [concrete resource type].
	//
	// [concrete resource type]: https://www.rfc-editor.org/rfc/rfc9553.html#resource-properties
	Type string `json:"@type,omitempty"`

	// The kind of the resource.
	//
	// The allowed values are defined in the property definition that makes use of the Resource type.
	//
	// Some property definitions may change this property from being optional to mandatory.
	//
	// A contact card with a `kind` property equal to `group` represents a group of contacts.
	//
	// Clients often present these separately from other contact cards.
	//
	// The `members` property, as defined in [RFC 9553, Section 2.1.6], contains a set of UIDs for other
	// contacts that are the members of this group.
	//
	// Clients should consider the group to contain any `ContactCard` with a matching UID, from
	// any account they have access to with support for the `urn:ietf:params:jmap:contacts` capability.
	//
	// UIDs that cannot be found SHOULD be ignored but preserved.
	//
	// For example, suppose a user adds contacts from a shared address book to their private group, then
	// temporarily loses access to this address book. The UIDs cannot be resolved so the contacts will
	// disappear from the group. However, if they are given permission to access the data again the UIDs
	// will be found and the contacts will reappear.
	//
	// [RFC 9553, Section 2.1.8]: https://www.rfc-editor.org/rfc/rfc9553#members
	Kind string `json:"kind,omitempty"`

	// The resource value.
	//
	// This MUST be a URI as defined in Section 3 of [RFC3986-section3].
	//
	// [RFC3986-section3]: https://www.rfc-editor.org/rfc/rfc3986.html#section-3
	Uri string `json:"uri,omitempty"`

	// The [RFC2046 media type] of the resource identified by the uri property value.
	//
	// [RFC2046 media type]: https://www.rfc-editor.org/rfc/rfc2046.html
	MediaType string `json:"mediaType,omitempty"`

	// The contexts in which to use this resource.
	//
	// The contexts in which to use the contact information.
	//
	// For example, someone might have distinct phone numbers for work and private contexts and may set the
	// desired context on the respective phone number in the phones (Section 2.3.3) property.
	//
	// This section defines common contexts.
	//
	// Additional contexts may be defined in the properties or data types that make use of this property.
	//
	// The enumerated common context values are:
	// !- `private`: the contact information that may be used in a private context.
	// !- `work`: the contact information that may be used in a professional context.
	Contexts map[string]bool `json:"contexts,omitempty"`

	// The [preference] of the resource in relation to other resources.
	//
	// A preference order for contact information.
	//
	// For example, a person may have two email addresses and prefer to be contacted with one of them.
	//
	// The value MUST be in the range of 1 to 100. Lower values correspond to a higher level of preference, with 1
	// being most preferred.
	//
	// If no preference is set, then the contact information MUST be interpreted as being least preferred.
	//
	// Note that the preference is only defined in relation to contact information of the same type.
	//
	// For example, the preference orders within emails and phone numbers are independent of each other.
	//
	// [preference]: https://www.rfc-editor.org/rfc/rfc9553.html#prop-pref
	Pref uint `json:"pref,omitzero"`

	// A [custom label] for the value.
	//
	// The labels associated with the contact data.
	//
	// Such labels may be set for phone numbers, email addresses, and other resources.
	//
	// Typically, these labels are displayed along with their associated contact data in graphical user interfaces.
	//
	// Note that succinct labels are best for proper display on small graphical interfaces and screens.
	//
	// [custom label]: https://www.rfc-editor.org/rfc/rfc9553.html#prop-label
	Label string `json:"label,omitempty"`

	// An id for the Blob representing the binary contents of the resource.
	//
	// This is a JMAP extension of JSContact, and only present in `Media` `Resource` types.
	//
	// When returning `ContactCard`s, any `Media` with a `data:` URI SHOULD return a `blobId` property
	// and omit the `uri` property.
	//
	// The `mediaType` property MUST also be set.
	//
	// Similarly, when creating or updating a `ContactCard`, clients MAY send a `blobId` instead
	// of the `uri` property for a `Media` object.
	BlobId string `json:"blobId,omitempty"`
}

type Relation struct {
	// The JSContact type of the object: the value MUST be `Relation``, if set.
	Type string `json:"@type,omitempty"`

	// The relationship of the related Card to the Card, defined as a set of relation types.
	//
	// The keys in the set define the relation type; the values for each key in the set MUST be "true".
	//
	// The relationship between the two objects is undefined if the set is empty.
	//
	// The initial list of enumerated relation types matches the IANA-registered TYPE `IANA-vCard``
	// parameter values of the vCard RELATED property ([Section 6.6.6 of RFC6350]):
	// !- `acquaintance`
	// !- `agent`
	// !- `child`
	// !- `co-resident`
	// !- `co-worker`
	// !- `colleague`
	// !- `contact`
	// !- `crush`
	// !- `date`
	// !- `emergency`
	// !- `friend`
	// !- `kin`
	// !- `me`
	// !- `met`
	// !- `muse`
	// !- `neighbor`
	// !- `parent`
	// !- `sibling`
	// !- `spouse`
	// !- `sweetheart`
	//
	// [Section 6.6.6 of RFC6350]: https://www.rfc-editor.org/rfc/rfc6350.html#section-6.6.6
	Relation map[string]bool `json:"relation,omitempty"`
}

type NameComponent struct {
	// The JSContact type of the object: the value MUST be `NameComponent`, if set.
	Type string `json:"@type,omitempty"`

	// The value of the name component.
	//
	// This can be composed of one or multiple words such as `Poe` or `van Gogh`.
	Value string `json:"value"`

	// The kind of the name component.
	//
	// !- `title`: an honorific title or prefix, e.g., `Mr.`, `Ms.`, or `Dr.`
	// !- `given`: a given name, also known as "first name" or "personal name"
	// !- `given2`: a name that appears between the given and surname such as a middle name or patronymic name
	// !- `surname`: a surname, also known as "last name" or "family name"
	// !- `surname2`: a secondary surname (used in some cultures), also known as "maternal surname"
	// !- `credential`: a credential, also known as "accreditation qualifier" or "honorific suffix", e.g., `B.A.`, `Esq.`
	// !- `generation`: a generation marker or qualifier, e.g., `Jr.` or `III`
	// !- `separator`: a formatting separator between two ordered name non-separator components; the value property of the component includes the verbatim separator, for example, a hyphen character or even an empty string. This value has higher precedence than the defaultSeparator property of the Name. Implementations MUST NOT insert two consecutive separator components; instead, they SHOULD insert a single separator component with the combined value; this component kind MUST NOT be set if the `Name` `isOrdered` property value is `false`
	Kind string `json:"kind"`

	// The pronunciation of the name component.
	//
	// If this property is set, then at least one of the `Name` object properties, `phoneticSystem` or `phoneticScript`,
	// MUST be set.
	Phonetic string `json:"phonetic,omitempty"`
}

type Nickname struct {
	// The JSContact type of the object: the value MUST be `Nickname`, if set.
	Type string `json:"@type,omitempty"`

	// The nickname.
	Name string `json:"name"`

	// The contexts in which to use the nickname.
	// TODO document https://www.rfc-editor.org/rfc/rfc9553.html#prop-contexts
	Contexts map[string]bool `json:"contexts,omitempty"`

	// The preference of the nickname in relation to other nicknames.
	//
	// A preference order for contact information. For example, a person may have two email addresses and
	// prefer to be contacted with one of them.
	//
	// The value MUST be in the range of 1 to 100. Lower values correspond to a higher level of preference,
	// with 1 being most preferred.
	//
	// If no preference is set, then the contact information MUST be interpreted as being least preferred.
	//
	// Note that the preference is only defined in relation to contact information of the same type.
	//
	// For example, the preference orders within emails and phone numbers are independent of each other.
	Pref uint `json:"pref,omitzero"`
}

type OrgUnit struct {
	// The JSContact type of the object: the value MUST be `OrgUnit`, if set.
	Type string `json:"@type,omitempty"`

	// The name of the organizational unit.
	Name string `json:"name"`

	// he value to lexicographically sort the organizational unit in relation to other organizational
	// units of the same level when compared by name.
	//
	// The level is defined by the array index of the organizational unit in the units property
	// of the Organization object.
	//
	// The property value defines the verbatim string value to compare.
	//
	// In absence of this property, the name property value MAY be used for comparison.
	SortAs string `json:"sortAs,omitempty"`
}

type Organization struct {
	// The JSContact type of the object: the value MUST be `Organization`, if set.
	Type string `json:"@type,omitempty"`

	// The name of the organization.
	Name string `json:"name,omitempty"`

	// A list of organizational units, ordered as descending by hierarchy.
	// (e.g., a geographic or functional division sorts before a department within that division).
	//
	// If set, the list MUST contain at least one entry
	Units []OrgUnit `json:"units,omitempty"`

	// The value to lexicographically sort the organization in relation to other organizations when
	// compared by name.
	//
	// The value defines the verbatim string value to compare.
	//
	// In absence of this property, the name property value MAY be used for comparison.
	SortAs string `json:"sortAs,omitempty"`

	// The contexts in which association with the organization applies.
	//
	// For example, membership in a choir may only apply in a private context.
	//
	// TODO document https://www.rfc-editor.org/rfc/rfc9553.html#prop-contexts
	Contexts map[string]bool `json:"contexts,omitempty"`
}

type Pronouns struct {
	// The JSContact type of the object: the value MUST be `Pronouns`, if set.
	Type string `json:"@type,omitempty"`

	// The pronouns.
	//
	// Any value or form is allowed.
	//
	// Examples in English include `she/her` and `they/them/theirs`.
	//
	// The value MAY be overridden in the `localizations` property.
	Pronouns string `json:"pronouns"`

	// The contexts in which to use the pronouns.
	Contexts map[string]bool `json:"contexts,omitempty"`

	// The preference of the pronouns in relation to other pronouns in the same context.
	//
	// A preference order for contact information. For example, a person may have two email addresses and
	// prefer to be contacted with one of them.
	//
	// The value MUST be in the range of 1 to 100. Lower values correspond to a higher level of preference,
	// with 1 being most preferred.
	//
	// If no preference is set, then the contact information MUST be interpreted as being least preferred.
	//
	// Note that the preference is only defined in relation to contact information of the same type.
	//
	// For example, the preference orders within emails and phone numbers are independent of each other.
	Pref uint `json:"pref,omitzero"`
}

type Title struct {
	// The JSContact type of the object: the value MUST be `Title`, if set.
	Type string `json:"@type,omitempty"`

	// The title or role name of the entity represented by the Card.
	Name string `json:"name"`

	// The organizational or situational kind of the title.
	//
	// Some organizations and individuals distinguish between titles as organizational
	// positions and roles as more temporary assignments such as in project management.
	//
	// The enumerated values are:
	// !- `title`
	// !- `role`
	Kind string `json:"kind,omitempty"`

	// The identifier of the organization in which this title is held.
	OrganizationId string `json:"organizationId,omitempty"`
}

type SpeakToAs struct {
	// The JSContact type of the object: the value MUST be `SpeakToAs`, if set.
	Type string `json:"@type,omitempty"`

	// The grammatical gender to use in salutations and other grammatical constructs.
	//
	// For example, the German language distinguishes by grammatical gender in salutations such as
	// `Sehr geehrte` (feminine) and `Sehr geehrter` (masculine).
	//
	// The enumerated values are:
	// !- `animate`
	// !- `common`
	// !- `feminine`
	// !- `inanimate`
	// !- `masculine`
	// !- `neuter`
	//
	// Note that the grammatical gender does not allow inferring the gender identities or assigned
	// sex of the contact.
	GrammaticalGender string `json:"grammaticalGender,omitempty"`

	// The pronouns that the contact chooses to use for themselves.
	Pronouns map[string]Pronouns `json:"pronouns,omitempty"`
}

type Name struct {
	// The JSContact type of the object: the value MUST be `Name`, if set.
	Type string `json:"@type,omitempty"`

	// The components making up this name.
	//
	// The components property MUST be set if the full property is not set; otherwise, it SHOULD be set.
	//
	// The component list MUST have at least one entry having a different kind property value than `separator`.
	//
	// `Name` components SHOULD be ordered such that when their values are joined as a `string`, a valid full name
	// of the entity is produced. If so, implementations MUST set the isOrdered property value to `true`.
	//
	// If the name `components` are ordered, then the `defaultSeparator` property and name components with the kind
	// property value set to `separator` give guidance on what characters to insert between components, but
	// implementations are free to choose any others.
	//
	// When lacking a separator, inserting a single space character in between the name component values is a good choice.
	//
	// If, instead, the name components follow no particular order, then the `isOrdered` property value MUST be
	// `false`, the `components` property MUST NOT contain a `NameComponent` with the `kind` property value set to
	// `separator`, and the `defaultSeparator` property MUST NOT be set.
	Components []NameComponent `json:"components,omitempty"`

	// The indicator if the name components in the components property are ordered.
	//
	// Default: `false`
	IsOrdered bool `json:"isOrdered,omitzero"`

	// The default separator to insert between name component values when concatenating all name component values to a single String.
	//
	// Also see the definition of the kind property value `separator` for the `NameComponent` object.
	//
	// The `defaultSeparator` property MUST NOT be set if the `Name` `isOrdered` property value is `false` or if
	// the components property is not set.
	//
	// example: {"name": {  "components": [{ "kind": "given", "value": "Diego" }, { "kind": "surname", "value": "Rivera" }, { "kind": "surname2", "value": "Barrientos" }], "isOrdered": true}
	DefaultSeparator string `json:"defaultSeparator,omitempty"`

	// The full name representation of the `Name`.
	//
	// The `full` property MUST be set if the components property is not set.
	//
	// example: Mr. John Q. Public, Esq.
	Full string `json:"full,omitempty"`

	// The value to lexicographically sort the name in relation to other names when compared by a name component type.
	//
	// The keys in the map define the name component type. The values define the verbatim string to compare when sorting
	// by the name component type.
	//
	// Absence of a key indicates that the name component type SHOULD NOT be considered during sort.
	//
	// Sorting by that missing name component type, or if the sortAs property is not set, is implementation-specific.
	//
	// The sortAs property MUST NOT be set if the components property is not set.
	//
	// Each key in the map MUST be a valid name component type value as defined for the kind property of the NameComponent
	// object.
	//
	// For each key in the map, there MUST exist at least one NameComponent object that has the type in the components
	// property of the name.
	SortAs map[string]string `json:"sortAs,omitempty"`

	// The script used in the value of the NameComponent phonetic property.
	// TODO https://www.rfc-editor.org/rfc/rfc9553.html#prop-phonetic
	PhoneticScript string `json:"phoneticScript,omitempty"`

	// The phonetic system used in the NameComponent phonetic property.
	// TODO https://www.rfc-editor.org/rfc/rfc9553.html#prop-phonetic
	PhoneticSystem string `json:"phoneticSystem,omitempty"`
}

type EmailAddress struct {
	// The JSContact type of the object: the value MUST be `EmailAddress`, if set.
	Type string `json:"@type,omitempty"`

	// The email address.
	//
	// This MUST be an addr-spec value as defined in [Section 3.4.1 of RFC5322].
	//
	// [Section 3.4.1 of RFC5322]: https://www.rfc-editor.org/rfc/rfc5322.html#section-3.4.1
	Address string `json:"address"`

	// The contexts in which to use this email address.
	Contexts map[string]bool `json:"contexts,omitempty"`

	// The preference of the email address in relation to other email addresses.
	//
	// A preference order for contact information. For example, a person may have two email addresses and
	// prefer to be contacted with one of them.
	//
	// The value MUST be in the range of 1 to 100. Lower values correspond to a higher level of preference,
	// with 1 being most preferred.
	//
	// If no preference is set, then the contact information MUST be interpreted as being least preferred.
	//
	// Note that the preference is only defined in relation to contact information of the same type.
	//
	// For example, the preference orders within emails and phone numbers are independent of each other.
	Pref uint `json:"pref,omitzero"`

	// A custom label for the value.
	//
	// The labels associated with the contact data. Such labels may be set for phone numbers, email addresses, and other resources.
	//
	// Typically, these labels are displayed along with their associated contact data in graphical user interfaces.
	//
	// Note that succinct labels are best for proper display on small graphical interfaces and screens.
	Label string `json:"label,omitempty"`
}

type OnlineService struct {
	// The JSContact type of the object: the value MUST be `OnlineService`, if set.
	Type string `json:"@type,omitempty"`

	// The name of the online service or protocol.
	//
	// The name MAY be capitalized the same as on the service's website, app, or publishing material,
	// but names MUST be considered equal if they match case-insensitively.
	//
	// Examples are `GitHub`, `kakao`, and `Mastodon`.
	Service string `json:"service,omitempty"`

	// The identifier for the entity represented by the Card at the online service.
	Uri string `json:"uri,omitempty"`

	// The name the entity represented by the Card at the online service.
	//
	// Any free-text value is allowed.
	User string `json:"user,omitempty"`

	// The contexts in which to use the service.
	Contexts map[string]bool `json:"contexts,omitempty"`

	// The preference of the service in relation to other services.
	//
	// A preference order for contact information. For example, a person may have two email addresses and
	// prefer to be contacted with one of them.
	//
	// The value MUST be in the range of 1 to 100. Lower values correspond to a higher level of preference,
	// with 1 being most preferred.
	//
	// If no preference is set, then the contact information MUST be interpreted as being least preferred.
	//
	// Note that the preference is only defined in relation to contact information of the same type.
	//
	// For example, the preference orders within emails and phone numbers are independent of each other.
	Pref uint `json:"pref,omitzero"`

	// A custom label for the value.
	//
	// The labels associated with the contact data. Such labels may be set for phone numbers, email addresses, and other resources.
	//
	// Typically, these labels are displayed along with their associated contact data in graphical user interfaces.
	//
	// Note that succinct labels are best for proper display on small graphical interfaces and screens.
	Label string `json:"label,omitempty"`
}

type Phone struct {
	// The JSContact type of the object: the value MUST be `Phone`, if set.
	Type string `json:"@type,omitempty"`

	// The phone number as either a URI or free text.
	//
	// Typical URI schemes are `tel` [RFC3966] or `sip` [RFC3261], but any URI scheme is allowed.
	//
	// [RFC3966]: https://www.rfc-editor.org/rfc/rfc3966.html
	// [RFC3261]: https://www.rfc-editor.org/rfc/rfc3261.html
	Number string `json:"number"`

	// The set of contact features that the phone number may be used for.
	//
	// The set is represented as an object, with each key being a method type.
	//
	// The boolean value MUST be `true`.
	//
	// The enumerated values are:
	// !- `mobile`: this number is for a mobile phone
	// !- `voice`: this number supports calling by voice
	// !- `text`: this number supports text messages (SMS)
	// !- `video`: this number supports video conferencing
	// !- `main-number`: this number is a main phone number such as the number of the front desk at a company, as opposed to a direct-dial number of an individual employee
	// !- `textphone`: this number is for a device for people with hearing or speech difficulties
	// !- `fax`: this number supports sending faxes
	// !- `pager`: this number is for a pager or beeper
	Features map[string]bool `json:"features,omitempty"`

	// The contexts in which to use the number.
	Contexts map[string]bool `json:"contexts,omitempty"`

	// The preference of the number in relation to other numbers.
	//
	// A preference order for contact information. For example, a person may have two email addresses and
	// prefer to be contacted with one of them.
	//
	// The value MUST be in the range of 1 to 100. Lower values correspond to a higher level of preference,
	// with 1 being most preferred.
	//
	// If no preference is set, then the contact information MUST be interpreted as being least preferred.
	//
	// Note that the preference is only defined in relation to contact information of the same type.
	//
	// For example, the preference orders within emails and phone numbers are independent of each other.
	Pref uint `json:"pref,omitzero"`

	// A custom label for the value.
	//
	// The labels associated with the contact data. Such labels may be set for phone numbers, email addresses, and other resources.
	//
	// Typically, these labels are displayed along with their associated contact data in graphical user interfaces.
	//
	// Note that succinct labels are best for proper display on small graphical interfaces and screens.
	Label string `json:"label,omitempty"`
}

type LanguagePref struct {
	// The JSContact type of the object: the value MUST be `LanguagePref`, if set.
	Type string `json:"@type,omitempty"`

	// The preferred language.
	//
	// This MUST be a language tag as defined in [RFC5646].
	//
	// [RFC5646]: https://www.rfc-editor.org/rfc/rfc5646.html
	Language string `json:"language"`

	// The contexts in which to use the language.
	Contexts map[string]bool `json:"contexts,omitempty"`

	// The preference of the language in relation to other languages of the same contexts.
	//
	// A preference order for contact information. For example, a person may have two email addresses and
	// prefer to be contacted with one of them.
	//
	// The value MUST be in the range of 1 to 100. Lower values correspond to a higher level of preference,
	// with 1 being most preferred.
	//
	// If no preference is set, then the contact information MUST be interpreted as being least preferred.
	//
	// Note that the preference is only defined in relation to contact information of the same type.
	//
	// For example, the preference orders within emails and phone numbers are independent of each other.
	Pref uint `json:"pref,omitzero"`
}

type SchedulingAddress struct {
	// The JSContact type of the object: the value MUST be `SchedulingAddress`, if set.
	Type string `json:"@type,omitempty"`

	// The address to use for calendar scheduling with the contact.
	Uri string `json:"uri,omitempty"`

	// The contexts in which to use the scheduling address.
	Contexts map[string]bool `json:"contexts,omitempty"`

	// The preference of the scheduling address in relation to other scheduling addresses.
	//
	// A preference order for contact information. For example, a person may have two email addresses and
	// prefer to be contacted with one of them.
	//
	// The value MUST be in the range of 1 to 100. Lower values correspond to a higher level of preference,
	// with 1 being most preferred.
	//
	// If no preference is set, then the contact information MUST be interpreted as being least preferred.
	//
	// Note that the preference is only defined in relation to contact information of the same type.
	//
	// For example, the preference orders within emails and phone numbers are independent of each other.
	Pref uint `json:"pref,omitzero"`

	// A custom label for the scheduling address.
	//
	// The labels associated with the contact data. Such labels may be set for phone numbers, email addresses, and other resources.
	//
	// Typically, these labels are displayed along with their associated contact data in graphical user interfaces.
	//
	// Note that succinct labels are best for proper display on small graphical interfaces and screens.
	Label string `json:"label,omitempty"`
}

type AddressComponent struct {
	// The JSContact type of the object: the value MUST be `AddressComponent`, if set.
	Type string `json:"@type,omitempty"`

	// The value of the address component.
	Value string `json:"value"`

	// The kind of the address component.
	//
	// The enumerated values are:
	// !- `room`: the room, suite number, or identifier
	// !- `apartment`: the extension designation such as the apartment number, unit, or box number
	// !- `floor`: the floor or level the address is located on
	// !- `building`: the building, tower, or condominium the address is located in
	// !- `number`: the street number, e.g., `"123"`; this value is not restricted to numeric values and can include any value such
	// as number ranges (`"112-10"`), grid style (`"39.2 RD"`), alphanumerics (`"N6W23001"`), or fractionals (`"123 1/2"`)
	// !- `name`: the street name
	// !- `block`: the block name or number
	// !- `subdistrict`: the subdistrict, ward, or other subunit of a district
	// !- `district`: the district name
	// !- `locality`: the municipality, city, town, village, post town, or other locality
	// !- `region`: the administrative area such as province, state, prefecture, county, or canton
	// !- `postcode`: the postal code, post code, ZIP code, or other short code associated with the address by the relevant country's postal system
	// !- `country`: the country name
	// !- `direction`: the cardinal direction or quadrant, e.g., "north"
	// !- `landmark`: the publicly known prominent feature that can substitute the street name and number, e.g., "White House" or "Taj Mahal"
	// !- `postOfficeBox`: the post office box number or identifier
	// !- `separator`: a formatting separator between two ordered address non-separator components; the value property of the component includes the
	// verbatim separator, for example, a hyphen character or even an empty string; this value has higher precedence than the `defaultSeparator` property
	// of the `Address`; implementations MUST NOT insert two consecutive separator components; instead, they SHOULD insert a single separator component
	// with the combined value; this component kind MUST NOT be set if the `Address` `isOrdered` property value is `false`.
	Kind string `json:"kind"`

	// The pronunciation of the name component.
	//
	// If this property is set, then at least one of the Address object `phoneticSystem` or `phoneticScript` properties MUST be set.
	Phonetic string `json:"phonetic,omitempty"`
}

// An Address object has the following properties, of which at least one of components, coordinates, countryCode, full or timeZone MUST be set.
type Address struct {
	// The JSContact type of the object: the value MUST be `Address`, if set.
	Type string `json:"@type,omitempty"`

	// The components that make up the address.
	//
	// The component list MUST have at least one entry that has a kind property value other than `separator`.
	//
	// Address components SHOULD be ordered such that when their values are joined as a String, a valid full address is produced.
	//
	// If so, implementations MUST set the isOrdered property value to `true`.
	//
	// If the address components are ordered, then the `defaultSeparator` property and address components with the `kind`
	// property value set to `separator` give guidance on what characters to insert between components, but implementations
	// are free to choose any others.
	//
	// When lacking a separator, inserting a single space character in between address component values is a good choice.
	//
	// If, instead, the address components follow no particular order, then the isOrdered property value MUST be `false`,
	// the components property MUST NOT contain an `AddressComponent` with the `kind` property value set to `separator`,
	// and the `defaultSeparator` property MUST NOT be set.
	Components []AddressComponent `json:"components,omitempty"`

	// The indicator if the address components in the components property are ordered (default: `false`).
	IsOrdered bool `json:"isOrdered,omitzero"`

	// The Alpha-2 country code as of [ISO.3166-1].
	//
	// [ISO.3166-1]: https://www.iso.org/iso-3166-country-codes.html
	CountryCode string `json:"countryCode,omitempty"`

	// A "geo:" URI [RFC5870] for the address.
	//
	// [RFC5870]: https://www.rfc-editor.org/rfc/rfc5870.html
	Coordinates string `json:"coordinates,omitempty"`

	// The time zone in which the address is located.
	//
	// This MUST be a time zone name registered in the IANA Time Zone Database [IANA-TZ].
	//
	// [IANA-TZ]: https://www.iana.org/time-zones
	TimeZone string `json:"timeZone,omitempty"`

	// The contexts in which to use this address.
	//
	// The boolean value MUST be `true`.
	//
	// In addition to the common contexts, allowed key values are:
	// !- `billing`: an address to be used for billing
	// !- `delivery`: an address to be used for delivering physical items
	Contexts map[string]bool `json:"contexts,omitempty"`

	// The full address, including street, region, or country.
	//
	// The purpose of this property is to define an address, even if the individual address components are not known.
	Full string `json:"full,omitempty"`

	// The default separator to insert between address component values when concatenating all address component values to a single String.
	//
	// Also see the definition of the `kind` property value `separator` for the `AddressComponent` object.
	//
	// The `defaultSeparator` property MUST NOT be set if the Address `isOrdered` property value is `false` or if the `components` property is not set.
	DefaultSeparator string `json:"defaultSeparator,omitempty"`

	// The preference of the address in relation to other addresses.
	//
	// A preference order for contact information. For example, a person may have two email addresses and
	// prefer to be contacted with one of them.
	//
	// The value MUST be in the range of 1 to 100. Lower values correspond to a higher level of preference,
	// with 1 being most preferred.
	//
	// If no preference is set, then the contact information MUST be interpreted as being least preferred.
	//
	// Note that the preference is only defined in relation to contact information of the same type.
	//
	// For example, the preference orders within emails and phone numbers are independent of each other.
	Pref uint `json:"pref,omitzero"`

	// The script used in the value of the Address phonetic property.
	// TODO https://www.rfc-editor.org/rfc/rfc9553.html#prop-phonetic
	PhoneticScript string `json:"phoneticScript,omitempty"`

	// The phonetic system used in the NameComAddressponent phonetic property.
	// TODO https://www.rfc-editor.org/rfc/rfc9553.html#prop-phonetic
	PhoneticSystem string `json:"phoneticSystem,omitempty"`
}

type AnniversaryDate interface {
	isAnniversaryDate() // marker
}

// A PartialDate object represents a complete or partial calendar date in the Gregorian calendar.
//
// It represents a complete date, a year, a month in a year, or a day in a month.
type PartialDate struct {
	// The JSContact type of the object; the value MUST be `PartialDate`, if set.
	Type string `json:"@type,omitempty"`

	// The calendar year.
	Year uint `json:"year,omitzero"`

	// The calendar month, represented as the integers 1 <= month <= 12.
	//
	// If this property is set, then either the `year` or the `day` property MUST be set.
	Month uint `json:"month,omitzero"`

	// The calendar month day, represented as the integers 1 <= day <= 31, depending on the validity
	// within the month and year.
	//
	// If this property is set, then the `month` property MUST be set.
	Day uint `json:"day,omitzero"`

	// The calendar system in which this date occurs, in lowercase.
	//
	// This MUST be either a calendar system name registered as a Common Locale Data Repository [CLDR] [RFC7529]
	// or a vendor-specific value.
	//
	// The year, month, and day still MUST be represented in the Gregorian calendar.
	//
	// Note that the year property might be required to convert the date between the Gregorian calendar
	// and the respective calendar system.
	//
	// [CLDR]: https://github.com/unicode-org/cldr/blob/latest/common/bcp47/calendar.xml
	// [RFC7529]: https://www.rfc-editor.org/rfc/rfc7529.html
	CalendarScale string `json:"calendarScale,omitempty"`
}

func (_ PartialDate) isAnniversaryDate() {
}

var _ AnniversaryDate = &PartialDate{}

type Timestamp struct {
	// The JSContact type of the object; the value MUST be `Timestamp`, if set.
	Type string `json:"@type,omitempty"`

	// The point in time in UTC time (UTCDateTime).
	Utc time.Time `json:"utc"`
}

var _ AnniversaryDate = &Timestamp{}

func (_ Timestamp) isAnniversaryDate() {
}

type Anniversary struct {
	// The JSContact type of the object: the value MUST be `Anniversary`, if set.
	Type string `json:"@type,omitempty"`

	// The kind of anniversary.
	//
	// The enumerated values are:
	// !- `birth`: a birthday anniversary
	// !- `death`: a deathday anniversary
	// !- `wedding`: a wedding day anniversary
	Kind string `json:"kind"`

	// The date of the anniversary in the Gregorian calendar.
	//
	// This MUST be either a whole or partial calendar date or a complete UTC timestamp
	// (see the definition of the `Timestamp` and `PartialDate` object types).
	Date AnniversaryDate `json:"date"`
}

type Author struct {
	// The JSContact type of the object: the value MUST be `Author`, if set.
	Type string `json:"@type,omitempty"`

	// The name of this author.
	Name string `json:"name,omitempty"`

	// The URI value that identifies the author.
	Uri string `json:"uri,omitempty"`
}

type Note struct {
	// The JSContact type of the object: the value MUST be `Note`, if set.
	Type string `json:"@type,omitempty"`

	// The free-text value of this note.
	Note string `json:"note"`

	// The date and time when this note was created.
	Created time.Time `json:"created,omitzero"`

	// The author of this note.
	Author *Author `json:"author,omitempty"`
}

type PersonalInfo struct {
	// The JSContact type of the object: the value MUST be `PersonalInfo`, if set.
	Type string `json:"@type,omitempty"`

	// The kind of personal information.
	//
	// The enumerated values are:
	// !- `expertise`: a field of expertise or a credential
	// !- `hobby`: a hobby
	// !- `interest`: an interest
	Kind string `json:"kind"`

	// The actual information.
	Value string `json:"value"`

	// The level of expertise or engagement in hobby or interest.
	//
	// The enumerated values are:
	// !- `high`
	// !- `medium`
	// !- `low`
	Level string `json:"level,omitempty"`

	// The position of the personal information in the list of all `PersonalInfo` objects that
	// have the same kind property value in the Card.
	//
	// If set, the `listAs` value MUST be higher than zero.
	//
	// Multiple personal information entries MAY have the same `listAs` property value or none.
	//
	// Sorting such same-valued entries is implementation-specific.
	ListAs uint `json:"listAs,omitzero"`

	// A [custom label].
	//
	// The labels associated with the contact data.
	//
	// Such labels may be set for phone numbers, email addresses, and other resources.
	//
	// Typically, these labels are displayed along with their associated contact data in graphical user interfaces.
	//
	// Note that succinct labels are best for proper display on small graphical interfaces and screens.
	//
	// [custom label]: https://www.rfc-editor.org/rfc/rfc9553.html#prop-label
	Label string `json:"label,omitempty"`
}

// A ContactCard object contains information about a person, company, or other entity, or represents a group of such entities.
//
// It is a JSCard (JSContact) object, as defined in [RFC9553], with two additional properties.
//
// A contact card with a `kind` property equal to `group` represents a group of contacts.
// Clients often present these separately from other contact cards.
//
// The `members` property, as defined in RFC XXX, Section XXX, contains a set of UIDs for other contacts that are the members
// of this group.
// Clients should consider the group to contain any `ContactCard` with a matching UID, from any account they have access to with
// support for the `urn:ietf:params:jmap:contacts` capability.
// UIDs that cannot be found SHOULD be ignored but preserved.
//
// For example, suppose a user adds contacts from a shared address book to their private group, then temporarily loses access to
// this address book.
// The UIDs cannot be resolved so the contacts will disappear from the group.
// However, if they are given permission to access the data again the UIDs will be found and the contacts will reappear.
//
// [RFC9553]: https://www.rfc-editor.org/rfc/rfc9553.html
type ContactCard struct {
	// The id of the Card (immutable; server-set).
	//
	// The id uniquely identifies a Card with a particular “uid” within a particular account.
	//
	// This is a JMAP extension and not part of [RFC9553].
	//
	// [RFC9553]: https://www.rfc-editor.org/rfc/rfc9553.html
	Id string `json:"id"`

	// The set of AddressBook ids this Card belongs to.
	//
	// A card MUST belong to at least one AddressBook at all times (until it is destroyed).
	//
	// The set is represented as an object, with each key being an AddressBook id.
	//
	// The value for each key in the object MUST be true.
	//
	// This is a JMAP extension and not part of [RFC9553].
	//
	// [RFC9553]: https://www.rfc-editor.org/rfc/rfc9553.html
	AddressBookIds map[string]bool `json:"addressBookIds"`

	// The JSContact type of the Card object: the value MUST be "Card".
	Type string `json:"@type,omitempty"`

	// The JSContact version of this Card.
	//
	// The value MUST be one of the IANA-registered JSContact Version values for the version property.
	//
	// example: 1.0
	Version string `json:"version"`

	// The date and time when the Card was created (UTCDateTime).
	//
	// example: 2022-09-30T14:35:10Z
	Created time.Time `json:"created,omitzero"`

	// The kind of the entity the Card represents (default: `individual``).
	//
	// Values are:
	// !- `individual``: a single person
	// !- group: a group of people or entities
	// !- org: an organization
	// !- location: a named location
	// !- device: a device such as an appliance, a computer, or a network element
	// !- application: a software application
	//
	// example: individual
	Kind string `json:"kind,omitempty"`

	// The language tag, as defined in [RFC5646].
	//
	// The language tag that best describes the language used for text in the Card, optionally including
	// additional information such as the script.
	//
	// Note that values MAY be localized in the `localizations` property.
	//
	// [RFC5646]: https://www.rfc-editor.org/rfc/rfc5646.html
	//
	// example: de-AT
	Language string `json:"language,omitempty"`

	// The set of Cards that are members of this group Card.
	//
	// Each key in the set is the uid property value of the member, and each boolean value MUST be `true`.
	// If this property is set, then the value of the kind property MUST be `group`.
	//
	// The opposite is not true. A group Card will usually contain the members property to specify the members
	// of the group, but it is not required to.
	//
	// A group Card without the members property can be considered an abstract grouping or one whose members
	// are known empirically (e.g., `IETF Participants`).
	//
	// example: {"kind": "group", "name": {"full": "The Doe family"}, "uid": "urn:uuid:ab4310aa-fa43-11e9-8f0b-362b9e155667", "members": {"urn:uuid:03a0e51f-d1aa-4385-8a53-e29025acd8af": true, "urn:uuid:b8767877-b4a1-4c70-9acc-505d3819e519": true}
	Members map[string]bool `json:"members,omitempty"`

	// The identifier for the product that created the Card.
	//
	// If set, the value MUST be at least one character long.
	//
	// example: ACME Contacts App version 1.23.5
	ProdId string `json:"prodId,omitempty"`

	// The set of Card objects that relate to the Card.
	//
	// The value is a map, where each key is the uid property value of the related Card, and the value
	// defines the relation
	//
	// ```json
	// {
	//   "relatedTo": {
	//     "urn:uuid:f81d4fae-7dec-11d0-a765-00a0c91e6bf6": {
	//       "relation": {"friend": true}
	//     },
	//     "8cacdfb7d1ffdb59@example.com": {
	//       "relation": {}
	//     }
	//   }
	// }
	// ```
	//
	// example: "relatedTo": {"urn:uuid:f81d4fae-7dec-11d0-a765-00a0c91e6bf6": {"relation": {"friend": true}}, "8cacdfb7d1ffdb59@example.com": {"relation": {}}}
	RelatedTo map[string]Relation `json:"relatedTo,omitempty"`

	// An identifier that associates the object as the same across different systems, address books, and views.
	//
	// The value SHOULD be a URN [RFC8141], but for compatibility with [RFC6350], it MAY also be a URI [RFC3986]
	// or free-text value.
	//
	// The value of the URN SHOULD be in the "uuid" namespace [RFC9562].
	//
	// [RFC9562] describes multiple versions of Universally Unique IDentifiers (UUIDs); UUID version 4 is RECOMMENDED.
	//
	// [RFC8141]: https://www.rfc-editor.org/rfc/rfc8141.html
	// [RFC6350]: https://www.rfc-editor.org/rfc/rfc6350.html
	// [RFC9562]: https://www.rfc-editor.org/rfc/rfc9562.html
	//
	// example: urn:uuid:f81d4fae-7dec-11d0-a765-00a0c91e6bf6
	Uid string `json:"uid"`

	// The date and time when the data in the Card was last modified (UTCDateTime).
	//
	// example: 2021-10-31T22:27:10Z
	Updated time.Time `json:"updated,omitzero"`

	// The name of the entity represented by the Card.
	//
	// This can be any type of name, e.g., it can, but need not, be the legal name of a person.
	Name *Name `json:"name,omitempty"`

	// The nicknames of the entity represented by the Card.
	Nicknames map[string]Nickname `json:"nicknames,omitempty"`

	// The company or organization names and units associated with the Card.
	Organizations map[string]Organization `json:"organizations,omitempty"`

	// The information that directs how to address, speak to, or refer to the entity that is represented by the Card.
	SpeakToAs *SpeakToAs `json:"speakToAs,omitempty"`

	// The job titles or functional positions of the entity represented by the Card.
	Titles map[string]Title `json:"titles,omitempty"`

	// The email addresses in which to contact the entity represented by the Card.
	Emails map[string]EmailAddress `json:"emails,omitempty"`

	// The online services that are associated with the entity represented by the Card.
	//
	// This can be messaging services, social media profiles, and other.
	OnlineServices map[string]OnlineService `json:"onlineServices,omitempty"`

	// The phone numbers by which to contact the entity represented by the Card.
	Phones map[string]Phone `json:"phones,omitempty"`

	// The preferred languages for contacting the entity associated with the Card.
	PreferredLanguages map[string]LanguagePref `json:"preferredLanguages,omitempty"`

	// The calendaring resources of the entity represented by the Card, such as to look up free-busy information.
	//
	// A Calendar object has all properties of the Resource data type, with the following additional definitions:
	// !- The `@type` property value MUST be `Calendar`, if set
	// !- The `kind` property is mandatory. Its enumerated values are:
	//   !- `calendar`: The resource is a calendar that contains entries such as calendar events or tasks
	//   !- `freeBusy`: The resource allows for free-busy lookups, for example, to schedule group events
	Calendars map[string]Resource `json:"calendars,omitempty"`

	// The scheduling addresses by which the entity may receive calendar scheduling invitations.
	SchedulingAddresses map[string]SchedulingAddress `json:"schedulingAddresses,omitempty"`

	// The addresses of the entity represented by the Card, such as postal addresses or geographic locations.
	Addresses map[string]Address `json:"addresses,omitempty"`

	// The cryptographic resources such as public keys and certificates associated with the entity represented by the Card.
	//
	// A CryptoKey object has all properties of the `Resource` data type, with the following additional definition:
	// the `@type` property value MUST be `CryptoKey`, if set.
	//
	// The following example shows how to refer to an external cryptographic resource:
	// ```
	// "cryptoKeys": {
	//   "mykey1": {
	//     "uri": "https://www.example.com/keys/jdoe.cer"
	//   }
	// }
	// ```
	CryptoKeys map[string]Resource `json:"cryptoKeys,omitempty"`

	// The directories containing information about the entity represented by the Card.
	//
	// A Directory object has all properties of the `Resource` data type, with the following additional definitions:
	// !- The `@type` property value MUST be `Directory`, if set
	// !- The `kind` property is mandatory; tts enumerated values are:
	//   !- `directory`: the resource is a directory service that the entity represented by the Card is a part of; this
	// typically is an organizational directory that also contains associated entities, e.g., co-workers and management
	// in a company directory
	//   !- `entry`: the resource is a directory entry of the entity represented by the Card; in contrast to the `directory`
	// type, this is the specific URI for the entity within a directory
	Directories map[string]DirectoryResource `json:"directories,omitempty"`

	// The links to resources that do not fit any of the other use-case-specific resource properties.
	//
	// A Link object has all properties of the `Resource` data type, with the following additional definitions:
	// !- The `@type` property value MUST be `Link`, if set
	// !- The `kind` property is optional; tts enumerated values are:
	//   !- `contact``: the resource is a URI by which the entity represented by the Card may be contacted;
	// this includes web forms or other media that require user interaction
	Links map[string]Resource `json:"links,omitempty"`

	// The media resources such as photographs, avatars, or sounds that are associated with the entity represented by the Card.
	//
	// A Media object has all properties of the Resource data type, with the following additional definitions:
	// !- the `@type` property value MUST be `Media`, if set
	// !- the `kind` property is mandatory; its enumerated values are:
	//   !- `photo`: the resource is a photograph or avatar
	//   !- `sound`: the resource is audio media, e.g., to specify the proper pronunciation of the name property contents
	//   !- `logo`: the resource is a graphic image or logo associated with the entity represented by the Card
	Media map[string]MediaResource `json:"media,omitempty"`

	// The property values localized to languages other than the main `language` of the Card.
	//
	// Localizations provide language-specific alternatives for existing property values and SHOULD NOT add new properties.
	//
	// The keys in the localizations property value are language tags [RFC5646]; the values are of type `PatchObject` and
	// localize the Card in that language tag.
	//
	// The paths in the `PatchObject` are relative to the Card that includes the localizations property.
	//
	// A patch MUST NOT target the localizations property.
	//
	// Conceptually, a Card is localized as follows:
	// !- Determine the language tag in which the Card should be localized.
	// !- If the localizations property includes a key for that language, obtain the PatchObject value;
	// if there is no such key, stop.
	// !- Create a copy of the Card, but do not copy the localizations property.
	// !- Apply all patches in the PatchObject to the copy of the Card.
	// !- Optionally, set the language property in the copy of the Card.
	// !- Use the patched copy of the Card as the localized variant of the original Card.
	//
	// A patch in the `PatchObject` may contain any value type.
	//
	// Its value MUST be a valid value according to the definition of the patched property.
	Localizations map[string]PatchObject `json:"localizations,omitempty"`

	// The memorable dates and events for the entity represented by the Card.
	Anniversaries map[string]Anniversary `json:"anniversaries,omitempty"`

	// The set of free-text keywords, also known as tags.
	//
	// Each key in the set is a keyword, and each boolean value MUST be `true`.
	Keywords map[string]bool `json:"keywords,omitempty"`

	// The free-text notes that are associated with the Card.
	Notes map[string]Note `json:"notes,omitempty"`

	// The personal information of the entity represented by the Card.
	PersonalInfo map[string]PersonalInfo `json:"personalInfo,omitempty"`
}
