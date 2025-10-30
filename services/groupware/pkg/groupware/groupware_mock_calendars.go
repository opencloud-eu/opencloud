package groupware

import (
	"github.com/opencloud-eu/opencloud/pkg/jmap"
	"github.com/opencloud-eu/opencloud/pkg/jscalendar"
)

var C1 = jmap.Calendar{
	Id:                    "thoo5she",
	Name:                  "Personal Calendar",
	Description:           "Camina Drummer's Personal Calendar",
	Color:                 "purple",
	SortOrder:             1,
	IsSubscribed:          true,
	IsVisible:             true,
	IsDefault:             true,
	IncludeInAvailability: jmap.IncludeInAvailabilityAll,
	DefaultAlertsWithTime: map[string]jscalendar.Alert{
		"eing7doh": {
			Type: jscalendar.AlertType,
			Trigger: jscalendar.AbsoluteTrigger{
				Type: jscalendar.AbsoluteTriggerType,
				When: mustParseTime("2025-09-30T20:34:12Z"),
			},
		},
	},
	DefaultAlertsWithoutTime: map[string]jscalendar.Alert{
		"oayooy0u": {
			Type: jscalendar.AlertType,
			Trigger: jscalendar.OffsetTrigger{
				Type:       jscalendar.OffsetTriggerType,
				Offset:     "PT5M",
				RelativeTo: jscalendar.RelativeToStart,
			},
		},
	},
	TimeZone: "CEST",
	ShareWith: map[string]jmap.CalendarRights{
		"ahn0doo8": {
			MayReadFreeBusy:  true,
			MayReadItems:     true,
			MayWriteAll:      true,
			MayWriteOwn:      true,
			MayUpdatePrivate: true,
			MayRSVP:          false,
			MayAdmin:         false,
			MayDelete:        false,
		},
	},
	MyRights: &jmap.CalendarRights{
		MayReadFreeBusy:  true,
		MayReadItems:     true,
		MayWriteAll:      false,
		MayWriteOwn:      false,
		MayUpdatePrivate: true,
		MayRSVP:          true,
		MayAdmin:         false,
		MayDelete:        false,
	},
}

var AllCalendars = []jmap.Calendar{C1}

