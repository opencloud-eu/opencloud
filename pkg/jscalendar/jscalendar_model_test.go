package jscalendar

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func jsoneq[X any](t *testing.T, expected string, object X) {
	data, err := json.MarshalIndent(object, "", "")
	require.NoError(t, err)
	require.JSONEq(t, expected, string(data))

	var rec X
	err = json.Unmarshal(data, &rec)
	require.NoError(t, err)
	require.Equal(t, object, rec)
}

func TestLocalDateTime(t *testing.T) {
	ts, err := time.Parse(time.RFC3339, "2025-09-25T18:26:14+02:00")
	require.NoError(t, err)

	ldt := &LocalDateTime{ts}

	str, err := json.MarshalIndent(ldt, "", "")
	require.NoError(t, err)

	require.Equal(t, "\"2025-09-25T16:26:14\"", string(str))
}

func TestLocalDateTimeUnmarshalling(t *testing.T) {
	ts, err := time.Parse(RFC3339Local, "2025-09-25T18:26:14")
	require.NoError(t, err)
	u := ts.UTC()

	var result LocalDateTime
	err = json.Unmarshal([]byte("\"2025-09-25T18:26:14Z\""), &result)
	require.NoError(t, err)

	require.Equal(t, result, LocalDateTime{u})
}

func TestRelation(t *testing.T) {
	jsoneq(t, `{
		"@type": "Relation",
		"relation": {
			"first": true,
			"parent": true
		}
	}`, Relation{
		Type: RelationType,
		Relation: map[Relationship]bool{
			RelationshipFirst:  true,
			RelationshipParent: true,
		},
	})
}

func TestLink(t *testing.T) {
	jsoneq(t, `{
		"@type": "Link",
		"href": "https://opencloud.eu.example.com/f72ae875-40be-48a4-84ff-aea9aed3e085.png",
		"cid": "c1",
		"contentType": "image/png",
		"size": 128912,
		"rel": "icon",
		"display": "thumbnail",
		"title": "the logo"
	}`, Link{
		Type:        LinkType,
		Href:        "https://opencloud.eu.example.com/f72ae875-40be-48a4-84ff-aea9aed3e085.png",
		Cid:         "c1",
		ContentType: "image/png",
		Size:        128912,
		Rel:         RelIcon,
		Display:     DisplayThumbnail,
		Title:       "the logo",
	})
}

func TestLocation(t *testing.T) {
	jsoneq(t, `{
		"@type": "Location",
		"name": "The Eiffel Tower",
		"description": "The big iron tower in the middle of Paris, can't miss it.",
		"locationTypes": {
			"landmark-address": true,
			"industrial": true
		},
		"relativeTo": "start",
		"timeZone": "Europe/Paris",
		"coordinates": "geo:48.8559324,2.2932441",
		"links": {
			"l1": {
				"@type": "Link",
				"href": "https://upload.wikimedia.org/wikipedia/commons/f/fd/Eiffel_blue.PNG",
				"cid": "cl1",
				"contentType": "image/png",
				"size": 12345,
				"rel": "icon",
				"display": "A blue Eiffel tower",
				"title": "Blue Eiffel Tower"
			}
		}
	}`, Location{
		Type:        LocationType,
		Name:        "The Eiffel Tower",
		Description: "The big iron tower in the middle of Paris, can't miss it.",
		LocationTypes: map[LocationTypeOption]bool{
			LocationTypeOptionLandmarkAddress: true,
			LocationTypeOptionIndustrial:      true,
		},
		RelativeTo:  LocationRelationStart,
		TimeZone:    "Europe/Paris",
		Coordinates: "geo:48.8559324,2.2932441",
		Links: map[string]Link{
			"l1": {
				Type:        LinkType,
				Href:        "https://upload.wikimedia.org/wikipedia/commons/f/fd/Eiffel_blue.PNG",
				Cid:         "cl1",
				ContentType: "image/png",
				Size:        12345,
				Rel:         RelIcon,
				Display:     "A blue Eiffel tower",
				Title:       "Blue Eiffel Tower",
			},
		},
	})
}

func TestVirtualLocation(t *testing.T) {
	jsoneq(t, `{
		"@type": "VirtualLocation",
		"name": "OpenTalk",
		"description": "The best videoconferencing.",
		"uri": "https://opentalk.eu",
		"features": {
			"video": true,
			"screen": true,
			"audio": true
		}
	}`, VirtualLocation{
		Type:        VirtualLocationType,
		Name:        "OpenTalk",
		Description: "The best videoconferencing.",
		Uri:         "https://opentalk.eu",
		Features: map[VirtualLocationFeature]bool{
			VirtualLocationFeatureVideo:  true,
			VirtualLocationFeatureScreen: true,
			VirtualLocationFeatureAudio:  true,
		},
	})
}

