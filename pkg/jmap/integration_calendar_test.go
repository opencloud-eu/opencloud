package jmap

import (
	"encoding/base64"
	"fmt"
	golog "log"
	"maps"
	"math"
	"math/rand"
	"slices"
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

func TestCalendars(t *testing.T) { //NOSONAR
	if skip(t) {
		return
	}

	containerTest(t,
		func(session *Session) string { return session.PrimaryAccounts.Calendars },
		func(resp CalendarGetResponse) []Calendar { return resp.List },
		func(obj Calendar) string { return obj.Id },
		func(s *StalwartTest, accountId string, ids []string, ctx Context) (CalendarGetResponse, SessionState, State, Language, Error) {
			return s.client.GetCalendars(accountId, ids, ctx)
		},
		func(s *StalwartTest, accountId string, id string, change CalendarChange, ctx Context) (Calendar, SessionState, State, Language, Error) { //NOSONAR
			return s.client.UpdateCalendar(accountId, id, change, ctx)
		},
		func(s *StalwartTest, accountId string, ids []string, ctx Context) (map[string]SetError, SessionState, State, Language, Error) { //NOSONAR
			return s.client.DeleteCalendar(accountId, ids, ctx)
		},
		func(s *StalwartTest, t *testing.T, accountId string, count uint, ctx Context, user User, principalIds []string) (CalendarBoxes, []Calendar, SessionState, State, error) {
			return s.fillCalendar(t, accountId, count, ctx, user, principalIds)
		},
		func(orig Calendar) CalendarChange {
			return CalendarChange{
				Description:  ptr(orig.Description + " (changed)"),
				IsSubscribed: ptr(!orig.IsSubscribed),
			}
		},
		func(t *testing.T, orig Calendar, _ CalendarChange, changed Calendar) {
			require.Equal(t, orig.Name, changed.Name)
			require.Equal(t, orig.Description+" (changed)", changed.Description)
			require.Equal(t, !orig.IsSubscribed, changed.IsSubscribed)
		},
	)
}

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
	ctx := s.Context(session)

	accountId, calendarId, expectedEventsById, boxes, err := s.fillEvents(t, count, ctx, user)
	require.NoError(err)
	require.NotEmpty(accountId)
	require.NotEmpty(calendarId)

	filter := CalendarEventFilterCondition{
		InCalendar: calendarId,
	}
	sortBy := []CalendarEventComparator{
		{Property: CalendarEventPropertyStart, IsAscending: true},
	}

	ss := EmptySessionState
	os := EmptyState
	{
		resultsByAccount, sessionState, state, _, err := s.client.QueryCalendarEvents([]string{accountId}, filter, sortBy, 0, "", nil, nil, true, ctx)
		require.NoError(err)

		require.Len(resultsByAccount, 1)
		require.Contains(resultsByAccount, accountId)
		results := resultsByAccount[accountId]
		require.Len(results.Results, int(count))
		require.Equal(uint(0), results.Limit)
		require.Equal(uint(0), results.Position)
		require.Equal(true, results.CanCalculateChanges)
		require.NotNil(results.Total)
		require.Equal(count, *results.Total)

		for _, actual := range results.Results {
			expected, ok := expectedEventsById[actual.Id]
			require.True(ok, "failed to find created contact by its id")
			matchEvent(t, actual, expected)
		}

		ss = sessionState
		os = state
	}

	{
		limit := uint(10)
		slices := count / limit
		remainder := count
		require.Greater(slices, uint(1), "we need to have more than 10 objects in order to test the pagination of search results")
		for i := range slices {
			position := int(i * limit)
			page := min(remainder, limit)
			m, sessionState, _, _, err := s.client.QueryCalendarEvents([]string{accountId}, filter, sortBy, position, "", nil, &limit, true, ctx)
			require.NoError(err)
			require.Len(m, 1)
			require.Contains(m, accountId)
			results := m[accountId]
			require.Equal(len(results.Results), int(page))
			require.Equal(limit, results.Limit)
			require.Equal(uint(position), results.Position)
			require.Equal(true, results.CanCalculateChanges)
			require.NotNil(results.Total)
			require.Equal(count, *results.Total)
			remainder -= uint(len(results.Results))

			require.Equal(ss, sessionState)
		}
	}

	for _, event := range expectedEventsById {
		change := CalendarEventChange{
			EventChange: jscalendar.EventChange{
				Status: ptr(jscalendar.StatusCancelled),
				ObjectChange: jscalendar.ObjectChange{
					Sequence:        uintPtr(99),
					ShowWithoutTime: truep,
				},
			},
		}
		changed, sessionState, state, _, err := s.client.UpdateCalendarEvent(accountId, event.Id, change, ctx)
		require.NoError(err)
		require.Equal(jscalendar.StatusCancelled, changed.Status)
		require.Equal(uint(99), changed.Sequence)
		require.Equal(true, changed.ShowWithoutTime)
		require.Equal(ss, sessionState)
		require.NotEqual(os, state)
		os = state
	}

	{
		ids := structs.Map(slices.Collect(maps.Values(expectedEventsById)), func(e CalendarEvent) string { return e.Id })
		errMap, sessionState, state, _, err := s.client.DeleteCalendarEvent(accountId, ids, ctx)
		require.NoError(err)
		require.Empty(errMap)

		require.Equal(ss, sessionState)
		require.NotEqual(os, state)
		os = state
	}

	{
		shouldBeEmpty, sessionState, state, _, err := s.client.QueryCalendarEvents([]string{accountId}, filter, sortBy, 0, "", nil, nil, true, ctx)
		require.NoError(err)
		require.Contains(shouldBeEmpty, accountId)
		resp := shouldBeEmpty[accountId]
		require.Empty(resp.Results)
		require.NotNil(resp.Total)
		require.Equal(uint(0), *resp.Total)
		require.Equal(ss, sessionState)
		require.Equal(os, state)
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

type CalendarBoxes struct {
	sharedReadOnly  bool
	sharedReadWrite bool
	sharedDelete    bool
	sortOrdered     bool
}

func (s *StalwartTest) fillCalendar( //NOSONAR
	t *testing.T,
	accountId string,
	count uint,
	ctx Context,
	_ User,
	principalIds []string,
) (CalendarBoxes, []Calendar, SessionState, State, error) {
	require := require.New(t)

	boxes := CalendarBoxes{}
	created := []Calendar{}
	ss := EmptySessionState
	as := EmptyState

	printer := func(s string) { golog.Println(s) }

	for i := range count {
		name := gofakeit.Company()
		description := gofakeit.SentenceSimple()
		subscribed := gofakeit.Bool()
		visible := gofakeit.Bool()
		color := gofakeit.HexColor()
		include := pickRandom(IncludeInAvailabilities...)
		dawtId := gofakeit.UUID()
		daotId := gofakeit.UUID()
		cal := CalendarChange{
			Name:                  &name,
			Description:           &description,
			IsSubscribed:          &subscribed,
			Color:                 &color,
			IsVisible:             &visible,
			IncludeInAvailability: &include,
			DefaultAlertsWithTime: map[string]jscalendar.Alert{
				dawtId: {
					Type: jscalendar.AlertType,
					Trigger: jscalendar.OffsetTrigger{
						Type:       jscalendar.OffsetTriggerType,
						Offset:     "-PT5M",
						RelativeTo: jscalendar.RelativeToStart,
					},
					Action: jscalendar.AlertActionDisplay,
				},
			},
			DefaultAlertsWithoutTime: map[string]jscalendar.Alert{
				daotId: {
					Type: jscalendar.AlertType,
					Trigger: jscalendar.OffsetTrigger{
						Type:       jscalendar.OffsetTriggerType,
						Offset:     "-PT24H",
						RelativeTo: jscalendar.RelativeToStart,
					},
					Action: jscalendar.AlertActionDisplay,
				},
			},
		}
		if i%2 == 0 {
			cal.SortOrder = uintPtr(gofakeit.Uint())
			boxes.sortOrdered = true
		}
		var sharing *CalendarRights = nil
		switch i % 4 {
		default:
			// no sharing
		case 1:
			sharing = &CalendarRights{
				MayReadFreeBusy:  true,
				MayReadItems:     true,
				MayRSVP:          true,
				MayAdmin:         false,
				MayDelete:        false,
				MayWriteAll:      false,
				MayWriteOwn:      false,
				MayUpdatePrivate: false,
			}
			boxes.sharedReadWrite = true
		case 2:
			sharing = &CalendarRights{
				MayReadFreeBusy:  true,
				MayReadItems:     true,
				MayRSVP:          true,
				MayAdmin:         false,
				MayDelete:        false,
				MayWriteAll:      false,
				MayWriteOwn:      true,
				MayUpdatePrivate: true,
			}
			boxes.sharedReadOnly = true
		case 3:
			sharing = &CalendarRights{
				MayReadFreeBusy:  true,
				MayReadItems:     true,
				MayRSVP:          true,
				MayAdmin:         false,
				MayDelete:        true,
				MayWriteAll:      true,
				MayWriteOwn:      true,
				MayUpdatePrivate: true,
			}
			boxes.sharedDelete = true
		}
		if sharing != nil {
			numPrincipals := 1 + rand.Intn(len(principalIds)-1)
			m := make(map[string]CalendarRights, numPrincipals)
			for _, p := range pickRandomN(numPrincipals, principalIds...) {
				m[p] = *sharing
			}
			cal.ShareWith = m
		}

		a, sessionState, state, _, err := s.client.CreateCalendar(accountId, cal, ctx)
		if err != nil {
			return boxes, created, ss, as, err
		}
		require.NotEmpty(sessionState)
		require.NotEmpty(state)
		if ss != EmptySessionState {
			require.Equal(ss, sessionState)
		}
		if as != EmptyState {
			require.NotEqual(as, state)
		}
		require.NotNil(a)
		created = append(created, *a)
		ss = sessionState
		as = state

		printer(fmt.Sprintf("📅 created %*s/%v id=%v", int(math.Log10(float64(count))+1), strconv.Itoa(int(i+1)), count, a.Id))
	}
	return boxes, created, ss, as, nil
}

type EventsBoxes struct {
	categories bool
	keywords   bool
	mayInvite  bool
}

func (s *StalwartTest) fillEvents( //NOSONAR
	t *testing.T,
	count uint,
	ctx Context,
	user User,
) (string, string, map[string]CalendarEvent, EventsBoxes, error) {
	require := require.New(t)
	c, err := NewTestJmapClient(ctx.Session, user.name, user.password, true, true)
	require.NoError(err)
	defer c.Close()

	boxes := EventsBoxes{}

	printer := func(s string) { golog.Println(s) }

	accountId := c.session.PrimaryAccounts.Calendars
	require.NotEmpty(accountId, "no primary account for calendars in session")

	calendarId := ""
	{
		calendarsById, err := c.objectsById(accountId, CalendarType)
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
		locationObjs := map[string]jscalendar.Location{}
		{
			n := 1
			if i%4 == 0 {
				n++
			}
			for range n {
				locationId, locationObj := pickLocation()
				locationObjs[locationId] = locationObj
				locationIds = append(locationIds, locationId)
				if n > 0 && mainLocationId == "" {
					mainLocationId = locationId
				}
			}
		}
		virtualLocationId, virtualLocationObj := pickVirtualLocation()
		participantObjs, organizerEmail := createParticipants(uid, locationIds, []string{virtualLocationId})
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

		sequence := uint(0)

		alertId := id()
		alertOffset := pickRandom("-PT5M", "-PT10M", "-PT15M")

		obj := CalendarEventChange{
			CalendarIds: toBoolMapS(calendarId),
			IsDraft:     &isDraft,
			EventChange: jscalendar.EventChange{
				Type:     jscalendar.EventType,
				Start:    jscalendar.LocalDateTime(start),
				Duration: ptr(jscalendar.Duration(duration)),
				Status:   &status,
				ObjectChange: jscalendar.ObjectChange{
					CommonObjectChange: jscalendar.CommonObjectChange{
						Uid:                    &uid,
						ProdId:                 &productName,
						Title:                  &title,
						Description:            &description,
						DescriptionContentType: &descriptionFormat,
						Locale:                 &locale,
						Color:                  &color,
					},
					Sequence:        uintPtr(sequence),
					ShowWithoutTime: falsep,
					FreeBusyStatus:  &freeBusy,
					Privacy:         &privacy,
					SentBy:          organizerEmail,
					Participants:    participantObjs,
					TimeZone:        &tz,
					HideAttendees:   falsep,
					ReplyTo: map[jscalendar.ReplyMethod]string{
						jscalendar.ReplyMethodImip: "mailto:" + organizerEmail, //NOSONAR
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
			obj.MayInviteSelf = truep
			obj.MayInviteOthers = truep
			boxes.mayInvite = true
		}

		if len(keywords) > 0 {
			obj.Keywords = keywords
			boxes.keywords = true
		}

		if len(categories) > 0 {
			obj.Categories = categories
			boxes.categories = true
		}

		if mainLocationId != "" {
			obj.MainLocationId = &mainLocationId
		}

		err = propmap(i%2 == 0, 1, 1, &obj.Links, func(int, string) (jscalendar.Link, error) {
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
			return jscalendar.Link{
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

		created, _, _, _, err := s.client.CreateCalendarEvent(accountId, obj, ctx)
		if err != nil {
			return accountId, calendarId, nil, boxes, err
		}

		filled[created.Id] = *created

		printer(fmt.Sprintf("📅 created %*s/%v id=%v", int(math.Log10(float64(count))+1), strconv.Itoa(int(i+1)), count, uid))
	}
	return accountId, calendarId, filled, boxes, nil
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

func pickLocation() (string, jscalendar.Location) {
	locationId := id()
	room := rooms[rand.Intn(len(rooms))]
	return locationId, room
}

func pickVirtualLocation() (string, jscalendar.VirtualLocation) {
	locationId := id()
	vroom := virtualRooms[rand.Intn(len(virtualRooms))]
	return locationId, vroom
}

var ChairRoles = toBoolMapS(jscalendar.RoleChair, jscalendar.RoleOwner)
var RegularRoles = toBoolMapS(jscalendar.RoleOptional)

func createParticipants(uid string, locationIds []string, virtualLocationIds []string) (map[string]jscalendar.Participant, string) {
	options := structs.Concat(locationIds, virtualLocationIds)
	n := 1 + rand.Intn(4)
	objs := map[string]jscalendar.Participant{}
	organizerId, organizerEmail, organizerObj := createParticipant(0, uid, pickRandom(options...), "", "")
	objs[organizerId] = organizerObj
	for i := 1; i < n; i++ {
		id, _, participantObj := createParticipant(i, uid, pickRandom(options...), organizerId, organizerEmail)
		objs[id] = participantObj
	}
	return objs, organizerEmail
}

func createParticipant(i int, uid string, locationId string, organizerEmail string, organizerId string) (string, string, jscalendar.Participant) {
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
		o.Description = description
		o.DescriptionContentType = descriptionContentType
	}

	err = propmap(i%2 == 0, 1, 2, &o.Links, func(int, string) (jscalendar.Link, error) {
		href := externalImageUri()
		title := person.FirstName + "'s Cake Day pick"
		return jscalendar.Link{
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

	return participantId, person.Contact.Email, o
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
