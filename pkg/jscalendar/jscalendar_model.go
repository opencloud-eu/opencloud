package jscalendar

import (
	"encoding/json"
	"fmt"
	"time"
)

// This is a date-time string with no time zone/offset information.
//
// It is otherwise in the same format as `UTCDateTime`, including fractional seconds.
//
// For example, `2006-01-02T15:04:05` and `2006-01-02T15:04:05.003` are both valid.
type LocalDateTime struct {
	time.Time
}

type TypeOfRelation string
type TypeOfLink string
type TypeOfEvent string
type TypeOfTask string
type TypeOfGroup string
type TypeOfLocation string
type TypeOfVirtualLocation string
type TypeOfRecurrenceRule string
type TypeOfNDay string
type TypeOfParticipant string
type TypeOfAlert string
type TypeOfOffsetTrigger string
type TypeOfAbsoluteTrigger string
type TypeOfTimeZone string
type TypeOfTimeZoneRule string

type Duration string       // TODO
type SignedDuration string // TODO

type Relationship string
type Display string
type Rel string
type LocationTypeOption string
type LocationRelation string
type VirtualLocationFeature string
type Frequency string
type Skip string
type DayOfWeek string
type FreeBusyStatus string
type Privacy string
type ReplyMethod string
type SendToMethod string
type ParticipantKind string
type Role string
type ParticipationStatus string
type ScheduleAgent string
type Progress string
type RelativeTo string
type AlertAction string
type Status string

const (
	EventMediaType = "application/jscalendar+json;type=event"
	TaskMediaType  = "application/jscalendar+json;type=task"
	GroupMediaType = "application/jscalendar+json;type=group"

	DefaultDescriptionContentType = "text/plain"

	RelationType        = TypeOfRelation("Relation")
	LinkType            = TypeOfLink("Link")
	EventType           = TypeOfEvent("Event")
	TaskType            = TypeOfTask("Task")
	GroupType           = TypeOfGroup("Group")
	LocationType        = TypeOfLocation("Location")
	VirtualLocationType = TypeOfVirtualLocation("VirtualLocation")
	RecurrenceRuleType  = TypeOfRecurrenceRule("RecurrenceRule")
	NDayType            = TypeOfNDay("NDay")
	ParticipantType     = TypeOfParticipant("Participant")
	AlertType           = TypeOfAlert("Alert")
	OffsetTriggerType   = TypeOfOffsetTrigger("OffsetTrigger")
	AbsoluteTriggerType = TypeOfAbsoluteTrigger("AbsoluteTrigger")
	TimeZoneType        = TypeOfTimeZone("TimeZone")
	TimeZoneRuleType    = TypeOfTimeZoneRule("TimeZoneRule")

	RelationshipFirst  = Relationship("first")
	RelationshipNext   = Relationship("next")
	RelationshipChild  = Relationship("child")
	RelationshipParent = Relationship("parent")

	DisplayBadge     = Display("badge")
	DisplayGraphic   = Display("graphic")
	DisplayFullsize  = Display("fullsize")
	DisplayThumbnail = Display("thumbnail")

	// curl https://www.iana.org/assignments/link-relations/link-relations.xml | xq -x '//record/value'|sort
	RelAbout                  = Rel("about")
	RelAcl                    = Rel("acl")
	RelAlternate              = Rel("alternate")
	RelAmphtml                = Rel("amphtml")
	RelApiCatalog             = Rel("api-catalog")
	RelAppendix               = Rel("appendix")
	RelAppleTouchIcon         = Rel("apple-touch-icon")
	RelAppleTouchStartupImage = Rel("apple-touch-startup-image")
	RelArchives               = Rel("archives")
	RelAuthor                 = Rel("author")
	RelBlockedBy              = Rel("blocked-by")
	RelBookmark               = Rel("bookmark")
	RelC2paManifest           = Rel("c2pa-manifest")
	RelCanonical              = Rel("canonical")
	RelChapter                = Rel("chapter")
	RelCiteAs                 = Rel("cite-as")
	RelCollection             = Rel("collection")
	RelCompressionDictionary  = Rel("compression-dictionary")
	RelContents               = Rel("contents")
	RelConvertedfrom          = Rel("convertedfrom")
	RelCopyright              = Rel("copyright")
	RelCreateForm             = Rel("create-form")
	RelCurrent                = Rel("current")
	RelDeprecation            = Rel("deprecation")
	RelDescribedby            = Rel("describedby")
	RelDescribes              = Rel("describes")
	RelDisclosure             = Rel("disclosure")
	RelDnsPrefetch            = Rel("dns-prefetch")
	RelDuplicate              = Rel("duplicate")
	RelEdit                   = Rel("edit")
	RelEditForm               = Rel("edit-form")
	RelEditMedia              = Rel("edit-media")
	RelEnclosure              = Rel("enclosure")
	RelExternal               = Rel("external")
	RelFirst                  = Rel("first")
	RelGeofeed                = Rel("geofeed")
	RelGlossary               = Rel("glossary")
	RelHelp                   = Rel("help")
	RelHosts                  = Rel("hosts")
	RelHub                    = Rel("hub")
	RelIceServer              = Rel("ice-server")
	RelIcon                   = Rel("icon")
	RelIndex                  = Rel("index")
	RelIntervalafter          = Rel("intervalafter")
	RelIntervalbefore         = Rel("intervalbefore")
	RelIntervalcontains       = Rel("intervalcontains")
	RelIntervaldisjoint       = Rel("intervaldisjoint")
	RelIntervalduring         = Rel("intervalduring")
	RelIntervalequals         = Rel("intervalequals")
	RelIntervalfinishedby     = Rel("intervalfinishedby")
	RelIntervalfinishes       = Rel("intervalfinishes")
	RelIntervalin             = Rel("intervalin")
	RelIntervalmeets          = Rel("intervalmeets")
	RelIntervalmetby          = Rel("intervalmetby")
	RelIntervaloverlappedby   = Rel("intervaloverlappedby")
	RelIntervaloverlaps       = Rel("intervaloverlaps")
	RelIntervalstartedby      = Rel("intervalstartedby")
	RelIntervalstarts         = Rel("intervalstarts")
	RelItem                   = Rel("item")
	RelLast                   = Rel("last")
	RelLatestVersion          = Rel("latest-version")
	RelLicense                = Rel("license")
	RelLinkset                = Rel("linkset")
	RelLrdd                   = Rel("lrdd")
	RelManifest               = Rel("manifest")
	RelMaskIcon               = Rel("mask-icon")
	RelMe                     = Rel("me")
	RelMediaFeed              = Rel("media-feed")
	RelMemento                = Rel("memento")
	RelMicropub               = Rel("micropub")
	RelModulepreload          = Rel("modulepreload")
	RelMonitor                = Rel("monitor")
	RelMonitorGroup           = Rel("monitor-group")
	RelNext                   = Rel("next")
	RelNextArchive            = Rel("next-archive")
	RelNofollow               = Rel("nofollow")
	RelNoopener               = Rel("noopener")
	RelNoreferrer             = Rel("noreferrer")
	RelOpener                 = Rel("opener")
	RelOpenid2LocalId         = Rel("openid2.local_id")
	RelOpenid2Provider        = Rel("openid2.provider")
	RelOriginal               = Rel("original")
	RelP3pv1                  = Rel("p3pv1")
	RelPayment                = Rel("payment")
	RelPingback               = Rel("pingback")
	RelPreconnect             = Rel("preconnect")
	RelPredecessorVersion     = Rel("predecessor-version")
	RelPrefetch               = Rel("prefetch")
	RelPreload                = Rel("preload")
	RelPrerender              = Rel("prerender")
	RelPrev                   = Rel("prev")
	RelPrevArchive            = Rel("prev-archive")
	RelPreview                = Rel("preview")
	RelPrevious               = Rel("previous")
	RelPrivacyPolicy          = Rel("privacy-policy")
	RelProfile                = Rel("profile")
	RelPublication            = Rel("publication")
	RelRdapActive             = Rel("rdap-active")
	RelRdapBottom             = Rel("rdap-bottom")
	RelRdapDown               = Rel("rdap-down")
	RelRdapTop                = Rel("rdap-top")
	RelRdapUp                 = Rel("rdap-up")
	RelRelated                = Rel("related")
	RelReplies                = Rel("replies")
	RelRestconf               = Rel("restconf")
	RelRuleinput              = Rel("ruleinput")
	RelSearch                 = Rel("search")
	RelSection                = Rel("section")
	RelSelf                   = Rel("self")
	RelService                = Rel("service")
	RelServiceDesc            = Rel("service-desc")
	RelServiceDoc             = Rel("service-doc")
	RelServiceMeta            = Rel("service-meta")
	RelSipTrunkingCapability  = Rel("sip-trunking-capability")
	RelSponsored              = Rel("sponsored")
	RelStart                  = Rel("start")
	RelStatus                 = Rel("status")
	RelStylesheet             = Rel("stylesheet")
	RelSubsection             = Rel("subsection")
	RelSuccessorVersion       = Rel("successor-version")
	RelSunset                 = Rel("sunset")
	RelTag                    = Rel("tag")
	RelTermsOfService         = Rel("terms-of-service")
	RelTimegate               = Rel("timegate")
	RelTimemap                = Rel("timemap")
	RelType                   = Rel("type")
	RelUgc                    = Rel("ugc")
	RelUp                     = Rel("up")
	RelVersionHistory         = Rel("version-history")
	RelVia                    = Rel("via")
	RelWebmention             = Rel("webmention")
	RelWorkingCopy            = Rel("working-copy")
	RelWorkingCopyOf          = Rel("working-copy-of")

	LocationTypeOptionAircraft              = LocationTypeOption("aircraft")
	LocationTypeOptionAirport               = LocationTypeOption("airport")
	LocationTypeOptionArena                 = LocationTypeOption("arena")
	LocationTypeOptionAutomobile            = LocationTypeOption("automobile")
	LocationTypeOptionBank                  = LocationTypeOption("bank")
	LocationTypeOptionBar                   = LocationTypeOption("bar")
	LocationTypeOptionBicycle               = LocationTypeOption("bicycle")
	LocationTypeOptionBus                   = LocationTypeOption("bus")
	LocationTypeOptionBusStation            = LocationTypeOption("bus-station")
	LocationTypeOptionCafe                  = LocationTypeOption("cafe")
	LocationTypeOptionCampground            = LocationTypeOption("campground")
	LocationTypeOptionCareFacility          = LocationTypeOption("care-facility")
	LocationTypeOptionClassroom             = LocationTypeOption("classroom")
	LocationTypeOptionClub                  = LocationTypeOption("club")
	LocationTypeOptionConstruction          = LocationTypeOption("construction")
	LocationTypeOptionConventionCenter      = LocationTypeOption("convention-center")
	LocationTypeOptionDetachedUnit          = LocationTypeOption("detached-unit")
	LocationTypeOptionFireStation           = LocationTypeOption("fire-station")
	LocationTypeOptionGovernment            = LocationTypeOption("government")
	LocationTypeOptionHospital              = LocationTypeOption("hospital")
	LocationTypeOptionHotel                 = LocationTypeOption("hotel")
	LocationTypeOptionIndustrial            = LocationTypeOption("industrial")
	LocationTypeOptionLandmarkAddress       = LocationTypeOption("landmark-address")
	LocationTypeOptionLibrary               = LocationTypeOption("library")
	LocationTypeOptionMotorcycle            = LocationTypeOption("motorcycle")
	LocationTypeOptionMunicipalGarage       = LocationTypeOption("municipal-garage")
	LocationTypeOptionMuseum                = LocationTypeOption("museum")
	LocationTypeOptionOffice                = LocationTypeOption("office")
	LocationTypeOptionOther                 = LocationTypeOption("other")
	LocationTypeOptionOutdoors              = LocationTypeOption("outdoors")
	LocationTypeOptionParking               = LocationTypeOption("parking")
	LocationTypeOptionPhoneBox              = LocationTypeOption("phone-box")
	LocationTypeOptionPlaceOfWorship        = LocationTypeOption("place-of-worship")
	LocationTypeOptionPostOffice            = LocationTypeOption("post-office")
	LocationTypeOptionPrison                = LocationTypeOption("prison")
	LocationTypeOptionPublic                = LocationTypeOption("public")
	LocationTypeOptionPublicTransport       = LocationTypeOption("public-transport")
	LocationTypeOptionResidence             = LocationTypeOption("residence")
	LocationTypeOptionRestaurant            = LocationTypeOption("restaurant")
	LocationTypeOptionSchool                = LocationTypeOption("school")
	LocationTypeOptionShoppingArea          = LocationTypeOption("shopping-area")
	LocationTypeOptionStadium               = LocationTypeOption("stadium")
	LocationTypeOptionStore                 = LocationTypeOption("store")
	LocationTypeOptionStreet                = LocationTypeOption("street")
	LocationTypeOptionTheater               = LocationTypeOption("theater")
	LocationTypeOptionTollBooth             = LocationTypeOption("toll-booth")
	LocationTypeOptionTownHall              = LocationTypeOption("town-hall")
	LocationTypeOptionTrain                 = LocationTypeOption("train")
	LocationTypeOptionTrainStation          = LocationTypeOption("train-station")
	LocationTypeOptionTruck                 = LocationTypeOption("truck")
	LocationTypeOptionUnderway              = LocationTypeOption("underway")
	LocationTypeOptionUnknown               = LocationTypeOption("unknown")
	LocationTypeOptionUtilitybox            = LocationTypeOption("utilitybox")
	LocationTypeOptionWarehouse             = LocationTypeOption("warehouse")
	LocationTypeOptionWasteTransferFacility = LocationTypeOption("waste-transfer-facility")
	LocationTypeOptionWater                 = LocationTypeOption("water")
	LocationTypeOptionWatercraft            = LocationTypeOption("watercraft")
	LocationTypeOptionWaterFacility         = LocationTypeOption("water-facility")
	LocationTypeOptionYouthCamp             = LocationTypeOption("youth-camp")

	LocationRelationStart = LocationRelation("start")
	LocationRelationEnd   = LocationRelation("end")

	VirtualLocationFeatureAudio     = VirtualLocationFeature("audio")
	VirtualLocationFeatureChat      = VirtualLocationFeature("chat")
	VirtualLocationFeatureFeed      = VirtualLocationFeature("feed")
	VirtualLocationFeatureModerator = VirtualLocationFeature("moderator")
	VirtualLocationFeaturePhone     = VirtualLocationFeature("phone")
	VirtualLocationFeatureScreen    = VirtualLocationFeature("screen")
	VirtualLocationFeatureVideo     = VirtualLocationFeature("video")

	FrequencyYearly   = Frequency("yearly")
	FrequencyMonthly  = Frequency("monthly")
	FrequencyWeekly   = Frequency("weekly")
	FrequencyDaily    = Frequency("daily")
	FrequencyHourly   = Frequency("hourly")
	FrequencyMinutely = Frequency("minutely")
	FrequencySecondly = Frequency("secondly")

	SkipOmit     = Skip("omit")
	SkipBackward = Skip("backward")
	SkipForward  = Skip("forward")

	DayOfWeekMonday    = DayOfWeek("mo")
	DayOfWeekTuesday   = DayOfWeek("tu")
	DayOfWeekWednesday = DayOfWeek("we")
	DayOfWeekThursday  = DayOfWeek("th")
	DayOfWeekFriday    = DayOfWeek("fr")
	DayOfWeekSaturday  = DayOfWeek("sa")
	DayOfWeekSunday    = DayOfWeek("su")

	RscaleIso8601 = "iso8601"

	FreeBusyStatusFree = FreeBusyStatus("free")
	FreeBusyStatusBusy = FreeBusyStatus("busy")

	PrivacyPublic  = Privacy("public")
	PrivacyPrivate = Privacy("private")
	PrivacySecret  = Privacy("secret")

	ReplyMethodImip  = ReplyMethod("imip")
	ReplyMethodWeb   = ReplyMethod("web")
	ReplyMethodOther = ReplyMethod("other")

	SendToMethodImip  = SendToMethod("imip")
	SendToMethodOther = SendToMethod("other")

	ParticipantKindIndividual = ParticipantKind("individual")
	ParticipantKindGroup      = ParticipantKind("group")
	ParticipantKindLocation   = ParticipantKind("location")
	ParticipantKindResource   = ParticipantKind("resource")

	RoleOwner         = Role("owner")
	RoleAttendee      = Role("attendee")
	RoleOptional      = Role("optional")
	RoleInformational = Role("informational")
	RoleChair         = Role("chair")
	RoleContact       = Role("contact")

	ParticipationStatusNeedsAction = ParticipationStatus("needs-action")
	ParticipationStatusAccepted    = ParticipationStatus("accepted")
	ParticipationStatusDeclined    = ParticipationStatus("declined")
	ParticipationStatusTentative   = ParticipationStatus("tentative")
	ParticipationStatusDelegated   = ParticipationStatus("delegated")

	ScheduleAgentServer = ScheduleAgent("server")
	ScheduleAgentClient = ScheduleAgent("client")
	ScheduleAgentNone   = ScheduleAgent("none")

	DefaultScheduleAgent = ScheduleAgentServer

	ProgressNeedsAction = Progress("needs-action")
	ProgressInProcess   = Progress("in-process")
	ProgressCompleted   = Progress("completed")
	ProgressFailed      = Progress("failed")
	ProgressCancelled   = Progress("cancelled")

	RelativeToStart = RelativeTo("start")
	RelativeToEnd   = RelativeTo("end")

	AlertActionDisplay = AlertAction("display")
	AlertActionEmail   = AlertAction("email")

	DefaultAlertAction = AlertActionDisplay

	StatusConfirmed = Status("confirmed")
	StatusCancelled = Status("cancelled")
	StatusTentative = Status("tentative")
)

