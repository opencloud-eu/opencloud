package remote

import (
	"encoding/json"
	"time"
)

type Notification any

type Topic string

const (
	TopicThemeAssigned   Topic = "theme.assigned"
	TopicThemeUnassigned Topic = "theme.unassigned"
)

type Message[P any] struct {
	Topic      Topic     `json:"topic"`
	TenantId   string    `json:"tenantId"`
	InstanceId string    `json:"instanceId"`
	Timestamp  time.Time `json:"timestamp"`
	Payload    P         `json:"payload"`
}

type MessagePayloadThemeAssigned struct {
	ThemeId string `json:"themeId"`
}

func dispatchMessage[T any](h func(message Message[T]) error, b []byte) error {
	var message Message[T]
	if err := json.Unmarshal(b, &message); err != nil {
		return err
	}

	return h(message)
}

type Handler struct {
	ThemeAssigned   func(Message[MessagePayloadThemeAssigned]) error
	ThemeUnassigned func(Message[struct{}]) error
}
