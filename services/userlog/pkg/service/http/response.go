package http

import "github.com/opencloud-eu/opencloud/services/userlog/pkg/service"

// GetEventResponseOC10 is the response from GET events endpoint in oc10 style
type GetEventResponseOC10 struct {
	OCS struct {
		Meta struct {
			Message    string `json:"message"`
			Status     string `json:"status"`
			StatusCode int    `json:"statuscode"`
		} `json:"meta"`
		Data []service.OC10Notification `json:"data"`
	} `json:"ocs"`
}

// DeleteEventsRequest is the expected body for the delete request
type DeleteEventsRequest struct {
	IDs []string `json:"ids"`
}

// PostEventsRequest is the expected body for the post request
type PostEventsRequest struct {
	// the event type, e.g. "deprovision"
	Type string `json:"type"`
	// arbitray data for the event
	Data map[string]string `json:"data"`
}
