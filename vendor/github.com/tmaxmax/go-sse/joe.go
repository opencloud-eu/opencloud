package sse

import (
	"context"
	"runtime/debug"
	"sync"
)

// A Replayer is a type that can replay older published events to new subscribers.
// Replayers use event IDs, the topics the events were published and optionally
// any other criteria to determine which are valid for replay.
//
// While replayers can require events to have IDs beforehand, they can also set the IDs themselves,
// automatically - it's up to the implementation. Replayers should not overwrite or remove any existing
// IDs and return an error instead.
//
// Replayers are not required to be thread-safe - server providers are required to ensure only
// one operation is executed on the replayer at any given time. Server providers may not execute
// replay operation concurrently with other operations, so make sure any action on the replayer
// blocks for as little as possible. If a replayer is thread-safe, some operations may be
// run in a separate goroutine - see the interface's method documentation.
//
// Executing actions that require waiting for a long time on I/O, such as HTTP requests or database
// calls must be handled with great care, so the server provider is not blocked. Reducing them to
// the minimum by using techniques such as caching or by executing them in separate goroutines is
// recommended, as long as the implementation fulfills the requirements.
//
// If not specified otherwise, the errors returned are implementation-specific.
type Replayer interface {
	// Put adds a new event to the replay buffer. The Message that is returned may not have the
	// same address, if the replayer automatically sets IDs.
	//
	// Put errors if the message couldn't be queued – if no topics are provided,
	// a message without an ID is put into a Replayer which does not
	// automatically set IDs, or a message with an ID is put into a Replayer which
	// does automatically set IDs. An error should be returned for other failures
	// related to the given message. When no topics are provided, ErrNoTopic should be
	// returned.
	//
	// The Put operation may be executed by the replayer in another goroutine only if
	// it can ensure that any Replay operation called after the Put goroutine is started
	// can replay the new received message. This also requires the replayer implementation
	// to be thread-safe.
	//
	// Replayers are not required to guarantee that immediately after Put returns
	// the new messages can be replayed. If an error occurs internally when putting the new message
	// and retrying the operation would block for too long, it can be aborted.
	//
	// To indicate a complete replayer failure (i.e. the replayer won't work after this point)
	// a panic should be used instead of an error.
	Put(message *Message, topics []string) (*Message, error)
	// Replay sends to a new subscriber all the valid events received by the replayer
	// since the event with the listener's ID. If the ID the listener provides
	// is invalid, the provider should not replay any events.
	//
	// Replay calls must return only after replaying is done.
	// Implementations should not keep references to the subscription client
	// after Replay returns.
	//
	// If an error is returned, then at least some messages weren't successfully replayed.
	// The error is nil if there were no messages to replay for the particular subscription
	// or if all messages were replayed successfully.
	//
	// If any messages are replayed, Client.Flush must be called by implementations.
	Replay(subscription Subscription) error
}

type (
	subscriber   chan<- error
	subscription struct {
		done subscriber
		Subscription
	}

	messageWithTopics struct {
		message *Message
		topics  []string
	}

	publishedMessage struct {
		replayerErr chan<- error
		messageWithTopics
	}
)

// Joe is a basic server provider that synchronously executes operations by queueing them in channels.
// Events are also sent synchronously to subscribers, so if a subscriber's callback blocks, the others
// have to wait.
//
// Joe optionally supports event replaying with the help of a Replayer.
//
// If the replayer panics, the subscription for which it panicked is considered failed
// and an error is returned, and thereafter the replayer is not used anymore – no replays
// will be attempted for future subscriptions.
// If due to some other unexpected scenario something panics internally, Joe will remove all subscribers
// and close itself, so subscribers don't end up blocked.
//
// He serves simple use-cases well, as he's light on resources, and does not require any external
// services. Also, he is the default provider for Servers.
type Joe struct {
	message        chan publishedMessage
	subscription   chan subscription
	unsubscription chan subscriber
	done           chan struct{}
	closed         chan struct{}
	subscribers    map[subscriber]Subscription

	// An optional replayer that Joe uses to resend older messages to new subscribers.
	Replayer Replayer

	initDone sync.Once
}

// Subscribe tells Joe to send new messages to this subscriber. The subscription
// is automatically removed when the context is done, a client error occurs
// or Joe is stopped.
//
// Subscribe returns without error only when the unsubscription is caused
// by the given context being canceled.
func (j *Joe) Subscribe(ctx context.Context, sub Subscription) error {
	j.init()

	// Without a buffered channel we risk a deadlock when Subscribe
	// stops receiving from this channel on done context and Joe
	// encounters an error when sending messages or replaying.
	done := make(chan error, 1)

	select {
	case <-j.done:
		return ErrProviderClosed
	case j.subscription <- subscription{done: done, Subscription: sub}:
	}

	select {
	case err := <-done:
		return err
	case <-j.closed:
		return ErrProviderClosed
	case <-ctx.Done():
	}

	select {
	case <-j.done:
		return ErrProviderClosed
	case j.unsubscription <- done:
		// NOTE(tmaxmax): should we return ctx.Err() instead?
		return nil
	}
}

