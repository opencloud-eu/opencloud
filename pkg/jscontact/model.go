// Implementation of the RFC 9553 JSContact data model, with JMAP Contacts additions.
//
// https://www.rfc-editor.org/rfc/rfc9553
//
// https://jmap.io/spec-contacts.html
package jscontact

import (
	"encoding/json"
	"fmt"
	"time"
)

// The kind of the name component.
//
// * `title`: an honorific title or prefix, e.g., `Mr.`, `Ms.`, or `Dr.`
// * `given`: a given name, also known as "first name" or "personal name"
// * `given2`: a name that appears between the given and surname such as a middle name or patronymic name
// * `surname`: a surname, also known as "last name" or "family name"
// * `surname2`: a secondary surname (used in some cultures), also known as "maternal surname"
// * `credential`: a credential, also known as "accreditation qualifier" or "honorific suffix", e.g., `B.A.`, `Esq.`
// * `generation`: a generation marker or qualifier, e.g., `Jr.` or `III`
// * `separator`: a formatting separator between two ordered name non-separator components; the value property of the component includes the verbatim separator, for example, a hyphen character or even an empty string. This value has higher precedence than the defaultSeparator property of the Name. Implementations MUST NOT insert two consecutive separator components; instead, they SHOULD insert a single separator component with the combined value; this component kind MUST NOT be set if the `Name` `isOrdered` property value is `false`
type NameComponentKind string

// The kind of the address component.
//
// The enumerated values are:
// * `room`: the room, suite number, or identifier
// * `apartment`: the extension designation such as the apartment number, unit, or box number
// * `floor`: the floor or level the address is located on
// * `building`: the building, tower, or condominium the address is located in
// * `number`: the street number, e.g., `"123"`; this value is not restricted to numeric values and can include any value such
// as number ranges (`"112-10"`), grid style (`"39.2 RD"`), alphanumerics (`"N6W23001"`), or fractionals (`"123 1/2"`)
// * `name`: the street name
// * `block`: the block name or number
// * `subdistrict`: the subdistrict, ward, or other subunit of a district
// * `district`: the district name
// * `locality`: the municipality, city, town, village, post town, or other locality
// * `region`: the administrative area such as province, state, prefecture, county, or canton
// * `postcode`: the postal code, post code, ZIP code, or other short code associated with the address by the relevant country's postal system
// * `country`: the country name
// * `direction`: the cardinal direction or quadrant, e.g., "north"
// * `landmark`: the publicly known prominent feature that can substitute the street name and number, e.g., "White House" or "Taj Mahal"
// * `postOfficeBox`: the post office box number or identifier
// * `separator`: a formatting separator between two ordered address non-separator components; the value property of the component includes the
// verbatim separator, for example, a hyphen character or even an empty string; this value has higher precedence than the `defaultSeparator` property
// of the `Address`; implementations MUST NOT insert two consecutive separator components; instead, they SHOULD insert a single separator component
// with the combined value; this component kind MUST NOT be set if the `Address` `isOrdered` property value is `false`.
type AddressComponentKind string

// The relationship of the related Card to the Card, defined as a set of relation types.
//
// The keys in the set define the relation type; the values for each key in the set MUST be "true".
//
// The relationship between the two objects is undefined if the set is empty.
//
// The initial list of enumerated relation types matches the IANA-registered TYPE `IANA-vCard“
// parameter values of the vCard RELATED property ([Section 6.6.6 of RFC6350](https://www.rfc-editor.org/rfc/rfc6350.html#section-6.6.6)):
// * `acquaintance`
// * `agent`
// * `child`
// * `co-resident`
// * `co-worker`
// * `colleague`
// * `contact`
// * `crush`
// * `date`
// * `emergency`
// * `friend`
// * `kin`
// * `me`
// * `met`
// * `muse`
// * `neighbor`
// * `parent`
// * `sibling`
// * `spouse`
// * `sweetheart`
type Relationship string

// The enumerated common context values are:
// * `private`: the contact information that may be used in a private context.
// * `work`: the contact information that may be used in a professional context.
type MediaContext string

// The enumerated common context values are:
// * `private`: the contact information that may be used in a private context.
// * `work`: the contact information that may be used in a professional context.
type NicknameContext string

// The contexts in which to use this address.
//
// The boolean value MUST be `true`.
//
// In addition to the common contexts, allowed key values are:
// * `billing`: an address to be used for billing
// * `delivery`: an address to be used for delivering physical items
type AddressContext string

// The enumerated common context values are:
// * `private`: the contact information that may be used in a private context.
// * `work`: the contact information that may be used in a professional context.
type DirectoryContext string

// The enumerated common context values are:
// * `private`: the contact information that may be used in a private context.
// * `work`: the contact information that may be used in a professional context.
type EmailAddressContext string

// The enumerated common context values are:
// * `private`: the contact information that may be used in a private context.
// * `work`: the contact information that may be used in a professional context.
type OnlineServiceContext string

// The enumerated common context values are:
// * `private`: the contact information that may be used in a private context.
// * `work`: the contact information that may be used in a professional context.
type OrganizationContext string

// The enumerated common context values are:
// * `private`: the contact information that may be used in a private context.
// * `work`: the contact information that may be used in a professional context.
type PronounsContext string

// The enumerated common context values are:
// * `private`: the contact information that may be used in a private context.
// * `work`: the contact information that may be used in a professional context.
type PhoneContext string

// The set of contact features that the phone number may be used for.
//
// The set is represented as an object, with each key being a method type.
//
// The boolean value MUST be `true`.
//
// The enumerated values are:
// * `mobile`: this number is for a mobile phone
// * `voice`: this number supports calling by voice
// * `text`: this number supports text messages (SMS)
// * `video`: this number supports video conferencing
// * `main-number`: this number is a main phone number such as the number of the front desk at a company, as opposed to a direct-dial number of an individual employee
// * `textphone`: this number is for a device for people with hearing or speech difficulties
// * `fax`: this number supports sending faxes
// * `pager`: this number is for a pager or beeper
type PhoneFeature string

// The organizational or situational kind of the title.
//
// Some organizations and individuals distinguish between titles as organizational
// positions and roles as more temporary assignments such as in project management.
//
// The enumerated values are:
// * `title`
// * `role`
type TitleKind string

// The grammatical gender to use in salutations and other grammatical constructs.
//
// For example, the German language distinguishes by grammatical gender in salutations such as
// `Sehr geehrte` (feminine) and `Sehr geehrter` (masculine).
//
// The enumerated values are:
// * `animate`
// * `common`
// * `feminine`
// * `inanimate`
// * `masculine`
// * `neuter`
//
// Note that the grammatical gender does not allow inferring the gender identities or assigned
// sex of the contact.
type GrammaticalGenderType string