var (
	Relationships = []Relationship{
		RelationshipFirst,
		RelationshipNext,
		RelationshipChild,
		RelationshipParent,
	}

	Displays = []Display{
		DisplayBadge,
		DisplayGraphic,
		DisplayFullsize,
		DisplayThumbnail,
	}

	Rels = []Rel{
		RelAbout,
		RelAcl,
		RelAlternate,
		RelAmphtml,
		RelApiCatalog,
		RelAppendix,
		RelAppleTouchIcon,
		RelAppleTouchStartupImage,
		RelArchives,
		RelAuthor,
		RelBlockedBy,
		RelBookmark,
		RelC2paManifest,
		RelCanonical,
		RelChapter,
		RelCiteAs,
		RelCollection,
		RelCompressionDictionary,
		RelContents,
		RelConvertedfrom,
		RelCopyright,
		RelCreateForm,
		RelCurrent,
		RelDeprecation,
		RelDescribedby,
		RelDescribes,
		RelDisclosure,
		RelDnsPrefetch,
		RelDuplicate,
		RelEdit,
		RelEditForm,
		RelEditMedia,
		RelEnclosure,
		RelExternal,
		RelFirst,
		RelGeofeed,
		RelGlossary,
		RelHelp,
		RelHosts,
		RelHub,
		RelIceServer,
		RelIcon,
		RelIndex,
		RelIntervalafter,
		RelIntervalbefore,
		RelIntervalcontains,
		RelIntervaldisjoint,
		RelIntervalduring,
		RelIntervalequals,
		RelIntervalfinishedby,
		RelIntervalfinishes,
		RelIntervalin,
		RelIntervalmeets,
		RelIntervalmetby,
		RelIntervaloverlappedby,
		RelIntervaloverlaps,
		RelIntervalstartedby,
		RelIntervalstarts,
		RelItem,
		RelLast,
		RelLatestVersion,
		RelLicense,
		RelLinkset,
		RelLrdd,
		RelManifest,
		RelMaskIcon,
		RelMe,
		RelMediaFeed,
		RelMemento,
		RelMicropub,
		RelModulepreload,
		RelMonitor,
		RelMonitorGroup,
		RelNext,
		RelNextArchive,
		RelNofollow,
		RelNoopener,
		RelNoreferrer,
		RelOpener,
		RelOpenid2LocalId,
		RelOpenid2Provider,
		RelOriginal,
		RelP3pv1,
		RelPayment,
		RelPingback,
		RelPreconnect,
		RelPredecessorVersion,
		RelPrefetch,
		RelPreload,
		RelPrerender,
		RelPrev,
		RelPrevArchive,
		RelPreview,
		RelPrevious,
		RelPrivacyPolicy,
		RelProfile,
		RelPublication,
		RelRdapActive,
		RelRdapBottom,
		RelRdapDown,
		RelRdapTop,
		RelRdapUp,
		RelRelated,
		RelReplies,
		RelRestconf,
		RelRuleinput,
		RelSearch,
		RelSection,
		RelSelf,
		RelService,
		RelServiceDesc,
		RelServiceDoc,
		RelServiceMeta,
		RelSipTrunkingCapability,
		RelSponsored,
		RelStart,
		RelStatus,
		RelStylesheet,
		RelSubsection,
		RelSuccessorVersion,
		RelSunset,
		RelTag,
		RelTermsOfService,
		RelTimegate,
		RelTimemap,
		RelType,
		RelUgc,
		RelUp,
		RelVersionHistory,
		RelVia,
		RelWebmention,
		RelWorkingCopy,
		RelWorkingCopyOf,
	}

	LocationTypeOptions = []LocationTypeOption{
		LocationTypeOptionAircraft,
		LocationTypeOptionAirport,
		LocationTypeOptionArena,
		LocationTypeOptionAutomobile,
		LocationTypeOptionBank,
		LocationTypeOptionBar,
		LocationTypeOptionBicycle,
		LocationTypeOptionBus,
		LocationTypeOptionBusStation,
		LocationTypeOptionCafe,
		LocationTypeOptionCampground,
		LocationTypeOptionCareFacility,
		LocationTypeOptionClassroom,
		LocationTypeOptionClub,
		LocationTypeOptionConstruction,
		LocationTypeOptionConventionCenter,
		LocationTypeOptionDetachedUnit,
		LocationTypeOptionFireStation,
		LocationTypeOptionGovernment,
		LocationTypeOptionHospital,
		LocationTypeOptionHotel,
		LocationTypeOptionIndustrial,
		LocationTypeOptionLandmarkAddress,
		LocationTypeOptionLibrary,
		LocationTypeOptionMotorcycle,
		LocationTypeOptionMunicipalGarage,
		LocationTypeOptionMuseum,
		LocationTypeOptionOffice,
		LocationTypeOptionOther,
		LocationTypeOptionOutdoors,
		LocationTypeOptionParking,
		LocationTypeOptionPhoneBox,
		LocationTypeOptionPlaceOfWorship,
		LocationTypeOptionPostOffice,
		LocationTypeOptionPrison,
		LocationTypeOptionPublic,
		LocationTypeOptionPublicTransport,
		LocationTypeOptionResidence,
		LocationTypeOptionRestaurant,
		LocationTypeOptionSchool,
		LocationTypeOptionShoppingArea,
		LocationTypeOptionStadium,
		LocationTypeOptionStore,
		LocationTypeOptionStreet,
		LocationTypeOptionTheater,
		LocationTypeOptionTollBooth,
		LocationTypeOptionTownHall,
		LocationTypeOptionTrain,
		LocationTypeOptionTrainStation,
		LocationTypeOptionTruck,
		LocationTypeOptionUnderway,
		LocationTypeOptionUnknown,
		LocationTypeOptionUtilitybox,
		LocationTypeOptionWarehouse,
		LocationTypeOptionWasteTransferFacility,
		LocationTypeOptionWater,
		LocationTypeOptionWatercraft,
		LocationTypeOptionWaterFacility,
		LocationTypeOptionYouthCamp,
	}

	LocationRelations = []LocationRelation{
		LocationRelationStart,
		LocationRelationEnd,
	}

	Frequencies = []Frequency{
		FrequencyYearly,
		FrequencyMonthly,
		FrequencyWeekly,
		FrequencyDaily,
		FrequencyHourly,
		FrequencyMinutely,
		FrequencySecondly,
	}

	Skips = []Skip{
		SkipOmit,
		SkipBackward,
		SkipForward,
	}

	RecurrentOverridesIgnoredPrefixes = []string{
		"@type",
		"excludedRecurrenceRules",
		"method",
		"privacy",
		"prodId",
		"recurrenceId",
		"recurrenceIdTimeZone",
		"recurrenceOverrides",
		"recurrenceRules",
		"relatedTo",
		"replyTo",
		"sentBy",
		"timeZones",
		"uid",
	}

	LocalizationRequiredSuffixes = []string{
		"title",
		"description",
		"name",
	}

	FreeBusyStatuses = []FreeBusyStatus{
		FreeBusyStatusFree,
		FreeBusyStatusBusy,
	}

	Privacies = []Privacy{
		PrivacyPublic,
		PrivacyPrivate,
		PrivacySecret,
	}

	ReplyMethods = []ReplyMethod{
		ReplyMethodImip,
		ReplyMethodWeb,
		ReplyMethodOther,
	}

	SendToMethods = []SendToMethod{
		SendToMethodImip,
		SendToMethodOther,
	}

	ParticipantKinds = []ParticipantKind{
		ParticipantKindIndividual,
		ParticipantKindGroup,
		ParticipantKindLocation,
		ParticipantKindResource,
	}

	Roles = []Role{
		RoleOwner,
		RoleAttendee,
		RoleOptional,
		RoleInformational,
		RoleChair,
		RoleContact,
	}

	ParticipationStatuses = []ParticipationStatus{
		ParticipationStatusNeedsAction,
		ParticipationStatusAccepted,
		ParticipationStatusDeclined,
		ParticipationStatusTentative,
		ParticipationStatusDelegated,
	}

	ScheduleAgents = []ScheduleAgent{
		ScheduleAgentServer,
		ScheduleAgentClient,
		ScheduleAgentNone,
	}

	Progresses = []Progress{
		ProgressNeedsAction,
		ProgressInProcess,
		ProgressCompleted,
		ProgressFailed,
		ProgressCancelled,
	}

	RelativeTos = []RelativeTo{
		RelativeToStart,
		RelativeToEnd,
	}

	AlertActions = []AlertAction{
		AlertActionDisplay,
		AlertActionEmail,
	}

	Statuses = []Status{
		StatusConfirmed,
		StatusCancelled,
		StatusTentative,
	}
)

