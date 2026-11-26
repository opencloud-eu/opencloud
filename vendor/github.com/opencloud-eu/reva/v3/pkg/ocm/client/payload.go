package client

import (
	"bytes"
	"encoding/json"
	"io"

	"github.com/opencloud-eu/reva/v3/internal/http/services/ocmd"
)

// https://cs3org.github.io/OCM-API/docs.html?branch=develop&repo=OCM-API&user=cs3org#/paths/~1notifications/post
// NotificationType one of "SHARE_ACCEPTED", "SHARE_DECLINED", "SHARE_CHANGE_PERMISSION", "SHARE_UNSHARED", "USER_REMOVED"
const (
	SHARE_UNSHARED          = "SHARE_UNSHARED"
	SHARE_CHANGE_PERMISSION = "SHARE_CHANGE_PERMISSION"
)

// InviteAcceptedRequest contains the parameters for accepting
// an invitation.
type InviteAcceptedRequest struct {
	UserID            string `json:"userID"`
	Email             string `json:"email"`
	Name              string `json:"name"`
	RecipientProvider string `json:"recipientProvider"`
	Token             string `json:"token"`
}

// User contains the remote user's information when accepting
// an invitation.
type User struct {
	UserID string `json:"userID"`
	Email  string `json:"email"`
	Name   string `json:"name"`
}

func (r *InviteAcceptedRequest) toJSON() (io.Reader, error) {
	var b bytes.Buffer
	if err := json.NewEncoder(&b).Encode(r); err != nil {
		return nil, err
	}
	return &b, nil
}

// NewShareRequest contains the parameters for creating a new OCM share.
type NewShareRequest struct {
	ShareWith         string         `json:"shareWith"`
	Name              string         `json:"name"`
	Description       string         `json:"description"`
	ProviderID        string         `json:"providerId"`
	Owner             string         `json:"owner"`
	Sender            string         `json:"sender"`
	OwnerDisplayName  string         `json:"ownerDisplayName"`
	SenderDisplayName string         `json:"senderDisplayName"`
	ShareType         string         `json:"shareType"`
	Expiration        uint64         `json:"expiration"`
	ResourceType      string         `json:"resourceType"`
	Protocols         ocmd.Protocols `json:"protocol"`
}

func (r *NewShareRequest) toJSON() (io.Reader, error) {
	var b bytes.Buffer
	if err := json.NewEncoder(&b).Encode(r); err != nil {
		return nil, err
	}
	return &b, nil
}

// Capabilities contains a set of properties exposed by
// a remote cloud storage.
type Capabilities struct {
	Enabled       bool   `json:"enabled"`
	APIVersion    string `json:"apiVersion"`
	EndPoint      string `json:"endPoint"`
	Provider      string `json:"provider"`
	ResourceTypes []struct {
		Name       string   `json:"name"`
		ShareTypes []string `json:"shareTypes"`
		Protocols  struct {
			Webdav *string `json:"webdav"`
			Webapp *string `json:"webapp"`
			Datatx *string `json:"datatx"`
		} `json:"protocols"`
	} `json:"resourceTypes"`
	Capabilities []string `json:"capabilities"`
}

// NotificationRequest is the request payload for the OCM API notifications endpoint.
// https://cs3org.github.io/OCM-API/docs.html?branch=develop&repo=OCM-API&user=cs3org#/paths/~1notifications/post
type NotificationRequest struct {
	NotificationType string `json:"notificationType" validate:"required"`
	ResourceType     string `json:"resourceType" validate:"required"`
	// Identifier to identify the shared resource at the provider side. This is unique per provider such that if the same resource is shared twice, this providerId will not be repeated.
	ProviderId string `json:"providerId" validate:"required"`
	// Optional additional parameters, depending on the notification and the resource type.
	Notification *Notification `json:"notification,omitempty"`
}

// Notification is the payload for the notification field in the NotificationRequest.
type Notification struct {
	// Owner        string `json:"owner,omitempty"`
	Grantee      string         `json:"grantee,omitempty"`
	SharedSecret string         `json:"sharedSecret,omitempty"`
	Protocols    ocmd.Protocols `json:"protocol,omitempty"`
}

// ToJSON returns the JSON io.Reader of the NotificationRequest.
func (r *NotificationRequest) ToJSON() (io.Reader, error) {
	var b bytes.Buffer
	if err := json.NewEncoder(&b).Encode(r); err != nil {
		return nil, err
	}
	return &b, nil
}

// NewShareResponse is the response returned when creating a new share.
type NewShareResponse struct {
	RecipientDisplayName string `json:"recipientDisplayName"`
}

// ErrorMessageResponse is the response returned by the OCM API in case of an error.
// https://cs3org.github.io/OCM-API/docs.html?branch=develop&repo=OCM-API&user=cs3org#/paths/~1notifications/post
type ErrorMessageResponse struct {
	Message          string             `json:"message"`
	ValidationErrors []*ValidationError `json:"validationErrors,omitempty"`
}

// ValidationError is the payload for the validationErrors field in the ErrorMessageResponse.
type ValidationError struct {
	Name    string `json:"name"`
	Message string `json:"message"`
}