// The kind of anniversary.
//
// The enumerated values are:
// * `birth`: a birthday anniversary
// * `death`: a deathday anniversary
// * `wedding`: a wedding day anniversary
type AnniversaryKind string

// The enumerated common context values are:
// * `private`: the contact information that may be used in a private context.
// * `work`: the contact information that may be used in a professional context.
type LanguagePrefContext string

// The enumerated common context values are:
// * `private`: the contact information that may be used in a private context.
// * `work`: the contact information that may be used in a professional context.
type SchedulingAddressContext string

// The kind of personal information.
//
// The enumerated values are:
// * `expertise`: a field of expertise or a credential
// * `hobby`: a hobby
// * `interest`: an interest
type PersonalInfoKind string

// The level of expertise or engagement in hobby or interest.
//
// The enumerated values are:
// * `high`
// * `medium`
// * `low`
type PersonalInfoLevel string

// The kind of the entity the Card represents (default: `individual“).
//
// Values are:
// * `individual`: a single person
// * `group`: a group of people or entities
// * `org`: an organization
// * `location`: a named location
// * `device`: a device such as an appliance, a computer, or a network element
// * `application`: a software application
type ContactCardKind string

// The kind of the `Directory` resource.
//
// The allowed values are defined in the property definition that makes use of the Resource type.
//
// Some property definitions may change this property from being optional to mandatory.
//
// A contact card with a `kind` property equal to `group` represents a group of contacts.
//
// Clients often present these separately from other contact cards.
//
// The `members` property, as defined in [RFC 9553, Section 2.1.6](https://www.rfc-editor.org/rfc/rfc9553#members),
// contains a set of UIDs for other contacts that are the members of this group.
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
type DirectoryKind string

// The kind of the `Calendar` resource.
//
// The allowed values are defined in the property definition that makes use of the Resource type.
//
// Some property definitions may change this property from being optional to mandatory.
//
// A contact card with a `kind` property equal to `group` represents a group of contacts.
//
// Clients often present these separately from other contact cards.
//
// The `members` property, as defined in [RFC 9553, Section 2.1.6](https://www.rfc-editor.org/rfc/rfc9553#members),
// contains a set of UIDs for other contacts that are the members of this group.
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
type CalendarKind string

// The kind of the `Link` resource.
//
// The allowed values are defined in the property definition that makes use of the Resource type.
//
// Some property definitions may change this property from being optional to mandatory.
//
// A contact card with a `kind` property equal to `group` represents a group of contacts.
//
// Clients often present these separately from other contact cards.
//
// The `members` property, as defined in [RFC 9553, Section 2.1.6]( https://www.rfc-editor.org/rfc/rfc9553#members),
// contains a set of UIDs for other contacts that are the members of this group.
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
type LinkKind string

// The kind of the `Media` resource.
//
// The allowed values are defined in the property definition that makes use of the Resource type.
//
// Some property definitions may change this property from being optional to mandatory.
//
// A contact card with a `kind` property equal to `group` represents a group of contacts.
//
// Clients often present these separately from other contact cards.
//
// The `members` property, as defined in [RFC 9553, Section 2.1.6](https://www.rfc-editor.org/rfc/rfc9553#members),
// contains a set of UIDs for other contacts that are the members of this group.
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
type MediaKind string

// The contexts in which to use this resource.
//
// The contexts in which to use the contact information.
//
// For example, someone might have distinct phone numbers for `work` and `private` contexts and may set the
// desired context on the respective phone number in the `phones` property.
//
// This section defines common contexts.
//
// Additional contexts may be defined in the properties or data types that make use of this property.
//
// The enumerated common context values are:
// * `private`: the contact information that may be used in a private context.
// * `work`: the contact information that may be used in a professional context.
type CalendarContext string

// The contexts in which to use this resource.
//
// The contexts in which to use the contact information.
//
// For example, someone might have distinct phone numbers for `work` and `private` contexts and may set the
// desired context on the respective phone number in the `phones` property.
//
// This section defines common contexts.
//
// Additional contexts may be defined in the properties or data types that make use of this property.
//
// The enumerated common context values are:
// * `private`: the contact information that may be used in a private context.
// * `work`: the contact information that may be used in a professional context.
type CryptoKeyContext string

// The contexts in which to use this resource.
//
// The contexts in which to use the contact information.
//
// For example, someone might have distinct phone numbers for `work` and `private` contexts and may set the
// desired context on the respective phone number in the `phones` property.
//
// This section defines common contexts.
//
// Additional contexts may be defined in the properties or data types that make use of this property.
//
// The enumerated common context values are:
// * `private`: the contact information that may be used in a private context.
// * `work`: the contact information that may be used in a professional context.
type LinkContext string

// The JSContact version of this Card.
//
// The value MUST be one of the IANA-registered JSContact Version values for the version property.
type JSContactVersion string

type TypeOfAddress string
type TypeOfAddressComponent string
type TypeOfAnniversary string
type TypeOfAuthor string
type TypeOfContactCard string
type TypeOfCalendar string
type TypeOfCryptoKey string
type TypeOfDirectory string
type TypeOfEmailAddress string
type TypeOfLanguagePref string
type TypeOfLink string
type TypeOfMedia string
type TypeOfName string
type TypeOfNameComponent string
type TypeOfNickname string
type TypeOfNote string
type TypeOfOnlineService string
type TypeOfOrganization string
type TypeOfOrgUnit string
type TypeOfPartialDate string
type TypeOfPersonalInfo string
type TypeOfPhone string
type TypeOfPronouns string
type TypeOfRelation string
type TypeOfSchedulingAddress string
type TypeOfSpeakToAs string
type TypeOfTimestamp string
type TypeOfTitle string

