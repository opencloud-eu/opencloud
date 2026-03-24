package jmap

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/stretchr/testify/require"

	"github.com/opencloud-eu/opencloud/pkg/jscalendar"
	"github.com/opencloud-eu/opencloud/pkg/structs"
)

// fields that are currently unsupported in Stalwart
const (
	EnableEventMayInviteFields              = false
	EnableEventParticipantDescriptionFields = false
)

func TestEvents(t *testing.T) {
	if skip(t) {
		return
	}

	count := uint(20 + rand.Intn(30))

	require := require.New(t)

	s, err := newStalwartTest(t)
	require.NoError(err)
	defer s.Close()

	user := pickUser()
	session := s.Session(user.name)

	accountId, calendarId, expectedEventsById, boxes, err := s.fillEvents(t, count, session, user)
	require.NoError(err)
	require.NotEmpty(accountId)
	require.NotEmpty(calendarId)

	filter := CalendarEventFilterCondition{
		InCalendar: calendarId,
	}
	sortBy := []CalendarEventComparator{
		{Property: CalendarEventPropertyStart, IsAscending: true},
	}

	contactsByAccount, _, _, _, err := s.client.QueryCalendarEvents([]string{accountId}, session, t.Context(), s.logger, "", filter, sortBy, 0, 0)
	require.NoError(err)

	require.Len(contactsByAccount, 1)
	require.Contains(contactsByAccount, accountId)
	contacts := contactsByAccount[accountId]
	require.Len(contacts, int(count))

	for _, actual := range contacts {
		expected, ok := expectedEventsById[actual.Id]
		require.True(ok, "failed to find created contact by its id")
		matchEvent(t, actual, expected)
	}

	exceptions := []string{}
	if !EnableEventMayInviteFields {
		exceptions = append(exceptions, "mayInvite")
	}
	allBoxesAreTicked(t, boxes, exceptions...)
}

func matchEvent(t *testing.T, actual CalendarEvent, expected CalendarEvent) {
	//require.Equal(t, expected, actual)
	deepEqual(t, expected, actual)
}

type EventsBoxes struct {
	categories bool
	keywords   bool
	mayInvite  bool
}