func TestNDay(t *testing.T) {
	jsoneq(t, `{
		"@type": "NDay",
		"day": "fr",
		"nthOfPeriod": -1
	}`, NDay{
		Type:        NDayType,
		Day:         DayOfWeekFriday,
		NthOfPeriod: -1,
	})
}

func TestRecurrenceRule(t *testing.T) {
	ts, err := time.Parse(time.RFC3339, "2025-09-25T18:26:14+02:00")
	require.NoError(t, err)
	ts = ts.UTC()

	jsoneq(t, `{
		"@type": "RecurrenceRule",
		"frequency": "daily",
		"interval": 1,
		"rscale": "iso8601",
		"skip": "forward",
		"firstDayOfWeek": "mo",
		"byDay": [
			{"@type": "NDay", "day": "mo", "nthOfPeriod": -1},
			{"@type": "NDay", "day": "tu"},
			{"day": "we"}
		],
		"byMonthDay": [1, 10, 31],
		"byMonth": ["1", "31L"],
		"byYearDay": [-1, 366],
		"byWeekNo": [-53, 53],
		"byHour": [0, 23],
		"byMinute": [0, 59],
		"bySecond": [0, 39],
		"bySetPosition": [-3, 3],
		"count": 2,
		"until": "2025-09-25T16:26:14Z"
	}`, RecurrenceRule{
		Type:           RecurrenceRuleType,
		Frequency:      FrequencyDaily,
		Interval:       1,
		Rscale:         RscaleIso8601,
		Skip:           SkipForward,
		FirstDayOfWeek: DayOfWeekMonday,
		ByDay: []NDay{
			{
				Type:        NDayType,
				Day:         DayOfWeekMonday,
				NthOfPeriod: -1,
			},
			{
				Type: NDayType,
				Day:  DayOfWeekTuesday,
			},
			{
				Day:         DayOfWeekWednesday,
				NthOfPeriod: 0,
			},
		},
		ByMonthDay:    []int{1, 10, 31},
		ByMonth:       []string{"1", "31L"},
		ByYearDay:     []int{-1, 366},
		ByWeekNo:      []int{-53, 53},
		ByHour:        []uint{0, 23},
		ByMinute:      []uint{0, 59},
		BySecond:      []uint{0, 39},
		BySetPosition: []int{-3, 3},
		Count:         2,
		Until:         &LocalDateTime{ts},
	})
}

