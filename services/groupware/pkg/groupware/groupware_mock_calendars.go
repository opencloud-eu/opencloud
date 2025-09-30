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
				When: MustParse("2025-09-30T20:34:12Z"),
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
	Event: jscalendar.Event{
		Type:     jscalendar.EventType,
		Start:    jscalendar.LocalDateTime{Time: MustParse("2025-09-30T12:00:00Z")},
		Duration: "PT30M",
		Status:   jscalendar.StatusConfirmed,
		Object: jscalendar.Object{
			CommonObject: jscalendar.CommonObject{
				Uid:                    "9a7ab91a-edca-4988-886f-25e00743430d",
				ProdId:                 "Mock 0.0",
				Created:                MustParse("2025-09-29T16:17:18Z"),
				Updated:                MustParse("2025-09-29T16:17:18Z"),
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
			Method:          jscalendar.MethodAdd,
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
			// TODO more properties, a lot more properties
		},
	},
}

var EventsMapByCalendarId = map[string][]jmap.CalendarEvent{
	C1.Id: {
		E1,
	},
}