func (s *StalwartTest) fillEvents( //NOSONAR
	t *testing.T,
	count uint,
	session *Session,
	user User,
) (string, string, map[string]CalendarEvent, EventsBoxes, error) {
	require := require.New(t)
	c, err := NewTestJmapClient(session, user.name, user.password, true, true)
	require.NoError(err)
	defer c.Close()

	boxes := EventsBoxes{}

	printer := func(s string) { log.Println(s) }

	accountId := c.session.PrimaryAccounts.Calendars
	require.NotEmpty(accountId, "no primary account for calendars in session")

	calendarId := ""
	{
		calendarsById, err := c.objectsById(accountId, CalendarType, JmapCalendars)
		require.NoError(err)

		for id, calendar := range calendarsById {
			if isDefault, ok := calendar["isDefault"]; ok {
				if isDefault.(bool) {
					calendarId = id
					break
				}
			}
		}
	}
	require.NotEmpty(calendarId)

	filled := map[string]CalendarEvent{}
	for i := range count {
		uid := gofakeit.UUID()

		isDraft := false
		mainLocationId := ""
		locationIds := []string{}
		locationMaps := map[string]map[string]any{}
		locationObjs := map[string]jscalendar.Location{}
		{
			n := 1
			if i%4 == 0 {
				n++
			}
			for range n {
				locationId, locationMap, locationObj := pickLocation()
				locationMaps[locationId] = locationMap
				locationObjs[locationId] = locationObj
				locationIds = append(locationIds, locationId)
				if n > 0 && mainLocationId == "" {
					mainLocationId = locationId
				}
			}
		}
		virtualLocationId, virtualLocationMap, virtualLocationObj := pickVirtualLocation()
		participantMaps, participantObjs, organizerEmail := createParticipants(uid, locationIds, []string{virtualLocationId})
		duration := pickRandom("PT30M", "PT45M", "PT1H", "PT90M")
		tz := pickRandom(timezones...)
		daysDiff := rand.Intn(31) - 15
		t := time.Now().Add(time.Duration(daysDiff) * time.Hour * 24)
		h := pickRandom(9, 10, 11, 14, 15, 16, 18)
		m := pickRandom(0, 30)
		t = time.Date(t.Year(), t.Month(), t.Day(), h, m, 0, 0, t.Location())
		start := strings.ReplaceAll(t.Format(time.DateTime), " ", "T")
		title := gofakeit.Sentence(1)
		description := gofakeit.Paragraph(1+rand.Intn(3), 1+rand.Intn(4), 1+rand.Intn(32), "\n")

		descriptionFormat := pickRandom("text/plain", "text/html") //NOSONAR
		if descriptionFormat == "text/html" {
			description = toHtml(description)
		}
		status := pickRandom(jscalendar.Statuses...)
		freeBusy := pickRandom(jscalendar.FreeBusyStatuses...)
		privacy := pickRandom(jscalendar.Privacies...)
		color := pickRandom(basicColors...)
		locale := pickLocale()
		keywords := pickKeywords()
		categories := pickCategories()

		sequence := 0

		alertId := id()
		alertOffset := pickRandom("-PT5M", "-PT10M", "-PT15M")

		event := map[string]any{
			"@type":                  "Event",
			"calendarIds":            toBoolMapS(calendarId),
			"isDraft":                isDraft,
			"start":                  start,
			"duration":               duration,
			"status":                 string(status),
			"uid":                    uid,
			"prodId":                 productName,
			"title":                  title,
			"description":            description,
			"descriptionContentType": descriptionFormat,
			"locale":                 locale,
			"color":                  color,
			"sequence":               sequence,
			"showWithoutTime":        false,
			"freeBusyStatus":         string(freeBusy),
			"privacy":                string(privacy),
			"sentBy":                 organizerEmail,
			"participants":           participantMaps,
			"timeZone":               tz,
			"hideAttendees":          false,
			"replyTo": map[string]string{
				"imip": "mailto:" + organizerEmail, //NOSONAR
			},
			"locations": locationMaps,
			"virtualLocations": map[string]any{
				virtualLocationId: virtualLocationMap,
			},
			"alerts": map[string]map[string]any{
				alertId: {
					"@type": "Alert",
					"trigger": map[string]any{
						"@type":      "OffsetTrigger",
						"offset":     alertOffset,
						"relativeTo": "start",
					},
				},
			},
		}

		obj := CalendarEvent{
			Id:          "",
			CalendarIds: toBoolMapS(calendarId),
			IsDraft:     isDraft,
			IsOrigin:    true,
			Event: jscalendar.Event{
				Type:     jscalendar.EventType,
				Start:    jscalendar.LocalDateTime(start),
				Duration: jscalendar.Duration(duration),
				Status:   status,
				Object: jscalendar.Object{
					CommonObject: jscalendar.CommonObject{
						Uid:                    uid,
						ProdId:                 productName,
						Title:                  title,
						Description:            description,
						DescriptionContentType: descriptionFormat,
						Locale:                 locale,
						Color:                  color,
					},
					Sequence:        uint(sequence),
					ShowWithoutTime: false,
					FreeBusyStatus:  freeBusy,
					Privacy:         privacy,
					SentBy:          organizerEmail,
					Participants:    participantObjs,
					TimeZone:        tz,
					HideAttendees:   false,
					ReplyTo: map[jscalendar.ReplyMethod]string{
						jscalendar.ReplyMethodImip: "mailto:" + organizerEmail,
					},
					Locations: locationObjs,
					VirtualLocations: map[string]jscalendar.VirtualLocation{
						virtualLocationId: virtualLocationObj,
					},
					Alerts: map[string]jscalendar.Alert{
						alertId: {
							Type: jscalendar.AlertType,
							Trigger: jscalendar.OffsetTrigger{
								Type:       jscalendar.OffsetTriggerType,
								Offset:     jscalendar.SignedDuration(alertOffset),
								RelativeTo: jscalendar.RelativeToStart,
							},
						},
					},
				},
			},
		}

		if EnableEventMayInviteFields {
			event["mayInviteSelf"] = true
			event["mayInviteOthers"] = true
			obj.MayInviteSelf = true
			obj.MayInviteOthers = true
			boxes.mayInvite = true
		}

		if len(keywords) > 0 {
			event["keywords"] = keywords
			obj.Keywords = keywords
			boxes.keywords = true
		}

		if len(categories) > 0 {
			event["categories"] = categories
			obj.Categories = categories
			boxes.categories = true
		}

		if mainLocationId != "" {
			event["mainLocationId"] = mainLocationId
			obj.MainLocationId = mainLocationId
		}

		err = propmap(i%2 == 0, 1, 1, event, "links", &obj.Links, func(int, string) (map[string]any, jscalendar.Link, error) {
			mime := ""
			uri := ""
			rel := jscalendar.RelAbout
			switch rand.Intn(2) {
			case 0:
				size := pickRandom(16, 24, 32, 48, 64)
				img := gofakeit.ImagePng(size, size)
				mime = "image/png"
				uri = "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(img)
			default:
				mime = "image/jpeg" //NOSONAR
				uri = externalImageUri()
			}
			return map[string]any{
					"@type":       "Link",
					"href":        uri,
					"contentType": mime,
					"rel":         string(rel),
				}, jscalendar.Link{
					Type:        jscalendar.LinkType,
					Href:        uri,
					ContentType: mime,
					Rel:         rel,
				}, nil
		})

		if rand.Intn(10) > 7 {
			frequency := pickRandom(jscalendar.FrequencyWeekly, jscalendar.FrequencyDaily)
			interval := pickRandom(1, 2)
			count := 1
			if frequency == jscalendar.FrequencyWeekly {
				count = 1 + rand.Intn(8)
			} else {
				count = 1 + rand.Intn(4)
			}
			event["recurrenceRule"] = map[string]any{
				"@type":          "RecurrenceRule",
				"frequency":      string(frequency),
				"interval":       interval,
				"rscale":         string(jscalendar.RscaleIso8601),
				"skip":           string(jscalendar.SkipOmit),
				"firstDayOfWeek": string(jscalendar.DayOfWeekMonday),
				"count":          count,
			}
			rr := jscalendar.RecurrenceRule{
				Type:           jscalendar.RecurrenceRuleType,
				Frequency:      frequency,
				Interval:       uint(interval),
				Rscale:         jscalendar.RscaleIso8601,
				Skip:           jscalendar.SkipOmit,
				FirstDayOfWeek: jscalendar.DayOfWeekMonday,
				Count:          uint(count),
			}
			obj.RecurrenceRule = &rr
		}

		id, err := s.CreateEvent(c, accountId, event)
		if err != nil {
			return accountId, calendarId, nil, boxes, err
		}

		obj.Id = id
		filled[id] = obj

		printer(fmt.Sprintf("📅 created %*s/%v id=%v", int(math.Log10(float64(count))+1), strconv.Itoa(int(i+1)), count, uid))
	}
	return accountId, calendarId, filled, boxes, nil
}

