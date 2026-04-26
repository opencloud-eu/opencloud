package centrifuge

// SubscribedEvent is an event context passed
// to subscribe success callback.
type SubscribedEvent struct {
	Positioned     bool
	Recoverable    bool
	StreamPosition *StreamPosition
	WasRecovering  bool
	Recovered      bool
	Data           []byte
}

// SubscriptionErrorEvent is a subscribe error event context passed to
// event callback.
type SubscriptionErrorEvent struct {
	Error error
}

// SubscribingEvent is an event passed to subscribing event handler.
type SubscribingEvent struct {
	Code   uint32
	Reason string
}

// UnsubscribedEvent is an event passed to unsubscribe event handler.
type UnsubscribedEvent struct {
	Code   uint32
	Reason string
}

// LeaveEvent has info about user who left channel.
type LeaveEvent struct {
	ClientInfo
}

// JoinEvent has info about user who joined channel.
type JoinEvent struct {
	ClientInfo
}

// PublicationEvent has info about received channel Publication.
type PublicationEvent struct {
	Publication
}

// PublicationHandler is a function to handle messages published in
// channels.
type PublicationHandler func(PublicationEvent)

// JoinHandler is a function to handle join messages.
type JoinHandler func(JoinEvent)

// LeaveHandler is a function to handle leave messages.
type LeaveHandler func(LeaveEvent)

// UnsubscribedHandler is a function to handle unsubscribe event.
type UnsubscribedHandler func(UnsubscribedEvent)

// SubscribingHandler is a function to handle subscribe success event.
type SubscribingHandler func(SubscribingEvent)

// SubscribedHandler is a function to handle subscribe success event.
type SubscribedHandler func(SubscribedEvent)

// SubscriptionErrorHandler is a function to handle subscribe error event.
type SubscriptionErrorHandler func(SubscriptionErrorEvent)

// subscriptionEventHub contains callback functions that will be called when
// corresponding event happens with subscription to channel.
type subscriptionEventHub struct {
	onSubscribing SubscribingHandler
	onSubscribed  SubscribedHandler
	onUnsubscribe UnsubscribedHandler
	onError       SubscriptionErrorHandler
	onPublication PublicationHandler
	onJoin        JoinHandler
	onLeave       LeaveHandler
}

// newSubscriptionEventHub initializes new subscriptionEventHub.
func newSubscriptionEventHub() *subscriptionEventHub {
	return &subscriptionEventHub{}
}

// OnSubscribing allows setting SubscribingHandler to SubEventHandler.
func (s *Subscription) OnSubscribing(handler SubscribingHandler) {
	s.events.onSubscribing = handler
}

// OnSubscribed allows setting SubscribedHandler to SubEventHandler.
func (s *Subscription) OnSubscribed(handler SubscribedHandler) {
	s.events.onSubscribed = handler
}

// OnUnsubscribed allows setting UnsubscribedHandler to SubEventHandler.
func (s *Subscription) OnUnsubscribed(handler UnsubscribedHandler) {
	s.events.onUnsubscribe = handler
}

// OnError allows setting SubscriptionErrorHandler to SubEventHandler.
func (s *Subscription) OnError(handler SubscriptionErrorHandler) {
	s.events.onError = handler
}

// OnPublication allows setting PublicationHandler to SubEventHandler.
func (s *Subscription) OnPublication(handler PublicationHandler) {
	s.events.onPublication = handler
}

// OnJoin allows setting JoinHandler to SubEventHandler.
func (s *Subscription) OnJoin(handler JoinHandler) {
	s.events.onJoin = handler
}

// OnLeave allows setting LeaveHandler to SubEventHandler.
func (s *Subscription) OnLeave(handler LeaveHandler) {
	s.events.onLeave = handler
}