const (
	JSContactVersion_1_0 = JSContactVersion("1.0")

	// Types.
	AddressType           = TypeOfAddress("Address")
	AddressComponentType  = TypeOfAddressComponent("AddressComponent")
	AnniversaryType       = TypeOfAnniversary("Anniversary")
	AuthorType            = TypeOfAuthor("Author")
	ContactCardType       = TypeOfContactCard("Card")
	CalendarType          = TypeOfCalendar("Calendar")
	CryptoKeyType         = TypeOfCryptoKey("CryptoKey")
	DirectoryType         = TypeOfDirectory("Directory")
	EmailAddressType      = TypeOfEmailAddress("EmailAddress")
	LanguagePrefType      = TypeOfLanguagePref("LanguagePref")
	LinkType              = TypeOfLink("Link")
	MediaType             = TypeOfMedia("Media")
	NameType              = TypeOfName("Name")
	NameComponentType     = TypeOfNameComponent("NameComponent")
	NicknameType          = TypeOfNickname("Nickname")
	NoteType              = TypeOfNote("Note")
	OnlineServiceType     = TypeOfOnlineService("OnlineService")
	OrganizationType      = TypeOfOrganization("Organization")
	OrgUnitType           = TypeOfOrgUnit("OrgUnit")
	PartialDateType       = TypeOfPartialDate("PartialDate")
	PersonalInfoType      = TypeOfPersonalInfo("PersonalInfo")
	PhoneType             = TypeOfPhone("Phone")
	PronounsType          = TypeOfPronouns("Pronouns")
	RelationType          = TypeOfRelation("Relation")
	SchedulingAddressType = TypeOfSchedulingAddress("SchedulingAddress")
	SpeakToAsType         = TypeOfSpeakToAs("SpeakToAs")
	TimestampType         = TypeOfTimestamp("Timestamp")
	TitleType             = TypeOfTitle("Title")

	// Kinds.

	AddressComponentKindRoom          = AddressComponentKind("room")
	AddressComponentKindApartment     = AddressComponentKind("apartment")
	AddressComponentKindFloor         = AddressComponentKind("floor")
	AddressComponentKindBuilding      = AddressComponentKind("building")
	AddressComponentKindNumber        = AddressComponentKind("number")
	AddressComponentKindName          = AddressComponentKind("name")
	AddressComponentKindBlock         = AddressComponentKind("block")
	AddressComponentKindSubdistrict   = AddressComponentKind("subdistrict")
	AddressComponentKindDistrict      = AddressComponentKind("district")
	AddressComponentKindLocality      = AddressComponentKind("locality")
	AddressComponentKindRegion        = AddressComponentKind("region")
	AddressComponentKindPostcode      = AddressComponentKind("postcode")
	AddressComponentKindCountry       = AddressComponentKind("country")
	AddressComponentKindDirection     = AddressComponentKind("direction")
	AddressComponentKindLandmark      = AddressComponentKind("landmark")
	AddressComponentKindPostOfficeBox = AddressComponentKind("postOfficeBox")
	AddressComponentKindSeparator     = AddressComponentKind("separator")

	AnniversaryKindBirth   = AnniversaryKind("birth")
	AnniversaryKindDeath   = AnniversaryKind("death")
	AnniversaryKindWedding = AnniversaryKind("wedding")

	CalendarKindCalendar = CalendarKind("calendar")
	CalendarKindFreeBusy = CalendarKind("freeBusy")

	ContactCardKindIndividual  = ContactCardKind("individual")
	ContactCardKindGroup       = ContactCardKind("group")
	ContactCardKindOrg         = ContactCardKind("org")
	ContactCardKindLocation    = ContactCardKind("location")
	ContactCardKindDevice      = ContactCardKind("device")
	ContactCardKindApplication = ContactCardKind("application")

	DirectoryKindDirectory = DirectoryKind("directory")
	DirectoryKindEntry     = DirectoryKind("entry")

	LinkKindContact = LinkKind("contact")

	MediaKindPhoto = MediaKind("photo")
	MediaKindSound = MediaKind("sound")
	MediaKindLogo  = MediaKind("logo")

	NameComponentKindTitle      = NameComponentKind("title")
	NameComponentKindGiven      = NameComponentKind("given")
	NameComponentKindGiven2     = NameComponentKind("given2")
	NameComponentKindSurname    = NameComponentKind("surname")
	NameComponentKindSurname2   = NameComponentKind("surname2")
	NameComponentKindCredential = NameComponentKind("credential")
	NameComponentKindGeneration = NameComponentKind("generation")
	NameComponentKindSeparator  = NameComponentKind("separator")

	PersonalInfoKindExpertise = PersonalInfoKind("expertise")
	PersonalInfoKindHobby     = PersonalInfoKind("hobby")
	PersonalInfoKindInterest  = PersonalInfoKind("interest")

	TitleKindTitle = TitleKind("title")
	TitleKindRole  = TitleKind("role")

	// Contexts.

	AddressContextBilling  = AddressContext("billing")
	AddressContextDelivery = AddressContext("delivery")
	AddressContextPrivate  = AddressContext("private")
	AddressContextWork     = AddressContext("work")

	CalendarContextPrivate = CalendarContext("private")
	CalendarContextWork    = CalendarContext("work")

	CryptoKeyContextPrivate = CryptoKeyContext("private")
	CryptoKeyContextWork    = CryptoKeyContext("work")

	DirectoryContextPrivate = DirectoryContext("private")
	DirectoryContextWork    = DirectoryContext("work")

	EmailAddressContextPrivate = EmailAddressContext("private")
	EmailAddressContextWork    = EmailAddressContext("work")

	LanguagePrefContextPrivate = LanguagePrefContext("private")
	LanguagePrefContextWork    = LanguagePrefContext("work")

	LinkContextPrivate = LinkContext("private")
	LinkContextWork    = LinkContext("work")

	MediaContextPrivate = MediaContext("private")
	MediaContextWork    = MediaContext("work")

	NicknameContextPrivate = NicknameContext("private")
	NicknameContextWork    = NicknameContext("work")

	OnlineServiceContextPrivate = OnlineServiceContext("private")
	OnlineServiceContextWork    = OnlineServiceContext("work")

	OrganizationContextPrivate = OrganizationContext("private")
	OrganizationContextWork    = OrganizationContext("work")

	PhoneContextPrivate = PhoneContext("private")
	PhoneContextWork    = PhoneContext("work")

	PronounsContextPrivate = PronounsContext("private")
	PronounsContextWork    = PronounsContext("work")

	SchedulingAddressContextPrivate = SchedulingAddressContext("private")
	SchedulingAddressContextWork    = SchedulingAddressContext("work")

	// Relations.

	RelationAcquaintance = Relationship("acquaintance")
	RelationAgent        = Relationship("agent")
	RelationChild        = Relationship("child")
	RelationCoResident   = Relationship("co-resident")
	RelationCoWorker     = Relationship("co-worker")
	RelationColleague    = Relationship("colleague")
	RelationContact      = Relationship("contact")
	RelationCrush        = Relationship("crush")
	RelationDate         = Relationship("date")
	RelationEmergency    = Relationship("emergency")
	RelationFriend       = Relationship("friend")
	RelationKin          = Relationship("kin")
	RelationMe           = Relationship("me")
	RelationMet          = Relationship("met")
	RelationMuse         = Relationship("muse")
	RelationNeighbor     = Relationship("neighbor")
	RelationParent       = Relationship("parent")
	RelationSibling      = Relationship("sibling")
	RelationSpouse       = Relationship("spouse")
	RelationSweetheart   = Relationship("sweetheart")

	// GrammaticalGenders.

	GrammaticalGenderAnimate   = GrammaticalGenderType("animate")
	GrammaticalGenderCommon    = GrammaticalGenderType("common")
	GrammaticalGenderFeminine  = GrammaticalGenderType("feminine")
	GrammaticalGenderInanimate = GrammaticalGenderType("inanimate")
	GrammaticalGenderMasculine = GrammaticalGenderType("masculine")
	GrammaticalGenderNeuter    = GrammaticalGenderType("neuter")

	// PersonalInfoLevels.

	PersonalInfoLevelHigh   = PersonalInfoLevel("high")
	PersonalInfoLevelMedium = PersonalInfoLevel("medium")
	PersonalInfoLevelLow    = PersonalInfoLevel("low")

	// PhoneFeatures.

	PhoneFeatureMobile     = PhoneFeature("mobile")
	PhoneFeatureVoice      = PhoneFeature("voice")
	PhoneFeatureText       = PhoneFeature("text")
	PhoneFeatureVideo      = PhoneFeature("video")
	PhoneFeatureMainNumber = PhoneFeature("main-number")
	PhoneFeatureTextPhone  = PhoneFeature("textphone")
	PhoneFeatureFax        = PhoneFeature("fax")
	PhoneFeaturePager      = PhoneFeature("pager")

	RscaleIso8601 = "iso8601"
)