const RFC3339Local = "2006-01-02T15:04:05"

func (t LocalDateTime) MarshalJSON() ([]byte, error) {
	return []byte("\"" + t.UTC().Format(RFC3339Local) + "\""), nil
}

func (t *LocalDateTime) UnmarshalJSON(b []byte) error {
	var tt time.Time
	err := tt.UnmarshalJSON(b)
	if err != nil {
		return err
	}
	t.Time = tt.UTC()
	return nil
}

// A `PatchObject` is of type `String[*]` and represents an unordered set of patches on a JSON object.
//
// Each key is a path represented in a subset of the JSON Pointer format [RFC6901].
//
// The paths have an implicit leading `/`, so each key is prefixed with `/` before applying the
// JSON Pointer evaluation algorithm.
//
// A patch within a `PatchObject` is only valid if all of the following conditions apply:
// !1. The pointer MUST NOT reference inside an array (i.e., you MUST NOT insert/delete
// from an array; the array MUST be replaced in its entirety instead).
// !2. All parts prior to the last (i.e., the value after the final slash) MUST already
// exist on the object being patched.
// !3. There MUST NOT be two patches in the `PatchObject` where the pointer of one is
// the prefix of the pointer of the other, e.g., `alerts/1/offset` and `alerts`.
// !4. The value for the patch MUST be valid for the property being set (of the correct
// type and obeying any other applicable restrictions), or, if null, the property
// MUST be optional.
//
// The value associated with each pointer determines how to apply that patch:
// !- If null, remove the property from the patched object. If the key is not present in the parent,
// this a no-op.
// !- If non-null, set the value given as the value for this property (this may be a replacement
// or addition to the object being patched).
//
// A `PatchObject` does not define its own `@type` property.
// An `@type` property in a patch MUST be handled as any other patched property value.Implementations
// MUST reject a `PatchObject` in its entirety if any of its patches are invalid.
// Implementations MUST NOT apply partial patches.
//
// The `PatchObject` format is used to significantly reduce file size and duplicated content when
// specifying variations to a common object, such as with recurring events or when translating the
// data into multiple languages.
//
// It can also better preserve semantic intent if only the properties that should differ between
// the two objects are patched. For example, if one person is not going to a particular instance
// of a regularly scheduled event, in iCalendar, you would have to duplicate the entire event in
// the override. In JSCalendar, this is a small patch to show the difference.
//
// As only this property is patched, if the location of the event is changed, the occurrence will
// automatically still inherit this.
type PatchObject map[string]any

// A Relation object defines the relation to other objects, using a possibly empty set of relation types.
//
// The object that defines this relation is the linking object, while the other object is the linked
// object.
type Relation struct {
	// This specifies the type of this object.
	//
	// This MUST be `Relation`.
	Type TypeOfRelation `json:"@type,omitempty"`

	// This describes how the linked object is related to the linking object.
	//
	// The relation is defined as a set of relation types.
	//
	// If empty, the relationship between the two objects is unspecified.
	//
	// Keys in the set MUST be one of the following values, specified in the
	// property definition where the `Relation` object is used:
	// !- `first`: The linked object is the first in a series the linking object is part of.
	// !- `next`: The linked object is next in a series the linking object is part of.
	// !- `child`: The linked object is a subpart of the linking object.
	// !- `parent`: The linking object is a subpart of the linked object.
	//
	// The value for each key in the map MUST be true.
	Relation map[Relationship]bool `json:"relation,omitempty"`
}

type Link struct {
	// This specifies the type of this object.
	//
	// This MUST be `Link`.
	Type TypeOfLink `json:"@type,omitempty"`

	// This is a URI from which the resource may be fetched.
	//
	// This MAY be a `data:` URL [RFC2397], but it is recommended that the file be hosted on a
	// server to avoid embedding arbitrarily large data in JSCalendar object instances.
	Href string `json:"href"`

	// This MUST be a valid content-id value according to the definition of Section 2 of [RFC2392].
	//
	// The value MUST be unique within this `Link` object but has no meaning beyond that.
	//
	// It MAY be different from the link id for this `Link` object.
	Cid string `json:"cid,omitempty"`

	// This is the media type [RFC6838] of the resource, if known.
	ContentType string `json:"contentType,omitempty"`

	// This is the size, in octets, of the resource when fully decoded
	// (i.e., the number of octets in the file the user would download), if known.
	//
	// Note that this is an informational estimate, and implementations must be prepared to handle
	// the actual size being quite different when the resource is fetched.
	Size uint `json:"size,omitzero"`

	// This identifies the relation of the linked resource to the object.
	//
	// If set, the value MUST be a relation type from the IANA "Link Relations" registry
	// [LINKRELS], as established in [RFC8288].
	Rel Rel `json:"rel,omitempty"`

	// This describes the intended purpose of a link to an image.
	//
	// If set, the `rel` property MUST be set to icon.
	//
	// The value MUST be one of the following values:
	// !- `badge`: an image meant to be displayed alongside the title of the object
	// !- `graphic`: a full image replacement for the object itself
	// !- `fullsize`: an image that is used to enhance the object
	// !- `thumbnail`: a smaller variant of fullsize to be used when space for the image is constrained
	Display Display `json:"display,omitempty"`

	// This is a human-readable, plain-text description of the resource.
	Title string `json:"title,omitempty"`
}

