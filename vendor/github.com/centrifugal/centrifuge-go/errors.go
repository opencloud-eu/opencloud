package centrifuge

import (
	"errors"
	"fmt"
)

var (
	// ErrTimeout returned if operation timed out.
	ErrTimeout = errors.New("timeout")
	// ErrClientDisconnected can be returned if client goes to
	// disconnected state while operation in progress.
	ErrClientDisconnected = errors.New("client disconnected")
	// ErrClientClosed can be returned if client is closed.
	ErrClientClosed = errors.New("client closed")
	// ErrSubscriptionUnsubscribed returned if Subscription is unsubscribed.
	ErrSubscriptionUnsubscribed = errors.New("subscription unsubscribed")
	// ErrDuplicateSubscription returned if subscription to the same channel
	// already registered in current client instance. This is due to the fact
	// that server does not allow subscribing to the same channel twice for
	// the same connection.
	ErrDuplicateSubscription = errors.New("duplicate subscription")
	// ErrUnauthorized is a special error which may be returned by application
	// from GetToken function to indicate lack of operation permission.
	ErrUnauthorized = errors.New("unauthorized")
)

type TransportError struct {
	Err error
}

func (t TransportError) Error() string {
	return fmt.Sprintf("transport error: %v", t.Err)
}

func (t TransportError) Unwrap() error {
	return t.Err
}

type ConnectError struct {
	Err error
}

func (c ConnectError) Error() string {
	return fmt.Sprintf("connect error: %v", c.Err)
}

func (c ConnectError) Unwrap() error {
	return c.Err
}

type RefreshError struct {
	Err error
}

func (r RefreshError) Error() string {
	return fmt.Sprintf("refresh error: %v", r.Err)
}

func (r RefreshError) Unwrap() error {
	return r.Err
}

type ConfigurationError struct {
	Err error
}

func (c ConfigurationError) Error() string {
	return fmt.Sprintf("configuration error: %v", c.Err)
}

func (c ConfigurationError) Unwrap() error {
	return c.Err
}

type SubscriptionSubscribeError struct {
	Err error
}

func (s SubscriptionSubscribeError) Error() string {
	return fmt.Sprintf("subscribe error: %v", s.Err)
}

func (s SubscriptionSubscribeError) Unwrap() error {
	return s.Err
}

type SubscriptionRefreshError struct {
	Err error
}

func (s SubscriptionRefreshError) Error() string {
	return fmt.Sprintf("refresh error: %v", s.Err)
}

func (s SubscriptionRefreshError) Unwrap() error {
	return s.Err
}