var (
	JSContactVersions = []JSContactVersion{
		JSContactVersion_1_0,
	}

	AddressComponentKinds = []AddressComponentKind{
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

	AddressContexts = []AddressContext{
		AddressContextBilling,
		AddressContextDelivery,
		AddressContextPrivate,
		AddressContextWork,
	}

	AnniversaryKinds = []AnniversaryKind{
		AnniversaryKindBirth,
		AnniversaryKindDeath,
		AnniversaryKindWedding,
	}

	CalendarContexts = []CalendarContext{
		CalendarContextPrivate,
		CalendarContextWork,
	}

	CalendarResourceKinds = []CalendarKind{
		CalendarKindCalendar,
		CalendarKindFreeBusy,
	}

	ContactCardKinds = []ContactCardKind{
		ContactCardKindIndividual,
		ContactCardKindGroup,
		ContactCardKindOrg,
		ContactCardKindLocation,
		ContactCardKindDevice,
		ContactCardKindApplication,
	}

	CryptoKeyContexts = []CryptoKeyContext{
		CryptoKeyContextPrivate,
		CryptoKeyContextWork,
	}

	DirectoryContexts = []DirectoryContext{
		DirectoryContextPrivate,
		DirectoryContextWork,
	}

	DirectoryKinds = []DirectoryKind{
		DirectoryKindDirectory,
		DirectoryKindEntry,
	}

	EmailAddressContexts = []EmailAddressContext{
		EmailAddressContextPrivate,
		EmailAddressContextWork,
	}

	GrammaticalGenders = []GrammaticalGenderType{
		GrammaticalGenderAnimate,
		GrammaticalGenderCommon,
		GrammaticalGenderFeminine,
		GrammaticalGenderInanimate,
		GrammaticalGenderMasculine,
		GrammaticalGenderNeuter,
	}

	LanguagePrefContexts = []LanguagePrefContext{
		LanguagePrefContextPrivate,
		LanguagePrefContextWork,
	}

	LinkContexts = []LinkContext{
		LinkContextPrivate,
		LinkContextWork,
	}

	LinkKinds = []LinkKind{
		LinkKindContact,
	}

	MediaContexts = []MediaContext{
		MediaContextPrivate,
		MediaContextWork,
	}

	MediaKinds = []MediaKind{
		MediaKindPhoto,
		MediaKindSound,
		MediaKindLogo,
	}

	NameComponentKinds = []NameComponentKind{
		NameComponentKindTitle,
		NameComponentKindGiven,
		NameComponentKindGiven2,
		NameComponentKindSurname,
		NameComponentKindSurname2,
		NameComponentKindCredential,
		NameComponentKindGeneration,
		NameComponentKindSeparator,
	}

	NicknameContexts = []NicknameContext{
		NicknameContextPrivate,
		NicknameContextWork,
	}

	OnlineServiceContexts = []OnlineServiceContext{
		OnlineServiceContextPrivate,
		OnlineServiceContextWork,
	}

	OrganizationContexts = []OrganizationContext{
		OrganizationContextPrivate,
		OrganizationContextWork,
	}

	PersonalInfoKinds = []PersonalInfoKind{
		PersonalInfoKindExpertise,
		PersonalInfoKindHobby,
		PersonalInfoKindInterest,
	}

	PersonalInfoLevels = []PersonalInfoLevel{
		PersonalInfoLevelHigh,
		PersonalInfoLevelMedium,
		PersonalInfoLevelLow,
	}

	PhoneContexts = []PhoneContext{
		PhoneContextPrivate,
		PhoneContextWork,
	}

	PhoneFeatures = []PhoneFeature{
		PhoneFeatureMobile,
		PhoneFeatureVoice,
		PhoneFeatureText,
		PhoneFeatureVideo,
		PhoneFeatureMainNumber,
		PhoneFeatureTextPhone,
		PhoneFeatureFax,
		PhoneFeaturePager,
	}

	PronounsContexts = []PronounsContext{
		PronounsContextPrivate,
		PronounsContextWork,
	}

	Relations = []Relationship{
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

	SchedulingAddressContexts = []SchedulingAddressContext{
		SchedulingAddressContextPrivate,
		SchedulingAddressContextWork,
	}

	TitleKinds = []TitleKind{
		TitleKindTitle,
		TitleKindRole,
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
// 1. The pointer MAY reference inside an array, but if the last reference token in the pointer is an array index,
// then the patch value MUST NOT be null. The pointer MUST NOT use `"-"` as an array index in any of its reference
// tokens (i.e., you MUST NOT insert/delete from an array, but you MAY replace the contents of its existing members.
// To add or remove members, one needs to replace the complete array value).
// 2. All reference tokens prior to the last (i.e., the value after the final slash) MUST already exist as values
// in the object being patched. If the last reference token is an array index, then a member at this index MUST
// already exist in the referenced array.
// 3. There MUST NOT be two patches in the `PatchObject` where the pointer of
// one is the prefix of the pointer of the other, e.g., `"addresses/1/city"` and `"addresses"`.
// 4. The value for the patch MUST be valid for the property being set (of the correct type and obeying any
// other applicable restrictions), or if null, the property MUST be optional.
//
// The value associated with each pointerdetermines how to apply that patch:
// * If null, remove the property from the patched object. If the key is not present in the parent, this is a no-op.
// * If non-null, set the value given as the value for this property (this may be a replacement or addition to the
// object being patched).
//
// A `PatchObject` does not define its own `@type` property. Instead, the `@type` property in a patch MUST be handled
// as any other patched property value.
//
// Implementations MUST reject a `PatchObject` in its entirety if any of its patches are invalid.
//
// Implementations MUST NOT apply partial patches.
type PatchObject map[string]any

type Calendar struct {
	// The JSContact type of the object.
	//
	// The value MUST be `Calendar`, if set.
	Type TypeOfCalendar `json:"@type,omitempty"`

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
	// The `members` property, as defined in [RFC 9553, Section 2.1.6](https://www.rfc-editor.org/rfc/rfc9553#members),
	// contains a set of UIDs for other contacts that are the members of this group.
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
	Kind CalendarKind `json:"kind,omitempty"`

	// The resource value.
	//
	// This MUST be a URI as defined in [Section 3 of RFC3986](https://www.rfc-editor.org/rfc/rfc3986.html#section-3).
	Uri string `json:"uri,omitempty"`

	// The [RFC2046 media type](https://www.rfc-editor.org/rfc/rfc2046.html) of the resource identified by the uri property value.
	MediaType string `json:"mediaType,omitempty"`

	// The contexts in which to use this resource.
	//
	// The contexts in which to use the contact information.
	//
	// For example, someone might have distinct phone numbers for `work` and `private` contexts and may set the
	// desired context on the respective phone number in the `phones` property.
	//
	// This section defines common contexts.
	//
	// Additional contexts may be defined in the properties or data types that make use of this property.
	//
	// The enumerated common context values are:
	// * `private`: the contact information that may be used in a private context.
	// * `work`: the contact information that may be used in a professional context.
	Contexts map[CalendarContext]bool `json:"contexts,omitempty"`

	// The [preference](https://www.rfc-editor.org/rfc/rfc9553.html#prop-pref) of the resource in relation to other resources.
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
	Pref uint `json:"pref,omitzero"`

	// A [custom label](https://www.rfc-editor.org/rfc/rfc9553.html#prop-label) for the value.
	//
	// The labels associated with the contact data.
	//
	// Such labels may be set for phone numbers, email addresses, and other resources.
	//
	// Typically, these labels are displayed along with their associated contact data in graphical user interfaces.
	//
	// Note that succinct labels are best for proper display on small graphical interfaces and screens.
	Label string `json:"label,omitempty"`
}

type CryptoKey struct {
	// The JSContact type of the object.
	//
	// The value MUST be `CryptoKey`, if set.
	Type TypeOfCryptoKey `json:"@type,omitempty"`

	// The resource value.
	//
	// This MUST be a URI as defined in [Section 3 of RFC3986](https://www.rfc-editor.org/rfc/rfc3986.html#section-3).
	Uri string `json:"uri,omitempty"`

	// The [RFC2046 media type](https://www.rfc-editor.org/rfc/rfc2046.html) of the resource identified by the uri property value.
	MediaType string `json:"mediaType,omitempty"`

	// The contexts in which to use this resource.
	//
	// The contexts in which to use the contact information.
	//
	// For example, someone might have distinct phone numbers for `work` and `private` contexts and may set the
	// desired context on the respective phone number in the `phones` property.
	//
	// This section defines common contexts.
	//
	// Additional contexts may be defined in the properties or data types that make use of this property.
	//
	// The enumerated common context values are:
	// * `private`: the contact information that may be used in a private context.
	// * `work`: the contact information that may be used in a professional context.
	Contexts map[CryptoKeyContext]bool `json:"contexts,omitempty"`

	// The [preference](https://www.rfc-editor.org/rfc/rfc9553.html#prop-pref) of the resource in relation to other resources.
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
	Pref uint `json:"pref,omitzero"`

	// A [custom label](https://www.rfc-editor.org/rfc/rfc9553.html#prop-label) for the value.
	//
	// The labels associated with the contact data.
	//
	// Such labels may be set for phone numbers, email addresses, and other resources.
	//
	// Typically, these labels are displayed along with their associated contact data in graphical user interfaces.
	//
	// Note that succinct labels are best for proper display on small graphical interfaces and screens.
	Label string `json:"label,omitempty"`
}

type Link struct {
	// The JSContact type of the object.
	//
	// The value MUST be `Link`, if set.
	Type TypeOfLink `json:"@type,omitempty"`

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
	// The `members` property, as defined in [RFC 9553, Section 2.1.6](https://www.rfc-editor.org/rfc/rfc9553#members),
	// contains a set of UIDs for other contacts that are the members of this group.
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
	Kind LinkKind `json:"kind,omitempty"`

	// The resource value.
	//
	// This MUST be a URI as defined in [Section 3 of RFC3986](https://www.rfc-editor.org/rfc/rfc3986.html#section-3).
	Uri string `json:"uri,omitempty"`

	// The [RFC2046 media type](https://www.rfc-editor.org/rfc/rfc2046.html) of the resource identified by the uri property value.
	MediaType string `json:"mediaType,omitempty"`

	// The contexts in which to use this resource.
	//
	// The contexts in which to use the contact information.
	//
	// For example, someone might have distinct phone numbers for `work` and `private` contexts and may set the
	// desired context on the respective phone number in the `phones` property.
	//
	// This section defines common contexts.
	//
	// Additional contexts may be defined in the properties or data types that make use of this property.
	//
	// The enumerated common context values are:
	// * `private`: the contact information that may be used in a private context.
	// * `work`: the contact information that may be used in a professional context.
	Contexts map[LinkContext]bool `json:"contexts,omitempty"`

	// The [preference](https://www.rfc-editor.org/rfc/rfc9553.html#prop-pref) of the resource in relation to other resources.
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
	Pref uint `json:"pref,omitzero"`

	// A [custom label](https://www.rfc-editor.org/rfc/rfc9553.html#prop-label) for the value.
	//
	// The labels associated with the contact data.
	//
	// Such labels may be set for phone numbers, email addresses, and other resources.
	//
	// Typically, these labels are displayed along with their associated contact data in graphical user interfaces.
	//
	// Note that succinct labels are best for proper display on small graphical interfaces and screens.
	Label string `json:"label,omitempty"`
}

type Directory struct {
	// The JSContact type of the object.
	//
	// The value MUST be `Directory`, if set.
	Type TypeOfDirectory `json:"@type,omitempty"`

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
	// The `members` property, as defined in [RFC 9553, Section 2.1.6](https://www.rfc-editor.org/rfc/rfc9553#members),
	// contains a set of UIDs for other contacts that are the members of this group.
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
	Kind DirectoryKind `json:"kind,omitempty"`

	// The resource value.
	//
	// This MUST be a URI as defined in Section 3 of [RFC3986-section3](https://www.rfc-editor.org/rfc/rfc3986.html#section-3).
	Uri string `json:"uri,omitempty"`

	// The [RFC2046 media type](https://www.rfc-editor.org/rfc/rfc2046.html) of the resource identified by the uri property value.
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
	// * `private`: the contact information that may be used in a private context.
	// * `work`: the contact information that may be used in a professional context.
	Contexts map[DirectoryContext]bool `json:"contexts,omitempty"`

	// The [preference](https://www.rfc-editor.org/rfc/rfc9553.html#prop-pref) of the resource in relation to other resources.
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
	Pref uint `json:"pref,omitzero"`

	// A [custom label](https://www.rfc-editor.org/rfc/rfc9553.html#prop-label) for the value.
	//
	// The labels associated with the contact data.
	//
	// Such labels may be set for phone numbers, email addresses, and other resources.
	//
	// Typically, these labels are displayed along with their associated contact data in graphical user interfaces.
	//
	// Note that succinct labels are best for proper display on small graphical interfaces and screens.
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

type Media struct {
	// The JSContact type of the object.
	//
	// The value MUST be `Media`, if set.
	Type TypeOfMedia `json:"@type,omitempty"`

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
	// The `members` property, as defined in [RFC 9553, Section 2.1.6](https://www.rfc-editor.org/rfc/rfc9553#members),
	// contains a set of UIDs for other contacts that are the members of this group.
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
	Kind MediaKind `json:"kind,omitempty"`

	// The resource value.
	//
	// This MUST be a URI as defined in Section 3 of [RFC3986-section3](https://www.rfc-editor.org/rfc/rfc3986.html#section-3).
	Uri string `json:"uri,omitempty"`

	// The [RFC2046 media type](https://www.rfc-editor.org/rfc/rfc2046.html) of the resource identified by the uri property value.
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
	// * `private`: the contact information that may be used in a private context.
	// * `work`: the contact information that may be used in a professional context.
	Contexts map[MediaContext]bool `json:"contexts,omitempty"`

	// The [preference](https://www.rfc-editor.org/rfc/rfc9553.html#prop-pref) of the resource in relation to other resources.
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
	Pref uint `json:"pref,omitzero"`

	// A [custom label](https://www.rfc-editor.org/rfc/rfc9553.html#prop-label) for the value.
	//
	// The labels associated with the contact data.
	//
	// Such labels may be set for phone numbers, email addresses, and other resources.
	//
	// Typically, these labels are displayed along with their associated contact data in graphical user interfaces.
	//
	// Note that succinct labels are best for proper display on small graphical interfaces and screens.
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
	// The JSContact type of the object: the value MUST be `Relation`, if set.
	Type TypeOfRelation `json:"@type,omitempty"`

	// The relationship of the related Card to the Card, defined as a set of relation types.
	//
	// The keys in the set define the relation type; the values for each key in the set MUST be "true".
	//
	// The relationship between the two objects is undefined if the set is empty.
	//
	// The initial list of enumerated relation types matches the IANA-registered TYPE `IANA-vCard`
	// parameter values of the vCard RELATED property ([Section 6.6.6 of RFC6350](https://www.rfc-editor.org/rfc/rfc6350.html#section-6.6.6)):
	// * `acquaintance`
	// * `agent`
	// * `child`
	// * `co-resident`
	// * `co-worker`
	// * `colleague`
	// * `contact`
	// * `crush`
	// * `date`
	// * `emergency`
	// * `friend`
	// * `kin`
	// * `me`
	// * `met`
	// * `muse`
	// * `neighbor`
	// * `parent`
	// * `sibling`
	// * `spouse`
	// * `sweetheart`
	Relation map[Relationship]bool `json:"relation,omitempty"`
}

type NameComponent struct {
	// The JSContact type of the object: the value MUST be `NameComponent`, if set.
	Type TypeOfNameComponent `json:"@type,omitempty"`

	// The value of the name component.
	//
	// This can be composed of one or multiple words such as `Poe` or `van Gogh`.
	Value string `json:"value"`

	// The kind of the name component.
	//
	// * `title`: an honorific title or prefix, e.g., `Mr.`, `Ms.`, or `Dr.`
	// * `given`: a given name, also known as "first name" or "personal name"
	// * `given2`: a name that appears between the given and surname such as a middle name or patronymic name
	// * `surname`: a surname, also known as "last name" or "family name"
	// * `surname2`: a secondary surname (used in some cultures), also known as "maternal surname"
	// * `credential`: a credential, also known as "accreditation qualifier" or "honorific suffix", e.g., `B.A.`, `Esq.`
	// * `generation`: a generation marker or qualifier, e.g., `Jr.` or `III`
	// * `separator`: a formatting separator between two ordered name non-separator components; the value property of the component includes the verbatim separator, for example, a hyphen character or even an empty string. This value has higher precedence than the defaultSeparator property of the Name. Implementations MUST NOT insert two consecutive separator components; instead, they SHOULD insert a single separator component with the combined value; this component kind MUST NOT be set if the `Name` `isOrdered` property value is `false`
	Kind NameComponentKind `json:"kind"`

	// The pronunciation of the name component.
	//
	// If this property is set, then at least one of the `Name` object properties, `phoneticSystem` or `phoneticScript`,
	// MUST be set.
	Phonetic string `json:"phonetic,omitempty"`
}

type Nickname struct {
	// The JSContact type of the object: the value MUST be `Nickname`, if set.
	Type TypeOfNickname `json:"@type,omitempty"`

	// The nickname.
	Name string `json:"name"`

	// The contexts in which to use the nickname.
	// TODO document https://www.rfc-editor.org/rfc/rfc9553.html#prop-contexts
	Contexts map[NicknameContext]bool `json:"contexts,omitempty"`

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
	Type TypeOfOrgUnit `json:"@type,omitempty"`

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
	Type TypeOfOrganization `json:"@type,omitempty"`

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
	Contexts map[OrganizationContext]bool `json:"contexts,omitempty"`
}

type Pronouns struct {
	// The JSContact type of the object: the value MUST be `Pronouns`, if set.
	Type TypeOfPronouns `json:"@type,omitempty"`

	// The pronouns.
	//
	// Any value or form is allowed.
	//
	// Examples in English include `she/her` and `they/them/theirs`.
	//
	// The value MAY be overridden in the `localizations` property.
	Pronouns string `json:"pronouns"`

	// The contexts in which to use the pronouns.
	Contexts map[PronounsContext]bool `json:"contexts,omitempty"`

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
	Type TypeOfTitle `json:"@type,omitempty"`

	// The title or role name of the entity represented by the Card.
	Name string `json:"name"`

	// The organizational or situational kind of the title.
	//
	// Some organizations and individuals distinguish between titles as organizational
	// positions and roles as more temporary assignments such as in project management.
	//
	// The enumerated values are:
	// * `title`
	// * `role`
	Kind TitleKind `json:"kind,omitempty"`

	// The identifier of the organization in which this title is held.
	OrganizationId string `json:"organizationId,omitempty"`
}

type SpeakToAs struct {
	// The JSContact type of the object: the value MUST be `SpeakToAs`, if set.
	Type TypeOfSpeakToAs `json:"@type,omitempty"`

	// The grammatical gender to use in salutations and other grammatical constructs.
	//
	// For example, the German language distinguishes by grammatical gender in salutations such as
	// `Sehr geehrte` (feminine) and `Sehr geehrter` (masculine).
	//
	// The enumerated values are:
	// * `animate`
	// * `common`
	// * `feminine`
	// * `inanimate`
	// * `masculine`
	// * `neuter`
	//
	// Note that the grammatical gender does not allow inferring the gender identities or assigned
	// sex of the contact.
	GrammaticalGender GrammaticalGenderType `json:"grammaticalGender,omitempty"`

	// The pronouns that the contact chooses to use for themselves.
	Pronouns map[string]Pronouns `json:"pronouns,omitempty"`
}

type Name struct {
	// The JSContact type of the object: the value MUST be `Name`, if set.
	Type TypeOfName `json:"@type,omitempty"`

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
	IsOrdered bool `json:"isOrdered,omitzero" doc:"opt" default:"false"`

	// The default separator to insert between name component values when concatenating all name component values to a single String.
	//
	// Also see the definition of the kind property value `separator` for the `NameComponent` object.
	//
	// The `defaultSeparator` property MUST NOT be set if the `Name` `isOrdered` property value is `false` or if
	// the components property is not set.
	DefaultSeparator string `json:"defaultSeparator,omitempty"`

	// The full name representation of the `Name`.
	//
	// The `full` property MUST be set if the components property is not set.
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
	Type TypeOfEmailAddress `json:"@type,omitempty"`

	// The email address.
	//
	// This MUST be an addr-spec value as defined in [Section 3.4.1 of RFC5322](https://www.rfc-editor.org/rfc/rfc5322.html#section-3.4.1).
	Address string `json:"address"`

	// The contexts in which to use this email address.
	Contexts map[EmailAddressContext]bool `json:"contexts,omitempty"`

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
	Type TypeOfOnlineService `json:"@type,omitempty"`

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
	Contexts map[OnlineServiceContext]bool `json:"contexts,omitempty"`

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
	Type TypeOfPhone `json:"@type,omitempty"`

	// The phone number as either a URI or free text.
	//
	// Typical URI schemes are `tel` [RFC3966] or `sip` [RFC3261], but any URI scheme is allowed.
	Number string `json:"number"`

	// The set of contact features that the phone number may be used for.
	//
	// The set is represented as an object, with each key being a method type.
	//
	// The boolean value MUST be `true`.
	//
	// The enumerated values are:
	// * `mobile`: this number is for a mobile phone
	// * `voice`: this number supports calling by voice
	// * `text`: this number supports text messages (SMS)
	// * `video`: this number supports video conferencing
	// * `main-number`: this number is a main phone number such as the number of the front desk at a company, as opposed to a direct-dial number of an individual employee
	// * `textphone`: this number is for a device for people with hearing or speech difficulties
	// * `fax`: this number supports sending faxes
	// * `pager`: this number is for a pager or beeper
	Features map[PhoneFeature]bool `json:"features,omitempty"`

	// The contexts in which to use the number.
	Contexts map[PhoneContext]bool `json:"contexts,omitempty"`

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
	Type TypeOfLanguagePref `json:"@type,omitempty"`

	// The preferred language.
	//
	// This MUST be a language tag as defined in [RFC5646].
	Language string `json:"language"`

	// The contexts in which to use the language.
	Contexts map[LanguagePrefContext]bool `json:"contexts,omitempty"`

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
	Type TypeOfSchedulingAddress `json:"@type,omitempty"`

	// The address to use for calendar scheduling with the contact.
	Uri string `json:"uri,omitempty"`

	// The contexts in which to use the scheduling address.
	Contexts map[SchedulingAddressContext]bool `json:"contexts,omitempty"`

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
	Type TypeOfAddressComponent `json:"@type,omitempty"`

	// The value of the address component.
	Value string `json:"value"`

	// The kind of the address component.
	//
	// The enumerated values are:
	// * `room`: the room, suite number, or identifier
	// * `apartment`: the extension designation such as the apartment number, unit, or box number
	// * `floor`: the floor or level the address is located on
	// * `building`: the building, tower, or condominium the address is located in
	// * `number`: the street number, e.g., `"123"`; this value is not restricted to numeric values and can include any value such
	// as number ranges (`"112-10"`), grid style (`"39.2 RD"`), alphanumerics (`"N6W23001"`), or fractionals (`"123 1/2"`)
	// * `name`: the street name
	// * `block`: the block name or number
	// * `subdistrict`: the subdistrict, ward, or other subunit of a district
	// * `district`: the district name
	// * `locality`: the municipality, city, town, village, post town, or other locality
	// * `region`: the administrative area such as province, state, prefecture, county, or canton
	// * `postcode`: the postal code, post code, ZIP code, or other short code associated with the address by the relevant country's postal system
	// * `country`: the country name
	// * `direction`: the cardinal direction or quadrant, e.g., "north"
	// * `landmark`: the publicly known prominent feature that can substitute the street name and number, e.g., "White House" or "Taj Mahal"
	// * `postOfficeBox`: the post office box number or identifier
	// * `separator`: a formatting separator between two ordered address non-separator components; the value property of the component includes the
	// verbatim separator, for example, a hyphen character or even an empty string; this value has higher precedence than the `defaultSeparator` property
	// of the `Address`; implementations MUST NOT insert two consecutive separator components; instead, they SHOULD insert a single separator component
	// with the combined value; this component kind MUST NOT be set if the `Address` `isOrdered` property value is `false`.
	Kind AddressComponentKind `json:"kind"`

	// The pronunciation of the name component.
	//
	// If this property is set, then at least one of the Address object `phoneticSystem` or `phoneticScript` properties MUST be set.
	Phonetic string `json:"phonetic,omitempty"`
}

// An Address object has the following properties, of which at least one of components, coordinates, countryCode, full or timeZone MUST be set.
type Address struct {
	// The JSContact type of the object: the value MUST be `Address`, if set.
	Type TypeOfAddress `json:"@type,omitempty"`

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

	// The indicator if the address components in the components property are ordered
	IsOrdered bool `json:"isOrdered,omitzero" doc:"opt" default:"false"`

	// The Alpha-2 country code as of [ISO.3166-1](https://www.iso.org/iso-3166-country-codes.html).
	CountryCode string `json:"countryCode,omitempty"`

	// A "geo:" URI [RFC5870] for the address.
	Coordinates string `json:"coordinates,omitempty"`

	// The time zone in which the address is located.
	//
	// This MUST be a time zone name registered in the IANA Time Zone Database [IANA-TZ](https://www.iana.org/time-zones).
	TimeZone string `json:"timeZone,omitempty"`

	// The contexts in which to use this address.
	//
	// The boolean value MUST be `true`.
	//
	// In addition to the common contexts, allowed key values are:
	// * `billing`: an address to be used for billing
	// * `delivery`: an address to be used for delivering physical items
	Contexts map[AddressContext]bool `json:"contexts,omitempty"`

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

type AnniversaryDateContainer struct {
	Value AnniversaryDate
}

func (a *Anniversary) UnmarshalJSON(b []byte) error {
	var typ struct {
		Date struct {
			Type string `json:"@type"`
		} `json:"date,omitzero"`
	}
	if err := json.Unmarshal(b, &typ); err != nil {
		return err
	}
	switch typ.Date.Type {
	case string(PartialDateType):
		a.Date = new(PartialDate)
	case string(TimestampType):
		a.Date = new(Timestamp)
	default:
		return fmt.Errorf("unsupported '%T.date' @type: \"%v\"", a, typ.Date.Type)
	}

	type tmp Anniversary
	return json.Unmarshal(b, (*tmp)(a))
}

// A PartialDate object represents a complete or partial calendar date in the Gregorian calendar.
//
// It represents a complete date, a year, a month in a year, or a day in a month.
type PartialDate struct {
	// The JSContact type of the object; the value MUST be `PartialDate`, if set.
	Type TypeOfPartialDate `json:"@type,omitempty"`

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
	// This MUST be either a calendar system name registered as a Common Locale Data Repository
	// [CLDR](https://github.com/unicode-org/cldr/blob/latest/common/bcp47/calendar.xml); [RFC7529]
	// or a vendor-specific value.
	//
	// The year, month, and day still MUST be represented in the Gregorian calendar.
	//
	// Note that the year property might be required to convert the date between the Gregorian calendar
	// and the respective calendar system.
	CalendarScale string `json:"calendarScale,omitempty"`
}

func (_ PartialDate) isAnniversaryDate() {
	// marker interface method, does not need to do anything
}

var _ AnniversaryDate = &PartialDate{}

type Timestamp struct {
	// The JSContact type of the object; the value MUST be `Timestamp`, if set.
	Type TypeOfTimestamp `json:"@type,omitempty"`

	// The point in time in UTC time (UTCDateTime).
	Utc time.Time `json:"utc"`
}

var _ AnniversaryDate = &Timestamp{}

func (_ Timestamp) isAnniversaryDate() {
	// marker interface method, does not need to do anything
}

type Anniversary struct {
	// The JSContact type of the object: the value MUST be `Anniversary`, if set.
	Type TypeOfAnniversary `json:"@type,omitempty"`

	// The kind of anniversary.
	//
	// The enumerated values are:
	// * `birth`: a birthday anniversary
	// * `death`: a deathday anniversary
	// * `wedding`: a wedding day anniversary
	Kind AnniversaryKind `json:"kind"`

	// The date of the anniversary in the Gregorian calendar.
	//
	// This MUST be either a whole or partial calendar date or a complete UTC timestamp
	// (see the definition of the `Timestamp` and `PartialDate` object types).
	Date AnniversaryDate `json:"date"`
}

type Author struct {
	// The JSContact type of the object: the value MUST be `Author`, if set.
	Type TypeOfAuthor `json:"@type,omitempty"`

	// The name of this author.
	Name string `json:"name,omitempty"`

	// The URI value that identifies the author.
	Uri string `json:"uri,omitempty"`
}

type Note struct {
	// The JSContact type of the object: the value MUST be `Note`, if set.
	Type TypeOfNote `json:"@type,omitempty"`

	// The free-text value of this note.
	Note string `json:"note"`

	// The date and time when this note was created.
	Created time.Time `json:"created,omitzero"`

	// The author of this note.
	Author *Author `json:"author,omitempty"`
}

type PersonalInfo struct {
	// The JSContact type of the object: the value MUST be `PersonalInfo`, if set.
	Type TypeOfPersonalInfo `json:"@type,omitempty"`

	// The kind of personal information.
	//
	// The enumerated values are:
	// * `expertise`: a field of expertise or a credential
	// * `hobby`: a hobby
	// * `interest`: an interest
	Kind PersonalInfoKind `json:"kind"`

	// The actual information.
	Value string `json:"value"`

	// The level of expertise or engagement in hobby or interest.
	//
	// The enumerated values are:
	// * `high`
	// * `medium`
	// * `low`
	Level PersonalInfoLevel `json:"level,omitempty"`

	// The position of the personal information in the list of all `PersonalInfo` objects that
	// have the same kind property value in the Card.
	//
	// If set, the `listAs` value MUST be higher than zero.
	//
	// Multiple personal information entries MAY have the same `listAs` property value or none.
	//
	// Sorting such same-valued entries is implementation-specific.
	ListAs uint `json:"listAs,omitzero"`

	// A [custom label](https://www.rfc-editor.org/rfc/rfc9553.html#prop-label).
	//
	// The labels associated with the contact data.
	//
	// Such labels may be set for phone numbers, email addresses, and other resources.
	//
	// Typically, these labels are displayed along with their associated contact data in graphical user interfaces.
	//
	// Note that succinct labels are best for proper display on small graphical interfaces and screens.
	Label string `json:"label,omitempty"`
}