type Location struct {
	// This specifies the type of this object.
	//
	// This MUST be `Location`.
	Type TypeOfLocation `json:"@type,omitempty"`

	// This is the human-readable name of the location.
	Name string `json:"name,omitempty"`

	// This is the human-readable, plain-text instructions for accessing this location.
	//
	// This may be an address, set of directions, door access code, etc.
	Description string `json:"description,omitempty"`

	// This is a set of one or more location types that describe this location.
	//
	// All types MUST be from the "Location Types Registry" [LOCATIONTYPES], as defined in [RFC4589].
	//
	// The set is represented as a map, with the keys being the location types.
	//
	// The value for each key in the map MUST be `true`.
	LocationTypes map[LocationTypeOption]bool `json:"locationTypes,omitempty"`

	// This specifies the relation between this location and the time of the JSCalendar object.
	//
	// This is primarily to allow events representing travel to specify the location of departure (at the
	// start of the event) and location of arrival (at the end); this is particularly important if these
	// locations are in different time zones, as a client may wish to highlight this information for the user.
	//
	// This MUST be one of the following values; any value the client or server doesn't understand
	// should be treated the same as if this property is omitted:
	// !- `start`: The event/task described by this JSCalendar object occurs at this location at the time the event/task starts.
	// !- `end`: The event/task described by this JSCalendar object occurs at this location at the time the event/task ends.
	RelativeTo LocationRelation `json:"relativeTo,omitempty"`

	// This is a time zone for this location.
	TimeZone string `json:"timeZone,omitempty"`

	// This is a geo: URI [RFC5870] for the location.
	Coordinates string `json:"coordinates,omitempty"`

	// This is a map of link ids to `Link` objects, representing external resources associated with this
	// location, for example, a vCard or image.
	//
	// If there are no links, this MUST be omitted (rather than specified as an empty set).
	Links map[string]Link `json:"links,omitempty"`
}

type VirtualLocation struct {
	// This specifies the type of this object. This MUST be `VirtualLocation`.
	Type TypeOfVirtualLocation `json:"@type,omitempty"`

	// This is the human-readable name of the virtual location.
	Name string `json:"name,omitempty"`

	// These are human-readable plain-text instructions for accessing this virtual location.
	//
	// This may be a conference access code, etc.
	Description string `json:"description,omitempty"`

	// Mandatory: this is a URI [RFC3986] that represents how to connect to this virtual location.
	//
	// This may be a telephone number (represented using the `tel:` scheme, e.g., `tel:+1-555-555-5555`)
	// for a teleconference, a web address for online chat, or any custom URI.
	Uri string `json:"uri"`

	// A set of features supported by this virtual location.
	//
	// The set is represented as a map, with the keys being the feature.
	//
	// The value for each key in the map MUST be true.
	//
	// The feature MUST be one of the following values; any value the client or server
	// doesn't understand should be treated the same as if this feature is omitted:
	// !- `audio`: Audio conferencing
	// !- `chat`: Chat or instant messaging
	// !- `feed`: Blog or atom feed
	// !- `moderator`: Provides moderator-specific features
	// !- `phone`: Phone conferencing
	// !- `screen`: Screen sharing
	// !- `video`: Video conferencing
	Features map[VirtualLocationFeature]bool `json:"features,omitempty"`
}

type NDay struct {
	// This specifies the type of this object. This MUST be `NDay`.
	Type TypeOfNDay `json:"@type,omitempty"`

	// This is a day of the week on which to repeat; the allowed values are the same as for the
	// `firstDayOfWeek` `recurrenceRule` property.
	//
	// This is the day of the week of the `BYDAY` part in iCalendar, converted to lowercase.
	Day DayOfWeek `json:"day"`

	// If present, rather than representing every occurrence of the weekday defined in the `day`
	// property, it represents only a specific instance within the recurrence period.
	//
	// The value can be positive or negative but MUST NOT be zero.
	//
	// A negative integer means the nth-last occurrence within that period (i.e., -1 is the last
	// occurrence, -2 the one before that, etc.).
	//
	// This is the ordinal part of the `BYDAY` value in iCalendar (e.g., `1` or `-3`).
	NthOfPeriod int `json:"nthOfPeriod,omitzero"`
}

// A RecurrenceRule object is a JSON object mapping of a `RECUR` value type in iCalendar
// [RFC5545] [RFC7529] and has the same semantics.
//
// [RFC5545]: https://www.rfc-editor.org/rfc/rfc5545.html
// [RFC7529]: https://www.rfc-editor.org/rfc/rfc7529.html
type RecurrenceRule struct {
	// This specifies the type of this object. This MUST be ` RecurrenceRule`.
	Type TypeOfRecurrenceRule `json:"@type,omitempty"`

	// This is the time span covered by each iteration of this recurrence rule.
	//
	// This MUST be one of the following values:
	// !- `yearly`
	// !- `monthly`
	// !- `weekly`
	// !- `daily`
	// !- `hourly`
	// !- `minutely`
	// !- `secondly`
	//
	// This is the `FREQ` part from iCalendar, converted to lowercase.
	Frequency Frequency `json:"frequency,omitempty"`

	// This is the interval of iteration periods at which the recurrence repeats.
	//
	// If included, it MUST be an integer >= `1`.
	//
	// This is the `INTERVAL` part from iCalendar.
	//
	// Default: 1
	Interval uint `json:"interval,omitzero"`

	// This is the calendar system in which this recurrence rule operates, in lowercase.
	//
	// This MUST be either a CLDR-registered calendar system name [CLDR] or a vendor-specific
	// value.
	//
	// This is the `RSCALE` part from iCalendar RSCALE [RFC7529], converted to lowercase.
	//
	// Default: gregorian
	//
	// [CLDR]: https://github.com/unicode-org/cldr/blob/latest/common/bcp47/calendar.xml
	// [RFC7529]: https://www.rfc-editor.org/rfc/rfc7529.html
	Rscale string `json:"rscale,omitempty"`

	// This is the behavior to use when the expansion of the recurrence produces invalid dates.
	//
	// This property only has an effect if the frequency is `yearly` or `monthly`.
	//
	// It MUST be one of the following values:
	// !- `omit`
	// !- `backward`
	// !- `forward`
	//
	// This is the `SKIP` part from iCalendar `RSCALE` [RFC7529], converted to lowercase.
	//
	// Default: omit
	Skip Skip `json:"skip,omitempty"`

	// This is the day on which the week is considered to start, represented as a lowercase, abbreviated,
	// and two-letter English day of the week.
	//
	// If included, it MUST be one of the following values:
	// !- `mo`
	// !- `tu`
	// !- `we`
	// !- `th`
	// !- `fr`
	// !- `sa`
	// !- `su`
	//
	// This is the `WKST` part from iCalendar.
	//
	// Default: mo
	FirstDayOfWeek DayOfWeek `json:"firstDayOfWeek,omitempty"`

	// These are days of the week on which to repeat.
	ByDay []NDay `json:"byDay,omitempty"`

	// These are the days of the month on which to repeat.
	//
	// Valid values are between 1 and the maximum number of days any month may have in the calendar given by
	// the `rscale` property and the negative values of these numbers.
	//
	// For example, in the Gregorian calendar, valid values are `1` to `31` and `-31` to `-1`.
	//
	// Negative values offset from the end of the month.
	//
	// The array MUST have at least one entry if included.
	//
	// This is the `BYMONTHDAY` part in iCalendar.
	ByMonthDay []int `json:"byMonthDay,omitempty"`

	// These are the months in which to repeat.
	//
	// Each entry is a string representation of a number, starting from `"1"` for the first month in the
	// calendar (e.g., `"1"` means January with the Gregorian calendar), with an optional `"L"` suffix
	// (see [RFC7529]) for leap months (this MUST be uppercase, e.g., `"3L"`).
	//
	// The array MUST have at least one entry if included.
	//
	// This is the `BYMONTH` part from iCalendar.
	//
	// [RFC7529]: https://www.rfc-editor.org/rfc/rfc7529.html
	ByMonth []string `json:"byMonth,omitempty"`

	// These are the days of the year on which to repeat.
	//
	// Valid values are between `1` and the maximum number of days any year may have in the calendar given
	// by the `rscale` property and the negative values of these numbers.
	//
	// For example, in the Gregorian calendar, valid values are `1` to `366` and `-366` to `-1`.
	//
	// Negative values offset from the end of the year.
	//
	// The array MUST have at least one entry if included.
	//
	// This is the `BYYEARDAY` part from iCalendar.
	ByYearDay []int `json:"byYearDay,omitempty"`

	// These are the weeks of the year in which to repeat.
	//
	// Valid values are between `1` and the maximum number of weeks any year may have in the calendar
	// given by the `rscale` property and the negative values of these numbers.
	//
	// For example, in the Gregorian calendar, valid values are `1` to `53` and `-53` to `-1`.
	//
	// The array MUST have at least one entry if included.
	//
	// This is the `BYWEEKNO` part from iCalendar.
	ByWeekNo []int `json:"byWeekNo,omitempty"`

	// These are the hours of the day in which to repeat.
	//
	// Valid values are `0` to `23`.
	//
	// The array MUST have at least one entry if included.
	//
	// This is the `BYHOUR` part from iCalendar.
	ByHour []uint `json:"byHour,omitempty"`

	// These are the minutes of the hour in which to repeat.
	//
	// Valid values are `0` to `59`.
	//
	// The array MUST have at least one entry if included.
	//
	// This is the `BYMINUTE` part from iCalendar.
	ByMinute []uint `json:"byMinute,omitempty"`

	// These are the seconds of the minute in which to repeat.
	//
	// Valid values are `0` to `60`.
	//
	// The array MUST have at least one entry if included.
	//
	// This is the `BYSECOND` part from iCalendar.
	BySecond []uint `json:"bySecond,omitempty"`

	// These are the occurrences within the recurrence interval to include in the final results.
	//
	// Negative values offset from the end of the list of occurrences.
	//
	// The array MUST have at least one entry if included.
	//
	// This is the `BYSETPOS` part from iCalendar.
	BySetPosition []int `json:"bySetPosition,omitempty"`

	// These are the number of occurrences at which to range-bound the recurrence.
	//
	// This MUST NOT be included if an until property is specified.
	//
	// This is the `COUNT` part from iCalendar.
	Count uint `json:"count,omitzero"`

	// These are the date-time at which to finish recurring.
	//
	// The last occurrence is on or before this date-time.
	//
	// This MUST NOT be included if a `count` property is specified.
	//
	// Note that if not specified otherwise for a specific JSCalendar object, this date is to be
	// interpreted in the time zone specified in the JSCalendar object's `timeZone` property.
	//
	// This is the UNTIL part from iCalendar.
	Until *LocalDateTime `json:"until,omitempty"`
}

