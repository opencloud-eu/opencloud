package groupware

import (
	"github.com/opencloud-eu/opencloud/pkg/jmap"
	"github.com/opencloud-eu/opencloud/pkg/jscalendar"
)

var TL1 = jmap.TaskList{
	Id:          "aemua9ai",
	Role:        jmap.TaskListRoleInbox,
	Name:        "Your Tasks",
	Description: "Your default list of tasks",
	Color:       "purple",
	KeywordColors: map[string]string{
		"todo": "blue",
		"done": "green",
	},
	CategoryColors: map[string]string{
		"work": "magenta",
	},
	SortOrder:    1,
	IsSubscribed: true,
	TimeZone:     "CEST",
	WorkflowStatuses: []string{
		"new", "todo", "in-progress", "done",
	},
	ShareWith: map[string]jmap.TaskRights{
		"eefeeb4p": {
			MayReadItems:     true,
			MayWriteAll:      false,
			MayWriteOwn:      true,
			MayUpdatePrivate: false,
			MayRSVP:          false,
			MayAdmin:         false,
			MayDelete:        false,
		},
	},
	MyRights: &jmap.TaskRights{
		MayReadItems:     true,
		MayWriteAll:      true,
		MayWriteOwn:      true,
		MayUpdatePrivate: true,
		MayRSVP:          true,
		MayAdmin:         false,
		MayDelete:        false,
	},
	DefaultAlertsWithTime: map[string]jscalendar.Alert{
		"saenee7a": {
			Type: jscalendar.AlertType,
			Trigger: jscalendar.OffsetTrigger{
				Type:       jscalendar.OffsetTriggerType,
				Offset:     "-PT10M",
				RelativeTo: jscalendar.RelativeToStart,
			},
			Action: jscalendar.AlertActionEmail,
		},
	},
	DefaultAlertsWithoutTime: map[string]jscalendar.Alert{
		"xiipaew9": {
			Type: jscalendar.AlertType,
			Trigger: jscalendar.OffsetTrigger{
				Type:       jscalendar.OffsetTriggerType,
				Offset:     "-PT12H",
				RelativeTo: jscalendar.RelativeToStart,
			},
			Action: jscalendar.AlertActionDisplay,
		},
	},
}

var T1 = jmap.Task{
	Id:             "laoj0ahk",
	TaskListId:     TL1.Id,
	IsDraft:        false,
	UtcStart:       jmap.UTCDate{Time: mustParseTime("2025-10-02T10:00:00Z")},
	UtcDue:         jmap.UTCDate{Time: mustParseTime("2025-10-12T18:00:00Z")},
	SortOrder:      1,
	WorkflowStatus: "new",
	Task: jscalendar.Task{
		Type: jscalendar.TaskType,
		Object: jscalendar.Object{
			CommonObject: jscalendar.CommonObject{
				Uid:                    "7da0d4a2-385c-430f-9022-61db302734d9",
				ProdId:                 "Mock 0.0",
				Created:                "2025-10-01T17:31:49",
				Updated:                "2025-10-01T17:35:12",
				Title:                  "Crossing the Ring",
				Description:            "We need to cross the Ring the protomolecule opened.",
				DescriptionContentType: "text/plain",
				Links: map[string]jscalendar.Link{
					"theisha5": {
						Type:        jscalendar.LinkType,
						Href:        "https://static.wikia.nocookie.net/expanse/images/e/ed/S03E09-SlowZone_01.jpg/revision/latest/scale-to-width-down/1000?cb=20180611184722",
						ContentType: "image/jpeg",
						Size:        109212,
						Rel:         jscalendar.RelIcon,
						Display:     "sol gate",
						Title:       "The Sol Ring Gate",
					},
				},
				Locale: "en-GB",
				Keywords: map[string]bool{
					"todo": true,
				},
				Categories: map[string]bool{
					"work": true,
				},
				Color: "yellow",
			},
			Sequence:        1,
			ShowWithoutTime: false,
			Locations: map[string]jscalendar.Location{
				"ruoth5uu": {
					Type:        jscalendar.LocationType,
					Name:        "Sol Gate",
					Description: "We meet at the Sol gate",
					LocationTypes: map[jscalendar.LocationTypeOption]bool{
						jscalendar.LocationTypeOptionLandmarkAddress: true,
					},
					Coordinates: "geo:40.4165583,-3.7063595",
					Links: map[string]jscalendar.Link{
						"jeeshei5": {
							Type:        jscalendar.LinkType,
							Href:        "https://expanse.fandom.com/wiki/Sol_gate",
							ContentType: "text/html",
							Title:       "The Sol Gate",
						},
					},
				},
			},
			Priority:       1,
			FreeBusyStatus: jscalendar.FreeBusyStatusBusy,
			Privacy:        jscalendar.PrivacySecret,
			Alerts: map[string]jscalendar.Alert{
				"eiphuw4a": {
					Type: jscalendar.AlertType,
					Trigger: jscalendar.AbsoluteTrigger{
						Type: jscalendar.AbsoluteTriggerType,
						When: mustParseTime("2025-12-01T10:11:12Z"),
					},
					Action: jscalendar.AlertActionDisplay,
				},
			},
			TimeZone:        "UTC",
			MayInviteSelf:   true,
			MayInviteOthers: true,
			HideAttendees:   true,
		},
		Due:               jscalendar.LocalDateTime("2025-12-01T10:11:12"),
		Start:             jscalendar.LocalDateTime("2025-10-01T08:00:00"),
		EstimatedDuration: "PT8W",
		PercentComplete:   5,
		Progress:          jscalendar.ProgressNeedsAction,
		ProgressUpdated:   mustParseTime("2025-10-01T08:12:39Z"),
	},
	EstimatedWork:   4,
	Impact:          "block",
	IsOrigin:        true,
	MayInviteSelf:   true,
	MayInviteOthers: true,
	HideAttendees:   false,
	Checklists: map[string]jmap.Checklist{
		"sae9aimu": {
			Type:  jmap.ChecklistType,
			Title: "Prerequisites",
			CheckItems: []jmap.CheckItem{
				{
					Type:       jmap.CheckItemType,
					Title:      "Control Medina Station",
					SortOrder:  1,
					IsComplete: true,
					Updated:    jmap.UTCDate{Time: mustParseTime("2025-04-01T09:32:10Z")},
					Assignee: &jmap.TaskPerson{
						Type:        jmap.TaskPersonType,
						Name:        "Fred Johnson",
						Uri:         "mailto:johnson@opa.org",
						PrincipalId: "nae5hu9t",
					},
					Comments: map[string]jmap.Comment{
						"ooze1iet": {
							Type:    jmap.CommentType,
							Message: "We first need to control Medina Station before we can get through the Sol Gate",
							Created: jmap.UTCDate{Time: mustParseTime("2025-04-01T12:11:10Z")},
							Updated: jmap.UTCDate{Time: mustParseTime("2025-04-01T12:29:19Z")},
							Author: &jmap.TaskPerson{
								Type:        jmap.TaskPersonType,
								Name:        "Anderson Dawes",
								Uri:         "mailto:adawes@opa.org",
								PrincipalId: "eshi9oot",
							},
						},
					},
				},
			},
		},
	},
}

var AllTaskLists = []jmap.TaskList{TL1}

var TaskMapByTaskListId = map[string][]jmap.Task{
	TL1.Id: {
		T1,
	},
}