// Publish tells Joe to send the given message to the subscribers.
// When a message is published to multiple topics, Joe makes sure to
// not send the Message multiple times to clients that are subscribed
// to more than one topic that receive the given Message. Every client
// receives each unique message once, regardless of how many topics it
// is subscribed to or to how many topics the message is published.
//
// It returns ErrNoTopic if no topics are provided, eventual Replayer.Put
// errors or ErrProviderClosed. If the replayer returns an error the
// message will still be sent but most probably it won't be replayed to
// new subscribers, depending on how the error is handled by the replay provider.
func (j *Joe) Publish(msg *Message, topics []string) error {
	if len(topics) == 0 {
		return ErrNoTopic
	}

	j.init()

	// Buffered to prevent a deadlock when Publish doesn't
	// receive from errs due to Joe being shut down and the
	// message published causes an error after the shutdown.
	errs := make(chan error, 1)

	pub := publishedMessage{replayerErr: errs}
	pub.message = msg
	pub.topics = topics

	// Waiting on done ensures Publish doesn't block the caller goroutine
	// when Joe is stopped and implements the required Provider behavior.
	select {
	case j.message <- pub:
		return <-errs
	case <-j.done:
		return ErrProviderClosed
	}
}

// Shutdown signals Joe to close all subscribers and stop receiving messages.
// It returns when all the subscribers are closed.
//
// Further calls to Stop will return ErrProviderClosed.
func (j *Joe) Shutdown(ctx context.Context) (err error) {
	j.init()

	defer func() {
		if r := recover(); r != nil {
			err = ErrProviderClosed
		}
	}()

	close(j.done)

	select {
	case <-j.closed:
	case <-ctx.Done():
		err = ctx.Err()
	}

	return
}

func (j *Joe) removeSubscriber(sub subscriber) {
	l := len(j.subscribers)
	delete(j.subscribers, sub)
	// We check that an element was deleted as removeSubscriber is called twice
	// in the following edge case: the subscriber context is done before a
	// published message is sent/flushed, and the send/flush returns an error.
	if l != len(j.subscribers) {
		close(sub)
	}
}

func (j *Joe) start(replay Replayer) {
	defer close(j.closed)

	for {
		select {
		case msg := <-j.message:
			if replay != nil {
				m, err := tryPut(msg.messageWithTopics, &replay)
				if _, isPanic := err.(replayPanic); err != nil && !isPanic { //nolint:errorlint // it's our error
					// NOTE(tmaxmax): We could return panic errors here but we'd have to expose
					// the error type in order for this error to be handled. Let's not change
					// the public errors for now. See also the other note below.
					msg.replayerErr <- err
				} else if m != nil {
					msg.message = m
				}
			}
			close(msg.replayerErr)

			for done, sub := range j.subscribers {
				if topicsIntersect(sub.Topics, msg.topics) {
					err := sub.Client.Send(msg.message)
					if err == nil {
						err = sub.Client.Flush()
					}

					if err != nil {
						done <- err
						// Technically it would be possible to just send the error,
						// as Subscribe would send an unsubscription signal. The problem
						// is that if the j.message channel is ready together with j.unsubscription
						// and j.message is picked we might send again to this now unsubscribed
						// subscriber, which will cause issues (e.g. deadlock on done).
						// This line here is the reason why we need to verify we actually
						// have this subscriber in removeSubscriber above.
						j.removeSubscriber(done)
					}
				}
			}
		case sub := <-j.subscription:
			var err error
			if replay != nil {
				err = tryReplay(sub.Subscription, &replay)
			}

			// NOTE(tmaxmax): We can't meaningfully handle replay panics in any way
			// other than disabling replay altogether. This ensures uptime
			// in the face of unexpected – returning the panic as an error
			// to the subscriber doesn't make sense, as it's probably not the subscriber's fault.
			if _, isPanic := err.(replayPanic); err != nil && !isPanic { //nolint:errorlint // it's our error
				sub.done <- err
				close(sub.done)
			} else {
				j.subscribers[sub.done] = sub.Subscription
			}
		case sub := <-j.unsubscription:
			j.removeSubscriber(sub)
		case <-j.done:
			return
		}
	}
}

func tryReplay(sub Subscription, replay *Replayer) (err error) { //nolint:gocritic // intended
	defer handleReplayerPanic(replay, &err)

	return (*replay).Replay(sub)
}

func tryPut(msg messageWithTopics, replay *Replayer) (m *Message, err error) { //nolint:gocritic // intended
	defer handleReplayerPanic(replay, &err)

	return (*replay).Put(msg.message, msg.topics)
}

type replayPanic struct{}

func (replayPanic) Error() string { return "replay provider panicked" }

func handleReplayerPanic(replay *Replayer, errp *error) { //nolint:gocritic // intended
	if r := recover(); r != nil {
		*replay = nil
		*errp = replayPanic{}
		// NOTE(tmaxmax): At least print a stacktrace. It's annoying when libraries recover from panics
		// and make them untraceable. Should we provide a way to handle these in a custom manner?
		debug.PrintStack()
	}
}

func (j *Joe) init() {
	j.initDone.Do(func() {
		j.message = make(chan publishedMessage)
		j.subscription = make(chan subscription)
		j.unsubscription = make(chan subscriber)
		j.done = make(chan struct{})
		j.closed = make(chan struct{})
		j.subscribers = map[subscriber]Subscription{}

		replay := j.Replayer
		if replay == nil {
			replay = noopReplayer{}
		}
		go j.start(replay)
	})
}