type Participant struct {
	// This specifies the type of this object. This MUST be ` Participant`.
	Type TypeOfParticipant `json:"@type,omitempty"`

	// This is the display name of the participant (e.g., `"Joe Bloggs"``).
	Name string `json:"name,omitempty"`

	// This is the email address to use to contact the participant or, for example,
	// match with an address book entry.
	//
	// If set, the value MUST be a valid addr-spec value as defined in Section 3.4.1 of [RFC5322].
	Email string `json:"email,omitempty"`

	// This is a plain-text description of this participant.
	//
	// For example, this may include more information about their role in the event or how best to contact them.
	Description string `json:"description,omitempty"`

	// This represents methods by which the participant may receive the invitation and updates to the calendar object.
	//
	// The keys in the property value are the available methods and MUST only contain ASCII alphanumeric characters (`A-Za-z0-9`).
	//
	// The value is a URI for the method specified in the key. Future methods may be defined in future specifications and
	// registered with IANA; a calendar client MUST ignore any method it does not understand but MUST preserve the method key and URI.
	//
	// This property MUST be omitted if no method is defined (rather than being specified as an empty object).
	//
	// The following methods are defined:
	// !- `imip`: The participant accepts an iMIP [RFC6047] request at this email address. The value MUST be a `mailto:` URI.
	// It MAY be different from the value of the participant's email property.
	// !- `other``: The participant is identified by this URI, but the method for submitting the invitation is undefined.
	SendTo map[SendToMethod]string `json:"sendTo,omitempty"`

	// This is what kind of entity this participant is, if known.
	//
	// This MUST be one of the following values, another value registered in the IANA "JSCalendar Enum Values" registry,
	// or a vendor-specific value (see Section 3.3).
	//
	// Any value the client or server doesn't understand should be treated the same as if this property is omitted.
	// !- `individual`: a single person
	// !- `group`: a collection of people invited as a whole
	// !- `location`: a physical location that needs to be scheduled, e.g., a conference room
	// !- `resource`: a non-human resource other than a location, such as a projector
	Kind ParticipantKind `json:"kind,omitempty"`

	// This is a set of roles that this participant fulfills.
	//
	// At least one role MUST be specified for the participant.
	//
	// The keys in the set MUST be one of the following values, another value registered in the IANA "JSCalendar Enum Values"
	// registry, or a vendor-specific value (see Section 3.3):
	// !- `owner`:  The participant is an owner of the object. This signifies they have permission to make changes to it
	// that affect the other participants. Nonowner participants may only change properties that affect only themselves
	// (for example, setting their own alerts or changing their RSVP status).
	// !- `attendee`: The participant is expected to be present at the event.
	// !- `optional`: The participant's involvement with the event is optional. This is expected to be primarily combined
	// with the `"attendee"` role.
	// !- `informational`: The participant is copied for informational reasons and is not expected to attend.
	// !- `chair`: The participant is in charge of the event/task when it occurs.
	// !- `contact`:  The participant is someone that may be contacted for information about the event.
	//
	// The value for each key in the map MUST be true. It is expected that no more than one of the roles
	// `"attendee"` and `"informational"` be present; if more than one are given, `"attendee"` takes precedence
	// over `"informational"`.
	//
	// Roles that are unknown to the implementation MUST be preserved.
	Roles map[Role]bool `json:"roles,omitempty"`

	// This is the location at which this participant is expected to be attending.
	//
	// If the value does not correspond to any location id in the `locations` property of the JSCalendar object,
	// this MUST be treated the same as if the participant's `locationId` were omitted.
	LocationId string `json:"locationId,omitempty"`

	// This is the language tag, as defined in [RFC5646], that best describes the participant's preferred language, if known.
	Language string `json:"language,omitempty"`

	// This is the participation status, if any, of this participant.
	//
	// The value MUST be one of the following values, another value registered in the IANA "JSCalendar Enum Values" registry,
	// or a vendor-specific value (see Section 3.3):
	// !- `needs-action`: No status has yet been set by the participant.
	// !- `accepted`: The invited participant will participate.
	// !- `declined`: The invited participant will not participate.
	// !- `tentative`: The invited participant may participate.
	// !- `delegated`: The invited participant has delegated their attendance to another participant, as specified in the `delegatedTo` property.
	ParticipationStatus ParticipationStatus `json:"participationStatus,omitempty"`

	// This is a note from the participant to explain their participation status.
	ParticipationComment string `json:"participationComment,omitempty"`

	// If true, the organizer is expecting the participant to notify them of their participation status.
	ExpectReply bool `json:"expectReply,omitzero"`

	// This is who is responsible for sending scheduling messages with this calendar object to the participant.
	//
	// The value MUST be one of the following values, another value registered in the IANA "JSCalendar Enum Values"
	// registry, or a vendor-specific value (see Section 3.3):
	// !- `server`: The calendar server will send the scheduling messages.
	// !- `client`: The calendar client will send the scheduling messages.
	// !- `none`: No scheduling messages are to be sent to this participant.
	//
	// Default: server
	ScheduleAgent ScheduleAgent `json:"scheduleAgent,omitempty"`

	// A client may set the property on a participant to true to request that the server send a scheduling
	// message to the participant when it would not normally do so (e.g., if no significant change is made
	// the object or the scheduleAgent is set to client).
	//
	// The property MUST NOT be stored in the JSCalendar object on the server or appear in a scheduling message.
	ScheduleForceSend bool `json:"scheduleForceSend,omitzero"`

	// This is the sequence number of the last response from the participant.
	//
	// If defined, this MUST be a nonnegative integer.
	//
	// This can be used to determine whether the participant has sent a new response following significant
	// changes to the calendar object and to determine if future responses are responding to a current or older view of the data.
	ScheduleSequence uint `json:"scheduleSequence,omitzero"`

	// This is a list of status codes, returned from the processing of the most recent scheduling message sent to this participant.
	//
	// The status codes MUST be valid statcode values as defined in the ABNF in [Section 3.8.8.3 of RFC5545].
	//
	// Servers MUST only add or change this property when they send a scheduling message to the participant.
	//
	// Clients SHOULD NOT change or remove this property if it was provided by the server.
	//
	// Clients MAY add, change, or remove the property for participants where the client is handling the scheduling.
	//
	// This property MUST NOT be included in scheduling messages.
	//
	// [Section 3.8.8.3 of RFC5545]: https://www.rfc-editor.org/rfc/rfc5545#section-3.8.8.3
	ScheduleStatus []string `json:"scheduleStatus,omitempty"`

	// This is the timestamp for the most recent response from this participant.
	//
	// This is the updated property of the last response when using iTIP. It can be compared to the updated property in
	// future responses to detect and discard older responses delivered out of order.
	ScheduleUpdated time.Time `json:"scheduleUpdated,omitzero"`

	// This is the email address in the `"From"` header of the email that last updated this participant via iMIP.
	//
	// This SHOULD only be set if the email address is different to that in the mailto URI of this participant's `imip`
	// method in the `sendTo` property (i.e., the response was received from a different address to that which the
	// invitation was sent to). If set, the value MUST be a valid addr-spec value as defined in Section 3.4.1 of [RFC5322].
	SentBy string `json:"sentBy,omitempty"`

	// This is the id of the participant who added this participant to the event/task, if known.
	InvitedBy string `json:"invitedBy,omitempty"`

	// This is set of participant ids that this participant has delegated their participation to.
	//
	// Each key in the set MUST be the id of a participant.
	//
	// The value for each key in the map MUST be true.
	//
	// If there are no delegates, this MUST be omitted (rather than specified as an empty set).
	DelegatedTo map[string]bool `json:"delegatedTo,omitempty"`

	// This is a set of participant ids that this participant is acting as a delegate for.
	//
	// Each key in the set MUST be the id of a participant.
	//
	// The value for each key in the map MUST be true.
	//
	// If there are no delegators, this MUST be omitted (rather than specified as an empty set).
	DelegatedFrom map[string]bool `json:"delegatedFrom,omitempty"`

	// This is a set of group participants that were invited to this calendar object, which caused this participant to
	// be invited due to their membership in the group(s).
	//
	// Each key in the set MUST be the id of a participant.
	//
	// The value for each key in the map MUST be true.
	//
	// If there are no groups, this MUST be omitted (rather than specified as an empty set).
	MemberOf map[string]bool `json:"memberOf,omitempty"`

	// This is a map of link ids to `Link` objects, representing external resources associated with this participant,
	// for example, a vCard or image.
	//
	// Only allowed for participants of a Task.
	//
	// If there are no links, this MUST be omitted (rather than specified as an empty set).
	Links map[string]Link `json:"links,omitempty"`

	// This represents the progress of the participant for this task.
	//
	// It MUST NOT be set if the `participationStatus` of this participant is any value other than `accepted`.
	//
	// Only allowed for participants of a Task.
	//
	// See Section 5.2.5 for allowed values and semantics.
	Progress Progress `json:"progress,omitempty"`

	// This specifies the date-time the progress property was last set on this participant.
	//
	// Only allowed for participants of a Task.
	//
	// See Section 5.2.6 for allowed values and semantics.
	ProgressUpdated time.Time `json:"progressUpdated,omitzero"`

	// This represents the percent completion of the participant for this task.
	//
	// Only allowed for participants of a Task.
	//
	// The property value MUST be a positive integer between 0 and 100.
	PercentComplete uint `json:"percentComplete,omitzero"`

	// This is a URI as defined by [@!RFC3986] or any other IANA-registered form for a URI.
	//
	// It is the same as the `CAL-ADDRESS` value of an `ATTENDEE` or `ORGANIZER` in iCalendar ([@!RFC5545]);
	// it globally identifies a particular participant, even across different events.
	//
	// This is a JMAP addition to JSCalendar.
	ScheduleId string `json:"scheduleId,omitempty"`
}

type Trigger interface {
	trigger()
}

type OffsetTrigger struct {
	// This specifies the type of this object. This MUST be `OffsetTrigger`.
	Type TypeOfOffsetTrigger `json:"@type,omitempty"`

	// This defines the offset at which to trigger the alert relative to the time property defined in
	// the `relativeTo` property of the alert.
	//
	// Negative durations signify alerts before the time property;
	// positive durations signify alerts after the time property.
	Offset SignedDuration `json:"offset"`

	// This specifies the time property that the alert offset is relative to.
	//
	// The value MUST be one of the following:
	// !- `start`: triggers the alert relative to the start of the calendar object
	// !- `end`: triggers the alert relative to the end/due time of the calendar object
	RelativeTo RelativeTo `json:"relativeTo,omitempty"`
}

var _ Trigger = OffsetTrigger{}

func (o OffsetTrigger) trigger() {}

type AbsoluteTrigger struct {
	// This specifies the type of this object. This MUST be `AbsoluteTrigger`.
	Type TypeOfAbsoluteTrigger `json:"@type,omitempty"`

	// This defines a specific UTC date-time when the alert is triggered.
	When time.Time `json:"when"`
}

var _ Trigger = AbsoluteTrigger{}

func (o AbsoluteTrigger) trigger() {}

// An `UnknownTrigger` object is an object that contains an `@type` property whose value is not recognized
// (i.e., not `OffsetTrigger` or `AbsoluteTrigger`) plus zero or more other properties.
//
// This is for compatibility with client extensions and future specifications.
//
// Implementations SHOULD NOT trigger for trigger types they do not understand but MUST preserve them.
type UnknownTrigger map[string]any

var _ Trigger = UnknownTrigger{}

func (o UnknownTrigger) trigger() {}