func TestParticipant(t *testing.T) {
	ts, err := time.Parse(time.RFC3339, "2025-09-25T18:26:14+02:00")
	require.NoError(t, err)
	ts = ts.UTC()

	ts2, err := time.Parse(time.RFC3339, "2025-09-29T14:32:19+02:00")
	require.NoError(t, err)
	ts2 = ts2.UTC()

	jsoneq(t, `{
		"@type": "Participant",
		"name": "Camina Drummer",
		"email": "camina@opa.org",
		"description": "Camina Drummer is a Belter serving as the current President of the Transport Union.",
		"sendTo": {
			"imip": "mailto:cdrummer@opa.org",
			"other": "https://opa.org/ping/camina"
		},
		"kind": "individual",
		"roles": {
			"attendee": true,
			"owner": true,
			"chair": true
		},
		"locationId": "98faaa01-b6db-4ddb-9574-e28ab83104e6",
		"language": "en-JM",
		"participationStatus": "accepted",
		"participationComment": "always there",
		"expectReply": true,
		"scheduleAgent": "server",
		"scheduleForceSend": true,
		"scheduleSequence": 3,
		"scheduleStatus": [
			"3.1",
			"2.0"
		],
		"scheduleUpdated": "2025-09-25T16:26:14Z",
		"sentBy": "adawes@opa.org",
		"invitedBy": "346be402-c340-4f3f-ac51-e4aa9955af4f",
		"delegatedTo": {
			"93230b90-70c6-4027-b2c1-3629877bfea5": true,
			"f5fae398-cfa3-4873-bbc7-0ca9d51de5b0": true
		},
		"delegatedFrom": {
			"a9c1c1a1-fecf-4214-a803-1ee209e2dbec": true
		},
		"memberOf": {
			"0f41473b-0edd-494d-b346-8d039009a2a5": true
		},
		"links":{
			"l1": {
				"@type": "Link",
				"href": "https://opa.org/opa.png",
				"cid": "c1",
				"contentType": "image/png",
				"size": 182912,
				"rel": "icon",
				"display": "Logo",
				"title": "OPA"
			}
		},
		"progress": "in-process",
		"progressUpdated": "2025-09-29T12:32:19Z",
		"percentComplete": 42
	}`, Participant{
		Type:        ParticipantType,
		Name:        "Camina Drummer",
		Email:       "camina@opa.org",
		Description: "Camina Drummer is a Belter serving as the current President of the Transport Union.",
		SendTo: map[SendToMethod]string{
			SendToMethodImip:  "mailto:cdrummer@opa.org",
			SendToMethodOther: "https://opa.org/ping/camina",
		},
		Kind: ParticipantKindIndividual,
		Roles: map[Role]bool{
			RoleAttendee: true,
			RoleOwner:    true,
			RoleChair:    true,
		},
		LocationId:           "98faaa01-b6db-4ddb-9574-e28ab83104e6",
		Language:             "en-JM",
		ParticipationStatus:  ParticipationStatusAccepted,
		ParticipationComment: "always there",
		ExpectReply:          true,
		ScheduleAgent:        ScheduleAgentServer,
		ScheduleForceSend:    true,
		ScheduleSequence:     3,
		ScheduleStatus: []string{
			"3.1",
			"2.0",
		},
		ScheduleUpdated: ts,
		SentBy:          "adawes@opa.org",
		InvitedBy:       "346be402-c340-4f3f-ac51-e4aa9955af4f",
		DelegatedTo: map[string]bool{
			"93230b90-70c6-4027-b2c1-3629877bfea5": true,
			"f5fae398-cfa3-4873-bbc7-0ca9d51de5b0": true,
		},
		DelegatedFrom: map[string]bool{
			"a9c1c1a1-fecf-4214-a803-1ee209e2dbec": true,
		},
		MemberOf: map[string]bool{
			"0f41473b-0edd-494d-b346-8d039009a2a5": true,
		},
		Links: map[string]Link{
			"l1": {
				Type:        LinkType,
				Href:        "https://opa.org/opa.png",
				Cid:         "c1",
				ContentType: "image/png",
				Size:        182912,
				Rel:         RelIcon,
				Display:     "Logo",
				Title:       "OPA",
			},
		},
		Progress:        ProgressInProcess,
		ProgressUpdated: ts2,
		PercentComplete: 42,
	})
}

func TestAlertWithAbsoluteTrigger(t *testing.T) {
	ts, err := time.Parse(time.RFC3339, "2025-09-25T18:26:14+02:00")
	require.NoError(t, err)
	ts = ts.UTC()

	jsoneq(t, `{
		"@type": "Alert",
		"trigger": {
			"@type": "AbsoluteTrigger",
			"when": "2025-09-25T16:26:14Z"
		},
		"acknowledged": "2025-09-25T16:26:14Z",
		"relatedTo": {
			"a2e729eb-7d9c-4ea7-8514-93d2590ef0a2": {
				"@type": "Relation",
				"relation": {
					"first": true
				}
			}
		},
		"action": "email"
	}`, Alert{
		Type: AlertType,
		Trigger: &AbsoluteTrigger{
			Type: AbsoluteTriggerType,
			When: ts,
		},
		Acknowledged: ts,
		RelatedTo: map[string]Relation{
			"a2e729eb-7d9c-4ea7-8514-93d2590ef0a2": {
				Type: RelationType,
				Relation: map[Relationship]bool{
					RelationshipFirst: true,
				},
			},
		},
		Action: AlertActionEmail,
	})
}

func TestAlertWithOffsetTrigger(t *testing.T) {
	ts, err := time.Parse(time.RFC3339, "2025-09-25T18:26:14+02:00")
	require.NoError(t, err)
	ts = ts.UTC()

	jsoneq(t, `{
		"@type": "Alert",
		"trigger": {
			"@type": "OffsetTrigger",
			"offset": "-PT5M",
			"relativeTo": "end"
		},
		"acknowledged": "2025-09-25T16:26:14Z",
		"relatedTo": {
			"a2e729eb-7d9c-4ea7-8514-93d2590ef0a2": {
				"@type": "Relation",
				"relation": {
					"first": true
				}
			}
		},
		"action": "email"		
	}`, Alert{
		Type: AlertType,
		Trigger: &OffsetTrigger{
			Type:       OffsetTriggerType,
			Offset:     "-PT5M",
			RelativeTo: RelativeToEnd,
		},
		Acknowledged: ts,
		RelatedTo: map[string]Relation{
			"a2e729eb-7d9c-4ea7-8514-93d2590ef0a2": {
				Type: RelationType,
				Relation: map[Relationship]bool{
					RelationshipFirst: true,
				},
			},
		},
		Action: AlertActionEmail,
	})
}