var E1 = jmap.CalendarEvent{
	Id: "ovei9oqu",
	CalendarIds: map[string]bool{
		C1.Id: true,
	},
	BaseEventId: "ahtah9qu",
	IsDraft:     true,
	IsOrigin:    true,
	UtcStart:    jmap.UTCDate{Time: mustParseTime("2025-10-01T00:00:00Z")},
	UtcEnd:      jmap.UTCDate{Time: mustParseTime("2025-10-07T00:00:00Z")},
	Event: jscalendar.Event{
		Type:     jscalendar.EventType,
		Start:    jscalendar.LocalDateTime("2025-09-30T12:00:00"),
		Duration: "PT30M",
		Status:   jscalendar.StatusConfirmed,
		Object: jscalendar.Object{
			CommonObject: jscalendar.CommonObject{
				Uid:                    "9a7ab91a-edca-4988-886f-25e00743430d",
				ProdId:                 "Mock 0.0",
				Created:                mustParseTime("2025-09-29T16:17:18Z"),
				Updated:                mustParseTime("2025-09-29T16:17:18Z"),
				Title:                  "Meeting of the Minds",
				Description:            "Internal meeting about the grand strategy for the future",
				DescriptionContentType: "text/plain",
				Links: map[string]jscalendar.Link{
					"cai0thoh": {
						Type:        jscalendar.LinkType,
						Href:        "https://example.com/9a7ab91a-edca-4988-886f-25e00743430d",
						Rel:         jscalendar.RelAbout,
						ContentType: "text/html",
					},
				},
				Locale: "en-US",
				Keywords: map[string]bool{
					"meeting": true,
					"secret":  true,
				},
				Categories: map[string]bool{
					"secret":   true,
					"internal": true,
				},
				Color: "purple",
				TimeZones: map[string]jscalendar.TimeZone{
					"airee8ai": {
						Type: jscalendar.TimeZoneType,
						TzId: "CEST",
					},
				},
			},
			RelatedTo:       map[string]jscalendar.Relation{},
			Sequence:        0,
			ShowWithoutTime: false,
			Locations: map[string]jscalendar.Location{
				"ux1uokie": {
					Type:        jscalendar.LocationType,
					Name:        "office",
					Description: "Office meeting room upstairs",
					LocationTypes: map[jscalendar.LocationTypeOption]bool{
						jscalendar.LocationTypeOptionOffice: true,
					},
					RelativeTo:  jscalendar.LocationRelationStart,
					TimeZone:    "CEST",
					Coordinates: "geo:52.5334956,13.4079872",
					Links: map[string]jscalendar.Link{
						"eefe2pax": {
							Type: jscalendar.LinkType,
							Href: "https://example.com/office",
						},
					},
				},
			},
			VirtualLocations: map[string]jscalendar.VirtualLocation{
				"em4eal0o": {
					Type:        jscalendar.VirtualLocationType,
					Name:        "opentalk",
					Description: "The opentalk Conference Room",
					Uri:         "https://meet.opentalk.eu",
					Features: map[jscalendar.VirtualLocationFeature]bool{
						jscalendar.VirtualLocationFeatureAudio:  true,
						jscalendar.VirtualLocationFeatureChat:   true,
						jscalendar.VirtualLocationFeatureVideo:  true,
						jscalendar.VirtualLocationFeatureScreen: true,
					},
				},
			},
			RecurrenceRule: &jscalendar.RecurrenceRule{
				Type:           jscalendar.RecurrenceRuleType,
				Frequency:      jscalendar.FrequencyWeekly,
				Interval:       1,
				Rscale:         jscalendar.RscaleIso8601,
				Skip:           jscalendar.SkipOmit,
				FirstDayOfWeek: jscalendar.DayOfWeekMonday,
				Count:          4,
			},
			FreeBusyStatus: jscalendar.FreeBusyStatusBusy,
			Privacy:        jscalendar.PrivacyPublic,
			ReplyTo: map[jscalendar.ReplyMethod]string{
				jscalendar.ReplyMethodImip: "mailto:organizer@example.com",
			},
			SentBy: "organizer@example.com",
			Participants: map[string]jscalendar.Participant{
				"eegh7uph": {
					Type:        jscalendar.ParticipantType,
					Name:        "Anderson Dawes",
					Email:       "adawes@opa.org",
					Description: "Called the meeting",
					SendTo: map[jscalendar.SendToMethod]string{
						jscalendar.SendToMethodImip: "mailto:adawes@opa.org",
					},
					Kind: jscalendar.ParticipantKindIndividual,
					Roles: map[jscalendar.Role]bool{
						jscalendar.RoleAttendee: true,
						jscalendar.RoleChair:    true,
						jscalendar.RoleOwner:    true,
					},
					LocationId:           "ux1uokie",
					Language:             "en-GB",
					ParticipationStatus:  jscalendar.ParticipationStatusAccepted,
					ParticipationComment: "I'll be there for sure",
					ExpectReply:          true,
					ScheduleAgent:        jscalendar.ScheduleAgentServer,
					ScheduleSequence:     1,
					ScheduleStatus:       []string{"1.0"},
					ScheduleUpdated:      mustParseTime("2025-10-01T11:59:12Z"),
					SentBy:               "adawes@opa.org",
					InvitedBy:            "eegh7uph",
					Links: map[string]jscalendar.Link{
						"ieni5eiw": {
							Type:        jscalendar.LinkType,
							Href:        "https://static.wikia.nocookie.net/expanse/images/1/1e/OPA_leader.png/revision/latest?cb=20250121103410",
							ContentType: "image/png",
							Rel:         jscalendar.RelIcon,
							Size:        192812,
							Display:     jscalendar.DisplayBadge,
							Title:       "Anderson Dawes' photo",
						},
					},
					ScheduleId: "mailto:adawes@opa.org",
				},
				"xeikie9p": {
					Type:        jscalendar.ParticipantType,
					Name:        "Klaes Ashford",
					Email:       "ashford@opa.org",
					Description: "As the first officer on the Behemoth",
					SendTo: map[jscalendar.SendToMethod]string{
						jscalendar.SendToMethodImip:  "mailto:ashford@opa.org",
						jscalendar.SendToMethodOther: "https://behemoth.example.com/ping/@ashford",
					},
					Kind: jscalendar.ParticipantKindIndividual,
					Roles: map[jscalendar.Role]bool{
						jscalendar.RoleAttendee: true,
					},
					LocationId:          "em4eal0o",
					Language:            "en-GB",
					ParticipationStatus: jscalendar.ParticipationStatusNeedsAction,
					ExpectReply:         true,
					ScheduleAgent:       jscalendar.ScheduleAgentServer,
					ScheduleSequence:    0,
					SentBy:              "adawes@opa.org",
					InvitedBy:           "eegh7uph",
					Links: map[string]jscalendar.Link{
						"oifooj6g": {
							Type:        jscalendar.LinkType,
							Href:        "https://static.wikia.nocookie.net/expanse/images/0/02/Klaes_Ashford_-_Expanse_season_4_promotional_2.png/revision/latest?cb=20191206012007",
							ContentType: "image/png",
							Rel:         jscalendar.RelIcon,
							Size:        201291,
							Display:     jscalendar.DisplayBadge,
							Title:       "Ashford on Medina Station",
						},
					},
					ScheduleId: "mailto:ashford@opa.org",
				},
			},
			Alerts: map[string]jscalendar.Alert{
				"ahqu4xi0": {
					Type: jscalendar.AlertType,
					Trigger: jscalendar.OffsetTrigger{
						Type:       jscalendar.OffsetTriggerType,
						Offset:     "PT-5M",
						RelativeTo: jscalendar.RelativeToStart,
					},
				},
			},
			TimeZone:        "UTC",
			MayInviteSelf:   true,
			MayInviteOthers: true,
			HideAttendees:   false,
		},
	},
}

var EventsMapByCalendarId = map[string][]jmap.CalendarEvent{
	C1.Id: {
		E1,
	},
}