type Alert struct {
	// This specifies the type of this object. This MUST be `Alert`.
	Type TypeOfAlert `json:"@type,omitempty"`

	// This defines when to trigger the alert.
	//
	// New types may be defined in future documents.
	Trigger Trigger `json:"trigger"`

	// This records when an alert was last acknowledged.
	//
	// This is set when the user has dismissed the alert; other clients that sync this property
	// SHOULD automatically dismiss or suppress duplicate alerts (alerts with the same alert id
	// that triggered on or before this date-time).
	//
	// For a recurring calendar object, setting the acknowledged property MUST NOT add a new override
	// to the recurrenceOverrides property.
	//
	// If the alert is not already overridden, the acknowledged property MUST be set on the alert
	// in the base event/task.
	//
	// Certain kinds of alert action may not provide feedback as to when the user sees them, for example,
	// email-based alerts.
	//
	// For those kinds of alerts, this property MUST be set immediately when the alert is triggered
	// and the action is successfully carried out.
	Acknowledged time.Time `json:"acknowledged,omitzero"`

	// This relates this alert to other alerts in the same JSCalendar object.
	//
	// If the user wishes to snooze an alert, the application MUST create an alert to trigger after snoozing.
	// This new snooze alert MUST set a parent relation to the identifier of the original alert.
	RelatedTo map[string]Relation `json:"relatedTo,omitempty"`

	// This describes how to alert the user.
	//
	// The value MUST be at most one of the following values, a value registered in the IANA "JSCalendar Enum Values"
	// registry, or a vendor-specific value (see Section 3.3):
	// !- `display`: The alert should be displayed as appropriate for the current device and user context.
	// !- `email`: The alert should trigger an email sent out to the user, notifying them of the alert. This action is
	// typically only appropriate for server implementations.
	//
	// Default: display
	Action AlertAction `json:"action,omitempty"`
}

func (a *Alert) UnmarshalJSON(b []byte) error {
	var typ struct {
		Trigger struct {
			Type string `json:"@type"`
		} `json:"trigger"`
	}
	if err := json.Unmarshal(b, &typ); err != nil {
		return err
	}
	switch typ.Trigger.Type {
	case string(OffsetTriggerType):
		a.Trigger = new(OffsetTrigger)
	case string(AbsoluteTriggerType):
		a.Trigger = new(AbsoluteTrigger)
	default:
		a.Trigger = new(UnknownTrigger)
	}

	type tmp Alert
	return json.Unmarshal(b, (*tmp)(a))
}

// A `TimeZoneRule` object maps a `STANDARD` or `DAYLIGHT` sub-component from iCalendar,
// with the restriction that, at most, one recurrence rule is allowed per rule.
type TimeZoneRule struct {
	// This specifies the type of this object. This MUST be `TimeZoneRule`.
	Type TypeOfTimeZoneRule `json:"@type,omitempty"`

	// This is the `DTSTART` property from iCalendar.
	Start LocalDateTime `json:"start"`

	// This is the `TZOFFSETFROM` property from iCalendar: specifies the offset that is in use prior to this time zone observance.
	//
	// This property specifies the offset that is in use prior to this time observance.
	//
	// It is used to calculate the absolute time at which the transition to a given observance takes place.
	//
	// The property value is a signed numeric indicating the number of hours and possibly minutes from UTC.
	//
	// Positive numbers represent time zones east of the prime meridian, or ahead of UTC.
	//
	// Negative numbers represent time zones west of the prime meridian, or behind UTC.
	//
	// Mandatory.
	//
	// example: -0500
	OffsetFrom string `json:"offsetFrom"`

	// This is the TZOFFSETTO property from iCalendar: specifies the offset that is in use in this time zone observance.
	//
	// This property specifies the offset that is in use in this time zone observance.
	//
	// It is used to calculate the absolute time for the new observance.
	//
	// The property value is a signed numeric indicating the number of hours and possibly minutes from UTC.
	//
	// Positive numbers represent time zones east of the prime meridian, or ahead of UTC.
	//
	// Negative numbers represent time zones west of the prime meridian, or behind UTC.
	//
	// Mandatory.
	//
	// example: +1245
	OffsetTo string `json:"offsetTo"`

	// This is the `RRULE` property mapped.
	//
	// uring recurrence rule evaluation, the `until` property value MUST be interpreted
	// as a local time in the UTC time zone.
	RecurrenceRules []RecurrenceRule `json:"recurrenceRules,omitempty"`

	// This maps the `RDATE` properties from iCalendar.
	//
	// The set is represented as an object, with the keys being the recurrence dates.
	//
	// The patch object MUST be the empty JSON object (`{}`).
	RecurrenceOverrides map[LocalDateTime]PatchObject `json:"recurrenceOverrides,omitempty"`

	// This maps the `TZNAME` properties from iCalendar to a JSON set.
	//
	// The set is represented as an object, with the keys being the names, excluding any
	// `tznparam` component from iCalendar.
	//
	// The value for each key in the map MUST be true.
	Names map[string]bool `json:"names,omitempty"`

	// This maps the `COMMENT` properties from iCalendar.
	//
	// The order MUST be preserved during conversion.
	Comments []string `json:"comments,omitempty"`
}

type TimeZone struct {
	// This specifies the type of this object. This MUST be `TimeZone`.
	Type TypeOfTimeZone `json:"@type,omitempty"`

	// This is the TZID property from iCalendar.
	//
	// Note that this implies that the value MUST be a valid `paramtext` value as specified in Section 3.1. of [RFC5545].
	TzId string `json:"tzId"`

	// This is the `LAST-MODIFIED` property from iCalendar.
	Updated time.Time `json:"updated,omitzero"`

	// This is the `TZURL` property from iCalendar.
	Url string `json:"url,omitempty"`

	// This is the TZUNTIL property from iCalendar, specified in [RFC7808].
	ValidUntil time.Time `json:"validUntil,omitzero"`

	// This maps the `TZID-ALIAS-OF` properties from iCalendar, specified in [RFC7808], to a JSON set of aliases.
	//
	// The set is represented as an object, with the keys being the aliases.
	//
	// The value for each key in the map MUST be `true`.
	Aliases map[string]bool `json:"aliases,omitempty"`

	// This the `STANDARD` sub-components from iCalendar.
	//
	// The order MUST be preserved during conversion.
	Standard []TimeZoneRule `json:"standard,omitempty"`

	// This the `DAYLIGHT` sub-components from iCalendar.
	//
	// The order MUST be preserved during conversion.
	Daylight []TimeZoneRule `json:"daylight,omitempty"`
}

type CommonObject struct {
	// This is a globally unique identifier used to associate objects representing the same event,
	// task, group, or other object across different systems, calendars, and views.
	//
	// For recurring events and tasks, the UID is associated with the base object and therefore
	// is the same for all occurrences; the combination of the UID with a recurrenceId identifies
	// a particular instance.
	//
	// The generator of the identifier MUST guarantee that the identifier is unique.
	//
	// [RFC4122] describes a range of established algorithms to generate universally unique identifiers
	// (UUIDs). UUID version 4, described in Section 4.4 of [RFC4122], is RECOMMENDED.
	//
	// For compatibility with UIDs [RFC5545], implementations MUST be able to receive and persist
	// values of at least 255 octets for this property, but they MUST NOT truncate values in the
	// middle of a UTF-8 multi-octet sequence.
	Uid string `json:"uid"`

	// This is the identifier for the product that last updated the JSCalendar object.
	//
	// This should be set whenever the data in the object is modified (i.e., whenever the updated property is set)
	//
	// .The vendor of the implementation MUST ensure that this is a globally unique identifier, using
	// ome technique such as a Formal Public Identifier (FPI) value, as defined in [ISO.9070.1991].
	//
	// This property SHOULD NOT be used to alter the interpretation of a JSCalendar object beyond the semantics
	// specified in this document.
	//
	// For example, it is not to be used to further the understanding of nonstandard properties, a practice
	// that is known to cause long-term interoperability problems.
	ProdId string `json:"prodId,omitempty"`

	// This is the date and time this object was initially created.
	//
	// TODO serialize as UTCDateTime
	Created time.Time `json:"created,omitzero"`

	// This is the date and time the data in this object was last modified (or its creation date/time
	// if not modified since).
	//
	// TODO serialize as UTCDateTime
	Updated time.Time `json:"updated,omitzero"`

	// This is a short summary of the object.
	Title string `json:"title,omitempty"`

	// This is a longer-form text description of the object.
	//
	// The content is formatted according to the `descriptionContentType` property.
	Description string `json:"description,omitempty"`

	// This describes the media type [RFC6838] of the contents of the description property.
	//
	// Media types MUST be subtypes of type text and SHOULD be text/plain or text/html [MEDIATYPES].
	//
	// They MAY include parameters, and the charset parameter value MUST be utf-8, if specified.
	//
	// Descriptions of type text/html MAY contain cid URLs [RFC2392] to reference links in the calendar
	// object by use of the cid property of the Link object.
	//
	// Default: text/plain
	DescriptionContentType string `json:"descriptionContentType,omitempty"`

	// This is a map of link ids to `Link` objects, representing external resources associated with the object.
	//
	// Links with a `rel` of `enclosure` MUST be considered by the client to be attachments for download.
	//
	// Links with a `rel` of `describedby` MUST be considered by the client to be alternative representations of the
	// `description`.
	//
	// Links with a `rel` of `icon` MUST be considered by the client to be images that it may use when presenting
	// the calendar data to a user. The `display` property may be set to indicate the purpose of this image.
	Links map[string]Link `json:"links,omitempty"`

	// This is the language tag, as defined in [RFC5646], that best describes the locale used for the text in
	// the calendar object, if known.
	//
	// [RFC5646]: https://www.rfc-editor.org/rfc/rfc5646.html
	Locale string `json:"locale,omitempty"`

	// This is a set of keywords or tags that relate to the object.
	//
	// The set is represented as a map, with the keys being the keywords.
	//
	// The value for each key in the map MUST be `true`.
	Keywords map[string]bool `json:"keywords,omitempty"`

	// This is a set of categories that relate to the calendar object.
	//
	// The set is represented as a map, with the keys being the categories specified as URIs.
	//
	// The value for each key in the map MUST be `true`.
	//
	// In contrast to keywords, categories are typically structured.
	//
	// For example, a vendor owning the domain `example.com` might define the categories
	// `http://example.com/categories/sports/american-football` and `http://example.com/categories/music/r-b`.
	Categories map[string]bool `json:"categories,omitempty"`

	// This is a color clients MAY use when displaying this calendar object.
	//
	// The value is a color name taken from the set of names defined in [Section 4.3 of CSS Color Module Level 3]
	// or an RGB value in hexadecimal notation, as defined in [Section 4.2.1 of CSS Color Module Level 3].
	//
	// [Section 4.3 of CSS Color Module Level 3]: https://www.w3.org/TR/css-color-3/#svg-color
	// [Section 4.2.1 of CSS Color Module Level 3]: https://www.w3.org/TR/css-color-3/#rgb-color
	Color string `json:"color,omitempty"`

	// This maps identifiers of custom time zones to their time zone definitions.
	//
	// The following restrictions apply for each key in the map:
	// !- To avoid conflict with names in the IANA Time Zone Database [TZDB], it MUST start with the `/` character.
	// !- It MUST be a valid `paramtext` value, as specified in Section 3.1 of [RFC5545].
	// !- At least one other property in the same JSCalendar object MUST reference a time zone using this identifier (i.e.,
	// orphaned time zones are not allowed).
	//
	// An identifier need only be unique to this JSCalendar object.
	//
	// It MAY differ from the tzId property value of the TimeZone object it maps to.
	//
	// A JSCalendar object may be part of a hierarchy of other JSCalendar objects (say, an `Event` is an entry in a `Group`).
	//
	// In this case, the set of time zones is the sum of the time zone definitions of this object and its parent objects.
	//
	// If multiple time zones with the same identifier exist, then the definition closest to the calendar object in relation
	// to its parents MUST be used.
	//
	// (In context of `Event`, a time zone definition in its `timeZones` property has precedence over a definition of the
	// same id in the `Group`).
	//
	// Time zone definitions in any children of the calendar object MUST be ignored.
	//
	// A `TimeZone` object maps a `VTIMEZONE` component from iCalendar, and the semantics are as defined in [RFC5545].
	//
	// A valid time zone MUST define at least one transition rule in the `standard` or `daylight` property.
	TimeZones map[string]TimeZone `json:"timeZones,omitempty"`
}