func TestAlertWithUnknownTrigger(t *testing.T) {
	ts, err := time.Parse(time.RFC3339, "2025-09-25T18:26:14+02:00")
	require.NoError(t, err)
	ts = ts.UTC()

	jsoneq(t, `{
		"@type": "Alert",
		"trigger": {
			"@type": "XYZTRIGGER",
			"abc": 123,
			"xyz": "zzz"
		},
		"acknowledged": "2025-09-25T16:26:14Z",
		"relatedTo": {
			"a2e729eb-7d9c-4ea7-8514-93d2590ef0a2": {
				"@type": "Relation",
				"relation": {
					"first": true
				}
			}
		},
		"action": "email"		
	}`, Alert{
		Type: AlertType,
		Trigger: &UnknownTrigger{
			"@type": "XYZTRIGGER",
			"abc":   123.0,
			"xyz":   "zzz",
		},
		Acknowledged: ts,
		RelatedTo: map[string]Relation{
			"a2e729eb-7d9c-4ea7-8514-93d2590ef0a2": {
				Type: RelationType,
				Relation: map[Relationship]bool{
					RelationshipFirst: true,
				},
			},
		},
		Action: AlertActionEmail,
	})
}

func TestTimeZoneRule(t *testing.T) {
	ts, err := time.Parse(time.RFC3339, "2025-09-25T18:26:14+02:00")
	require.NoError(t, err)
	ts = ts.UTC()

	l1 := LocalDateTime{ts}

	jsoneq(t, `{
		"@type": "TimeZoneRule",
		"start": "2025-09-25T16:26:14Z",
		"offsetFrom": "-0200",
		"offsetTo": "+0200",
		"recurrenceRules": [
			{
				"@type": "RecurrenceRule",
				"frequency": "weekly",
				"interval": 2,
				"rscale": "iso8601",
				"skip": "omit",
				"firstDayOfWeek": "mo",
				"byDay": [
					{
						"@type": "NDay",
						"day": "fr"
					}
				],
				"byHour": [14],
				"byMinute": [0],
				"count": 4
			}
		],
		"recurrenceOverrides": {
			"2025-09-25T16:26:14Z": {}
		},
		"names": {
			"CEST": true
		},
		"comments": ["this is a comment"]
	}`, TimeZoneRule{
		Type:       TimeZoneRuleType,
		Start:      LocalDateTime{ts},
		OffsetFrom: "-0200",
		OffsetTo:   "+0200",
		RecurrenceRules: []RecurrenceRule{
			{
				Type:           RecurrenceRuleType,
				Frequency:      FrequencyWeekly,
				Interval:       2,
				Rscale:         RscaleIso8601,
				Skip:           SkipOmit,
				FirstDayOfWeek: DayOfWeekMonday,
				ByDay: []NDay{
					{
						Type: NDayType,
						Day:  DayOfWeekFriday,
					},
				},
				ByHour: []uint{
					14,
				},
				ByMinute: []uint{
					0,
				},
				Count: 4,
			},
		},
		RecurrenceOverrides: map[LocalDateTime]PatchObject{
			l1: {},
		},
		Names: map[string]bool{
			"CEST": true,
		},
		Comments: []string{
			"this is a comment",
		},
	})
}

func TestTimeZone(t *testing.T) {
	ts, err := time.Parse(time.RFC3339, "2025-09-25T18:26:14+02:00")
	require.NoError(t, err)
	ts = ts.UTC()

	jsoneq(t, `{
		"@type": "TimeZone",
		"tzId": "cest",
		"updated": "2025-09-25T16:26:14Z",
		"url": "https://timezones.net/cest",
		"validUntil": "2025-09-25T16:26:14Z",
		"aliases": {
			"cet": true
		},
		"standard": [{
			"@type": "TimeZoneRule",
			"start": "2025-09-25T16:26:14Z",
			"offsetFrom": "-0200",
			"offsetTo": "+1245"
		}],
		"daylight": [{
			"@type": "TimeZoneRule",
			"start": "2025-09-25T16:26:14Z",
			"offsetFrom": "-0200",
			"offsetTo": "+1245"
		}]
	}`, TimeZone{
		Type:       TimeZoneType,
		TzId:       "cest",
		Updated:    ts,
		Url:        "https://timezones.net/cest",
		ValidUntil: ts,
		Aliases: map[string]bool{
			"cet": true,
		},
		Standard: []TimeZoneRule{
			{
				Type:       TimeZoneRuleType,
				Start:      LocalDateTime{ts},
				OffsetFrom: "-0200",
				OffsetTo:   "+1245",
			},
		},
		Daylight: []TimeZoneRule{
			{
				Type:       TimeZoneRuleType,
				Start:      LocalDateTime{ts},
				OffsetFrom: "-0200",
				OffsetTo:   "+1245",
			},
		},
	})
}