func (s *StalwartTest) CreateEvent(j *TestJmapClient, accountId string, event map[string]any) (string, error) {
	return j.create1(accountId, CalendarEventType, JmapCalendars, event)
}

var rooms = []jscalendar.Location{
	{
		Type:          jscalendar.LocationType,
		Name:          "Office meeting room upstairs",
		LocationTypes: toBoolMapS(jscalendar.LocationTypeOptionOffice),
		Coordinates:   "geo:52.5335389,13.4103296",
		Links: map[string]jscalendar.Link{
			"l1": {Href: "https://www.heinlein-support.de/"},
		},
	},
	{
		Type:          jscalendar.LocationType,
		Name:          "office-nue",
		LocationTypes: toBoolMapS(jscalendar.LocationTypeOptionOffice),
		Coordinates:   "geo:49.4723337,11.1042282",
		Links: map[string]jscalendar.Link{
			"l2": {Href: "https://www.workandpepper.de/"},
		},
	},
	{
		Type:          jscalendar.LocationType,
		Name:          "Meetingraum Prenzlauer Berg",
		LocationTypes: toBoolMapS(jscalendar.LocationTypeOptionOffice, jscalendar.LocationTypeOptionPublic),
		Coordinates:   "geo:52.554222,13.4142387",
		Links: map[string]jscalendar.Link{
			"l3": {Href: "https://www.spacebase.com/en/venue/meeting-room-prenzlauer-be-11499/"},
		},
	},
	{
		Type:          jscalendar.LocationType,
		Name:          "Meetingraum LIANE 1",
		LocationTypes: toBoolMapS(jscalendar.LocationTypeOptionOffice, jscalendar.LocationTypeOptionLibrary),
		Coordinates:   "geo:52.4854301,13.4224763",
		Links: map[string]jscalendar.Link{
			"l4": {Href: "https://www.spacebase.com/en/venue/rent-a-jungle-8372/"},
		},
	},
	{
		Type:          jscalendar.LocationType,
		Name:          "Dark Horse",
		LocationTypes: toBoolMapS(jscalendar.LocationTypeOptionOffice),
		Coordinates:   "geo:52.4942254,13.4346015",
		Links: map[string]jscalendar.Link{
			"l5": {Href: "https://www.spacebase.com/en/event-venue/workshop-white-space-2667/"},
		},
	},
}