// TODO
//
// ### Recurrence Properties
//
// Some events and tasks occur at regular or irregular intervals. Rather than having to copy the data for every occurrence,
// there can be a base event with rules to generate recurrences and/or overrides that add extra dates or exceptions to the rules.
//
// The recurrence set is the complete set of instances for an object. It is generated by considering the following properties in
// order, all of which are optional:
// !- The `recurrenceRules` property generates a set of extra date-times on which the object occurs.
// !- The `excludedRecurrenceRules` property generates a set of date-times that are to be removed from the previously generated
// set of date-times on which the object occurs.
// !- The `recurrenceOverrides` property defines date-times that are added or excluded to form the final set. (This property
// may also contain changes to the object to apply to particular instances.)
type Object struct {
	CommonObject

	// This relates the object to other JSCalendar objects.
	//
	// This is represented as a map of the UIDs of the related objects to information about the relation.
	//
	// If an object is split to make a "this and future" change to a recurrence, the original object MUST
	// be truncated to end at the previous occurrence before this split, and a new object is created to
	// represent all the occurrences after the split.
	//
	// A next relation MUST be set on the original object's relatedTo property for the UID of the new object.
	//
	// A first relation for the UID of the first object in the series MUST be set on the new object.
	// Clients can then follow these UIDs to get the complete set of objects if the user wishes to modify
	// them all at once.
	RelatedTo map[string]Relation `json:"relatedTo,omitempty"`

	// Initially zero, this MUST be incremented by one every time a change is made to the object, except
	// if the change only modifies the `participants` property.
	//
	// This is used as part of the iCalendar Transport-independent Interoperability Protocol (iTIP) [RFC5546]
	// to know which version of the object a scheduling message relates to.
	Sequence uint `json:"sequence,omitzero"`

	/*
		// CalendarEvent objects MUST NOT have a “method” property as this is only used when representing iTIP
		// [@!RFC5546] scheduling messages, not events in a data store.
		Method Method `json:"method,omitempty"`
	*/

	// This indicates that the time is not important to display to the user when rendering this calendar object.
	//
	// An example of this is an event that conceptually occurs all day or across multiple days, such as
	// `"New Year's Day"` or `"Italy Vacation"`.
	//
	// While the time component is important for free-busy calculations and checking for scheduling clashes,
	// calendars may choose to omit displaying it and/or display the object separately to other objects to
	// enhance the user's view of their schedule.
	//
	// Such events are also commonly known as "all-day" events.
	//
	// Default: false
	ShowWithoutTime bool `json:"showWithoutTime,omitzero"`

	// This is a map of location ids to `Location` objects, representing locations associated with the object.
	Locations map[string]Location `json:"locations,omitempty"`

	// This is a map of virtual location ids to VirtualLocation objects, representing virtual locations, such as
	// video conferences or chat rooms, associated with the object.
	VirtualLocations map[string]VirtualLocation `json:"virtualLocations,omitempty"`

	// If present, this JSCalendar object represents one occurrence of a recurring JSCalendar object.
	//
	// If present, the `recurrenceRules` and `recurrenceOverrides` properties MUST NOT be present.
	//
	// The value is a date-time either produced by the `recurrenceRules` of the base event or
	// added as a key to the `recurrenceOverrides` property of the base event.
	RecurrenceId *LocalDateTime `json:"recurrenceId,omitempty"`

	// Identifies the time zone of the main JSCalendar object, of which this JSCalendar object is a recurrence instance.
	//
	// This property MUST be set if the `recurrenceId` property is set.
	//
	// It MUST NOT be set if the `recurrenceId` property is not set.
	RecurrenceIdTimeZone string `json:"recurrenceIdTimeZone,omitempty"`

	// This defines a set of recurrence rules (repeating patterns) for recurring calendar objects.
	//
	// TODO select the right documentation for each copy of the Object class:
	//
	// An Event recurs by applying the recurrence rules to the start date-time.
	//
	// A Task recurs by applying the recurrence rules to the start date-time, if defined; otherwise, it recurs by
	// the due date-time, if defined. If the task defines neither a start nor due date-time, it MUST NOT
	// define a `recurrenceRules` property.
	//
	// If multiple recurrence rules are given, each rule is to be applied, and then the union of the results are used,
	// ignoring any duplicates.
	RecurrenceRules []RecurrenceRule `json:"recurrenceRules,omitempty"`

	// This defines a set of recurrence rules (repeating patterns) for date-times on which the object will not occur.
	//
	// The rules are interpreted the same as for the `recurrenceRules` property, with the exception that the initial
	// date-time to which the rule is applied (the `"start"` date-time for events or the `"start"` or `"due"`
	// date-time for tasks) is only considered part of the expansion if it matches the rule.
	//
	// The resulting set of date-times is then removed from those generated by the `recurrenceRules` property.
	ExcludedRecurrenceRules []RecurrenceRule `json:"excludedRecurrenceRules,omitempty"`

	// Maps recurrence ids (the date-time produced by the recurrence rule) to the overridden properties of the
	// recurrence instance.
	//
	// If the recurrence id does not match a date-time from the recurrence rule (or no rule is specified), it
	// is to be treated as an additional occurrence (like an `RDATE` from iCalendar).
	//
	// The patch object may often be empty in this case.
	//
	// If the patch object defines the `excluded` property of an occurrence to be `true`, this occurrence is
	// omitted from the final set of recurrences for the calendar object (like an `EXDATE` from iCalendar).
	//
	// Such a patch object MUST NOT patch any other property.
	//
	// By default, an occurrence inherits all properties from the main object except the start (or due)
	// date-time, which is shifted to match the recurrence id `LocalDateTime`.
	//
	// However, individual properties of the occurrence can be modified by a patch or multiple patches.
	//
	// It is valid to patch the `start` property value, and this patch takes precedence over the value
	// generated from the recurrence id.
	//
	// Both the recurrence id as well as the patched start date-time may occur before the original JSCalendar
	// object's start or due date.
	//
	// A pointer in the `PatchObject` MUST be ignored if it starts with one of the following prefixes:
	// !- `@type`
	// !- `excludedRecurrenceRules`
	// !- `method`
	// !- `privacy`
	// !- `prodId`
	// !- `recurrenceId`
	// !- `recurrenceIdTimeZone`
	// !- `recurrenceOverrides`
	// !- `recurrenceRules`
	// !- `relatedTo`
	// !- `replyTo`
	// !- `sentBy`
	// !- `timeZones`
	// !- `uid`
	RecurrenceOverrides map[LocalDateTime]PatchObject `json:"recurrenceOverrides,omitempty"`

	// This defines if this object is an overridden, excluded instance of a recurring JSCalendar object.
	//
	// If this property value is `true`, this calendar object instance MUST be removed from the occurrence expansion.
	//
	// The absence of this property, or the presence of its default value as `false`, indicates that this
	// instance MUST be included in the occurrence expansion.
	Excluded bool `json:"excluded,omitzero"`

	// This specifies a priority for the calendar object.
	//
	// This may be used as part of scheduling systems to help resolve conflicts for a time period.
	//
	// The priority is specified as an integer in the range `0` to `9`.
	//
	// A value of `0` specifies an undefined priority, for which the treatment will vary by situation.
	//
	// A value of `1` is the highest priority.
	//
	// A value of `2` is the second highest priority.
	//
	// Subsequent numbers specify a decreasing ordinal priority.
	//
	// A value of `9` is the lowest priority.
	//
	// Other integer values are reserved for future use.
	Priority int `json:"priority,omitzero"`

	// This specifies how this calendar object should be treated when calculating free-busy state.
	//
	// This MUST be one of the following values, another value registered in the IANA
	// "JSCalendar Enum Values" registry, or a vendor-specific value (see Section 3.3):
	// !- `free`
	// !- `busy` (default)
	FreeBusyStatus FreeBusyStatus `json:"freeBusyStatus,omitempty"`

	// Privacy level.
	//
	// Calendar objects are normally collected together and may be shared with other users.
	// The `privacy` property allows the object owner to indicate that it should not be shared or should
	// only have the time information shared but the details withheld.
	//
	// Enforcement of the restrictions indicated by this property is up to the API via which this object is accessed.
	//
	// This property MUST NOT affect the information sent to scheduled participants; it is only
	// interpreted by protocols that share the calendar objects belonging to one user with other users.
	//
	// The value MUST be one of the following values, another value registered in the IANA "JSCalendar Enum Values"
	// registry, or a vendor-specific value (see Section 3.3).
	//
	// Any value the client or server doesn't understand should be preserved but treated as equivalent to private.
	//
	// !- `public`: The full details of the object are visible to those whom the object's calendar is shared with.
	// !- `private`: The details of the object are hidden; only the basic time and metadata are shared.
	// !- `secret`: The object is hidden completely (as though it did not exist) when the calendar this object is in is shared.
	//
	// When the `privacy` property is set to `private`, the following properties MAY be shared; any other
	// properties MUST NOT be shared:
	// !- `@type`
	// !- `created`
	// !- `due`
	// !- `duration`
	// !- `estimatedDuration`
	// !- `freeBusyStatus`
	// !- `privacy`
	// !- `recurrenceOverrides` (Only patches that apply to another permissible property are allowed to be shared.)
	// !- `sequence`
	// !- `showWithoutTime`
	// !- `start`
	// !- `timeZone`
	// !- `timeZones`
	// !- `uid`
	// !- `updated`
	Privacy Privacy `json:"privacy,omitempty"`

	// This represents methods by which participants may submit their response to the organizer of the calendar object.
	//
	// The keys in the property value are the available methods and MUST only contain ASCII alphanumeric characters
	// (`A-Za-z0-9`). The value is a URI for the method specified in the key.
	//
	// Future methods may be defined in future specifications and registered with IANA; a calendar client MUST
	// ignore any method it does not understand but MUST preserve the method key and URI.
	//
	// This property MUST be omitted if no method is defined (rather than being specified as an empty object).
	//
	// The following methods are defined:
	// !- `imip`: The organizer accepts an iCalendar Message-Based Interoperability Protocol (iMIP)
	// [RFC6047] response at this email address. The value MUST be a `mailto:` URI.
	// !- `web`: Opening this URI in a web browser will provide the user with a page where they can
	// submit a reply to the organizer. The value MUST be a URL using the `https:` scheme.
	// !- `other`: The organizer is identified by this URI, but the method for submitting the response
	// is undefined.
	ReplyTo map[ReplyMethod]string `json:"replyTo,omitempty"`

	// This is the email address in the `"From"` header of the email in which this calendar object was received.
	//
	// This is only relevant if the calendar object is received via iMIP or as an attachment to a message.
	//
	// If set, the value MUST be a valid addr-spec value as defined in Section 3.4.1 of [RFC5322].
	SentBy string `json:"sentBy,omitempty"`

	// This is a map of participant ids to participants, describing their participation in the calendar object.
	//
	// If this property is set and any participant has a `sendTo` property, then the `replyTo` property of this
	// calendar object MUST define at least one reply method.
	Participants map[string]Participant `json:"participants,omitempty"`

	// A request status as returned from processing the most recent scheduling request for this JSCalendar object.
	//
	// The allowed values are defined by the ABNF definitions of `statcode`, `statdesc` and `extdata` in
	// Section 3.8.8.3 of [RFC5545] and the following ABNF [RFC5234]:
	//
	// ```text
	// reqstatus = statcode ";" statdesc [";" extdata]
	// ```
	//
	// Servers MUST only add or change this property when they performe a scheduling action.
	//
	// Clients SHOULD NOT change or remove this property if it was provided by the server.
	//
	// Clients MAY add, change, or remove the property when the client is handling the scheduling.
	//
	// This property MUST only be included in scheduling messages according to the rules defined for the
	// `REQUEST-STATUS` iCalendar property in [RFC5546].
	RequestStatus string `json:"requestStatus,omitempty"`

	// If `true`, use the user's default alerts and ignore the value of the alerts property.
	//
	// Fetching user defaults is dependent on the API from which this JSCalendar object is being fetched and
	// is not defined in this specification.
	//
	// If an implementation cannot determine the user's default alerts, or none are set, it MUST process
	// he alerts property as if `useDefaultAlerts` is set to false.
	//
	// Default: false
	UseDefaultAlerts bool `json:"useDefaultAlerts,omitzero"`

	// This is a map of alert ids to Alert objects, representing alerts/reminders to display or send
	// to the user for this calendar object.
	Alerts map[string]Alert `json:"alerts,omitempty"`

	// A map where each key is a language tag [RFC5646], and the corresponding value is a set of patches
	// to apply to the calendar object in order to localize it into that locale.
	//
	// See the description of PatchObject (Section 1.4.9) for the structure of the PatchObject.
	//
	// The patches are applied to the top-level calendar object. In addition, the locale property of the patched
	// object is set to the language tag.
	//
	// All pointers for patches MUST end with one of the following suffixes; any patch that does not follow
	// this MUST be ignored unless otherwise specified in a future RFC:
	// !- `title`
	// !- `description`
	// !- `name`
	//
	// A patch MUST NOT have the prefix `recurrenceOverrides`; any localization of the override MUST be a
	// patch to the `localizations` property inside the override instead.
	//
	// For example, a patch to `locations/abcd1234/title` is permissible, but a patch to `uid` or
	// `recurrenceOverrides/2020-01-05T14:00:00/title` is not.
	//
	// Note that this specification does not define how to maintain validity of localized content.
	//
	// For example, a client application changing a JSCalendar object's `title` property might also
	// need to update any localizations of this property. Client implementations SHOULD provide the means
	// to manage localizations, but how to achieve this is specific to the application's workflow and requirements.
	Localizations map[string]PatchObject `json:"localizations,omitempty"`

	// This identifies the time zone the object is scheduled in or is null for floating time.
	//
	// This is either a name from the IANA Time Zone Database [TZDB] or the `TimeZoneId` of a custom time zone
	// from the `timeZones property`.
	//
	// If omitted, this MUST be presumed to be `null` (i.e., floating time).
	TimeZone string `json:"timeZone,omitempty"`

	// If true, any user may add themselves to the event as a participant with the
	// `attendee` role.
	//
	// This property MUST NOT be altered in the `recurrenceOverrides`; it may only be set on the base object.
	//
	// This indicates the event will accept "party crasher" RSVPs via iTIP, subject to any
	// other domain-specific restrictions, and users may add themselves to the event via JMAP as
	// long as they have the mayRSVP permission for the calendar.
	//
	// This is a JMAP addition to JSCalendar.
	//
	// default: false
	MayInviteSelf bool `json:"mayInviteSelf,omitzero"`

	// If true, any current participant with the `attendee` role may add new participants with the
	// `attendee` role to the event.
	//
	// This property MUST NOT be altered in the `recurrenceOverrides`; it may only be set on the base object.
	//
	// The `mayRSVP` permission for the calendar is also required in conjunction with this event property
	// for users to be allowed to make this change via JMAP.
	//
	// This is a JMAP addition to JSCalendar.
	//
	// default: false
	MayInviteOthers bool `json:"mayInviteOthers,omitzero"`

	// If true, only the owners of the event may see the full set of participants.
	//
	// Other sharees of the event may only see the owners and themselves.
	//
	// This property MUST NOT be altered in the `recurrenceOverrides`; it may only be set on the base object.
	//
	// This is a JMAP addition to JSCalendar.
	//
	// default: false
	HideAttendees bool `json:"hideAttendees,omitzero"`
}