func TestEvent(t *testing.T) {
	ts1, err := time.Parse(time.RFC3339, "2025-09-25T18:26:14+02:00")
	require.NoError(t, err)
	ts1 = ts1.UTC()

	ts2, err := time.Parse(time.RFC3339, "2025-09-29T15:53:01+02:00")
	require.NoError(t, err)
	ts2 = ts2.UTC()

	jsoneq(t, `{
		"@type": "Event",
		"start": "2025-09-25T16:26:14Z",
		"duration": "PT10M",
		"status": "confirmed",
		"uid": "b422cfec-f7b4-4e04-8ec6-b794007f63f1",
		"prodId": "OpenCloud 1.0",
		"created": "2025-09-25T16:26:14Z",
		"updated": "2025-09-29T13:53:01Z",
		"title": "End of year party",
		"description": "It's the party at the end of the year.",
		"descriptionContentType": "text/plain",
		"links": {
			"l1": {
				"@type": "Link",
				"href": "https://opencloud.eu/eoy-party/2025",
				"contentType": "text/html",
				"rel": "about"
			}
		},
		"locale": "en-GB",
		"keywords": {
			"k1": true
		},
		"categories": {
			"cat": true
		},
		"color": "oil",
		"timeZones": {
			"cest": {
				"@type": "TimeZone",
				"tzId": "cest"
			}
		},
		"relatedTo": {
			"a": {
				"@type": "Relation",
				"relation": {
					"next": true
				}
			}
		},		
		"sequence": 3,
		"method": "refresh",
		"showWithoutTime": true,
		"locations": {
			"loc1": {
				"@type": "Location",
				"name": "Steel Cactus Mexican Grill",
				"description": "The Steel Cactus Mexican Grill used to be on the Hecate Navy Base. The place closed down and is now a take-out restaurant that sells to-go cups of Thai food",
				"locationTypes": {
					"bar": true
				},
				"relativeTo": "start",
				"timeZone": "cest",
				"coordinates": "geo:16.7685657,-4.8629852",
				"links": {
					"l1": {
						"@type": "Link",
						"href": "https://mars.gov/bars/steelcactus",
						"rel": "about"
					}
				}
			}
		}
	}`, Event{
		Type:     EventType,
		Start:    LocalDateTime{ts1},
		Duration: "PT10M",
		Status:   "confirmed",
		Object: Object{
			CommonObject: CommonObject{
				Uid:                    "b422cfec-f7b4-4e04-8ec6-b794007f63f1",
				ProdId:                 "OpenCloud 1.0",
				Created:                ts1,
				Updated:                ts2,
				Title:                  "End of year party",
				Description:            "It's the party at the end of the year.",
				DescriptionContentType: "text/plain",
				Links: map[string]Link{
					"l1": {
						Type:        LinkType,
						Href:        "https://opencloud.eu/eoy-party/2025",
						ContentType: "text/html",
						Rel:         RelAbout,
					},
				},
				Locale: "en-GB",
				Keywords: map[string]bool{
					"k1": true,
				},
				Categories: map[string]bool{
					"cat": true,
				},
				Color: "oil",
				TimeZones: map[string]TimeZone{
					"cest": {
						Type: TimeZoneType,
						TzId: "cest",
					},
				},
			},
			RelatedTo: map[string]Relation{
				"a": {
					Type: RelationType,
					Relation: map[Relationship]bool{
						RelationshipNext: true,
					},
				},
			},
			Sequence:        3,
			Method:          MethodRefresh,
			ShowWithoutTime: true,
			Locations: map[string]Location{
				"loc1": {
					Type:        LocationType,
					Name:        "Steel Cactus Mexican Grill",
					Description: "The Steel Cactus Mexican Grill used to be on the Hecate Navy Base. The place closed down and is now a take-out restaurant that sells to-go cups of Thai food",
					LocationTypes: map[LocationTypeOption]bool{
						LocationTypeOptionBar: true,
					},
					RelativeTo:  LocationRelationStart,
					TimeZone:    "cest",
					Coordinates: "geo:16.7685657,-4.8629852",
					Links: map[string]Link{
						"l1": {
							Type: LinkType,
							Href: "https://mars.gov/bars/steelcactus",
							Rel:  RelAbout,
						},
					},
				},
			},
		},
	})
}