var virtualRooms = []jscalendar.VirtualLocation{
	{
		Type: jscalendar.VirtualLocationType,
		Name: "opentalk",
		Uri:  "https://meet.opentalk.eu/fake/room/06fb8f7d-42eb-4212-8112-769fac2cb111",
		Features: toBoolMapS(
			jscalendar.VirtualLocationFeatureAudio,
			jscalendar.VirtualLocationFeatureChat,
			jscalendar.VirtualLocationFeatureVideo,
			jscalendar.VirtualLocationFeatureScreen,
		),
	},
}

func pickLocation() (string, map[string]any, jscalendar.Location) {
	locationId := id()
	room := rooms[rand.Intn(len(rooms))]
	b, err := json.Marshal(room)
	if err != nil {
		panic(err)
	}
	var m map[string]any
	err = json.Unmarshal(b, &m)
	if err != nil {
		panic(err)
	}
	return locationId, m, room
}

func pickVirtualLocation() (string, map[string]any, jscalendar.VirtualLocation) {
	locationId := id()
	vroom := virtualRooms[rand.Intn(len(virtualRooms))]
	b, err := json.Marshal(vroom)
	if err != nil {
		panic(err)
	}
	var m map[string]any
	err = json.Unmarshal(b, &m)
	if err != nil {
		panic(err)
	}
	return locationId, m, vroom
}

var ChairRoles = toBoolMapS(jscalendar.RoleChair, jscalendar.RoleOwner)
var RegularRoles = toBoolMapS(jscalendar.RoleOptional)

func createParticipants(uid string, locationIds []string, virtualLocationIds []string) (map[string]map[string]any, map[string]jscalendar.Participant, string) {
	options := structs.Concat(locationIds, virtualLocationIds)
	n := 1 + rand.Intn(4)
	maps := map[string]map[string]any{}
	objs := map[string]jscalendar.Participant{}
	organizerId, organizerEmail, organizerMap, organizerObj := createParticipant(0, uid, pickRandom(options...), "", "")
	maps[organizerId] = organizerMap
	objs[organizerId] = organizerObj
	for i := 1; i < n; i++ {
		id, _, participantMap, participantObj := createParticipant(i, uid, pickRandom(options...), organizerId, organizerEmail)
		maps[id] = participantMap
		objs[id] = participantObj
	}
	return maps, objs, organizerEmail
}