type Event struct {
	Type TypeOfEvent `json:"@type,omitempty"`

	Object

	// This is the date/time the event starts in the event's time zone (as specified in the timeZone property, see Section 4.7.1).
	Start LocalDateTime `json:"start"`

	// This is the zero or positive duration of the event in the event's start time zone.
	//
	// The end time of an event can be found by adding the duration to the event's start time.
	//
	// An Event MAY involve start and end locations that are in different time zones
	// (e.g., a transcontinental flight). This can be expressed using the `relativeTo` and `timeZone` properties of
	// the `Event`'s Location objects (see Section 4.2.5).
	Duration Duration `json:"duration,omitempty"`

	// This is the scheduling status (Section 4.4) of an Event.
	//
	// If set, it MUST be one of the following values, another value registered in the IANA
	// "JSCalendar Enum Values" registry, or a vendor-specific value (see Section 3.3):
	// !- `confirmed`: indicates the event is definitely happening
	// !- `cancelled`: indicates the event has been cancelled
	// !- `tentative`: indicates the event may happen
	Status Status `json:"status,omitempty"`
}

type Task struct {
	Type TypeOfTask `json:"@type,omitempty"`

	Object

	// This is the date/time the task is due in the task's time zone.
	Due LocalDateTime `json:"due,omitzero"`

	// This the date/time the task should start in the task's time zone.
	Start LocalDateTime `json:"start,omitzero"`

	// This specifies the estimated positive duration of time the task takes to complete.
	EstimatedDuration Duration `json:"estimatedDuration,omitempty"`

	// This represents the percent completion of the task overall.
	//
	// The property value MUST be a positive integer between `0` and `100`.
	PercentComplete uint `json:"percentComplete,omitzero"`

	// This defines the progress of this task.
	//
	// If omitted, the default progress (Section 4.4) of a Task is defined as follows (in order of evaluation):
	// !- `completed`: if the progress property value of all participants is completed
	// !- `failed`: if at least one progress property value of a participant is failed
	// !- `in-process`: if at least one progress property value of a participant is in-process
	// !- `needs-action`: if none of the other criteria match
	//
	// If set, it MUST be one of the following values, another value registered in the IANA "JSCalendar Enum Values"
	// registry, or a vendor-specific value (see Section 3.3):
	// !- `needs-action`: indicates the task needs action
	// !- `in-process`: indicates the task is in process
	// !- `completed`: indicates the task is completed
	// !- `failed`: indicates the task failed
	// !- `cancelled`: indicates the task was cancelled
	Progress Progress `json:"progress,omitempty"`

	// This specifies the date/time the progress property of either the task overall (Section 5.2.5) or
	// a specific participant (Section 4.4.6) was last updated.
	//
	// If the task is recurring and has future instances, a client may want to keep track of the last progress
	// update timestamp of a specific task recurrence but leave other instances unchanged.
	//
	// One way to achieve this is by overriding the `progressUpdated` property in the task `recurrenceOverrides` property.
	//
	// However, this could produce a long list of timestamps for regularly recurring tasks.
	//
	// An alternative approach is to split the `Task` into a current, single instance of `Task` with this instance
	// progress update time and a future recurring instance.
	//
	// See also Section 4.1.3 on splitting.
	ProgressUpdated time.Time `json:"progressUpdated,omitzero"`
}

type GroupEntry interface {
	groupEntry()
}

func (e Event) groupEntry() {}

var _ GroupEntry = Event{}

func (t Task) groupEntry() {}

var _ GroupEntry = Task{}

type Group struct {
	Type TypeOfGroup `json:"@type,omitempty"`

	CommonObject

	// This is a collection of group members.
	//
	// Implementations MUST ignore entries of unknown type.
	Entries []GroupEntry `json:"entries"`

	// This is the source from which updated versions of this group may be retrieved.
	//
	// The value MUST be a URI.
	Source string `json:"source,omitempty"`
}

func (g *Group) UnmarshalJSON(b []byte) error {
	var typ struct {
		Entries []struct {
			Type string `json:"@type"`
		} `json:"entries"`
	}
	if err := json.Unmarshal(b, &typ); err != nil {
		return err
	}
	entries := make([]GroupEntry, len(typ.Entries))
	for i, entry := range typ.Entries {
		switch entry.Type {
		case string(EventType):
			entries[i] = new(Event)
		case string(TaskType):
			entries[i] = new(Task)
		default:
			return fmt.Errorf("unsupported '%T.type' @type: \"%v\"", entry, entry.Type)
		}
	}

	type tmp Group
	return json.Unmarshal(b, (*tmp)(g))
}

// mlr --csv --headerless-csv-output cut -f Token ./location-type-registry-1.csv |sort|perl -ne 'chomp; print "LocationTypeOption".ucfirst($_)." = LocationTypeOption(\"".$_."\")\n"'