func createParticipant(i int, uid string, locationId string, organizerEmail string, organizerId string) (string, string, map[string]any, jscalendar.Participant) {
	participantId := id()
	person := gofakeit.Person()
	roles := RegularRoles
	if i == 0 {
		roles = ChairRoles
	}
	status := jscalendar.ParticipationStatusAccepted
	if i != 0 {
		status = pickRandom(
			jscalendar.ParticipationStatusNeedsAction,
			jscalendar.ParticipationStatusAccepted,
			jscalendar.ParticipationStatusDeclined,
			jscalendar.ParticipationStatusTentative,
		)
		//, delegated + set "delegatedTo"
	}
	statusComment := ""
	if rand.Intn(5) >= 3 {
		statusComment = gofakeit.HipsterSentence(1 + rand.Intn(5))
	}
	if i == 0 {
		organizerEmail = person.Contact.Email
		organizerId = participantId
	}

	name := person.FirstName + " " + person.LastName
	email := person.Contact.Email
	description := gofakeit.SentenceSimple()
	descriptionContentType := pickRandom("text/html", "text/plain")
	if descriptionContentType == "text/html" {
		description = toHtml(description)
	}
	language := pickLanguage()
	updated := "2025-10-01T01:59:12Z"
	updatedTime, err := time.Parse(time.RFC3339, updated)
	if err != nil {
		panic(err)
	}

	var calendarAddress string
	{
		pos := strings.LastIndex(email, "@")
		if pos < 0 {
			calendarAddress = email
		} else {
			local := email[0:pos]
			domain := email[pos+1:]
			calendarAddress = local + "+itip+" + uid + "@" + "itip." + domain
		}
	}

	m := map[string]any{
		"@type":                "Participant",
		"name":                 name,
		"email":                email,
		"calendarAddress":      calendarAddress,
		"kind":                 "individual",
		"roles":                structs.MapKeys(roles, func(r jscalendar.Role) string { return string(r) }),
		"locationId":           locationId,
		"language":             language,
		"participationStatus":  string(status),
		"participationComment": statusComment,
		"expectReply":          true,
		"scheduleAgent":        "server",
		"scheduleSequence":     1,
		"scheduleStatus":       []string{"1.0"},
		"scheduleUpdated":      updated,
		"sentBy":               organizerEmail,
		"invitedBy":            organizerId,
		"scheduleId":           "mailto:" + email,
	}
	o := jscalendar.Participant{
		Type:                 jscalendar.ParticipantType,
		Name:                 name,
		Email:                email,
		Kind:                 jscalendar.ParticipantKindIndividual,
		CalendarAddress:      calendarAddress,
		Roles:                roles,
		LocationId:           locationId,
		Language:             language,
		ParticipationStatus:  status,
		ParticipationComment: statusComment,
		ExpectReply:          true,
		ScheduleAgent:        jscalendar.ScheduleAgentServer,
		ScheduleSequence:     uint(1),
		ScheduleStatus:       []string{"1.0"},
		ScheduleUpdated:      updatedTime,
		SentBy:               organizerEmail,
		InvitedBy:            organizerId,
		ScheduleId:           "mailto:" + email,
	}

	if EnableEventParticipantDescriptionFields {
		m["description"] = description
		m["descriptionContentType"] = descriptionContentType
		o.Description = description
		o.DescriptionContentType = descriptionContentType
	}

	err = propmap(i%2 == 0, 1, 2, m, "links", &o.Links, func(int, string) (map[string]any, jscalendar.Link, error) {
		href := externalImageUri()
		title := person.FirstName + "'s Cake Day pick"
		return map[string]any{
				"@type":       "Link",
				"href":        href,
				"contentType": "image/jpeg",
				"rel":         "icon",
				"display":     "badge",
				"title":       title,
			}, jscalendar.Link{
				Type:        jscalendar.LinkType,
				Href:        href,
				ContentType: "image/jpeg",
				Rel:         jscalendar.RelIcon,
				Display:     jscalendar.DisplayBadge,
				Title:       title,
			}, nil
	})
	if err != nil {
		panic(err)
	}

	return participantId, person.Contact.Email, m, o
}

var Keywords = []string{
	"office",
	"important",
	"sales",
	"coordination",
	"decision",
}

func pickKeywords() map[string]bool {
	return toBoolMap(pickRandoms(Keywords...))
}

var Categories = []string{
	"http://opencloud.eu/categories/secret",
	"http://opencloud.eu/categories/internal",
}

func pickCategories() map[string]bool {
	return toBoolMap(pickRandoms(Categories...))
}
