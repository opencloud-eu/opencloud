package sse

import (
	"errors"
	"strconv"
	"time"
)

// NewFiniteReplayer creates a finite replay provider with the given max
// count and auto ID behaviour.
//
// Count is the maximum number of events FiniteReplayer should hold as
// valid. It must be greater than zero.
//
// AutoIDs configures FiniteReplayer to automatically set the IDs of
// events.
func NewFiniteReplayer(
	count int, autoIDs bool,
) (*FiniteReplayer, error) {
	if count < 2 {
		return nil, errors.New("count must be at least 2")
	}

	r := &FiniteReplayer{}
	r.buf.buf = make([]messageWithTopics, count)
	if autoIDs {
		r.currentID = new(uint64)
	}

	return r, nil
}

// FiniteReplayer is a replayer that replays at maximum a certain number of events.
// The events must have an ID unless the replayer is configured to set IDs automatically.
type FiniteReplayer struct {
	currentID *uint64
	buf       queue[messageWithTopics]
}

// Put puts a message into the replayer's buffer. If there are more messages than the maximum
// number, the oldest message is removed.
func (f *FiniteReplayer) Put(message *Message, topics []string) (*Message, error) {
	if len(topics) == 0 {
		return nil, ErrNoTopic
	}

	message, err := ensureID(message, f.currentID)
	if err != nil {
		return nil, err
	}

	f.buf.enqueue(messageWithTopics{message: message, topics: topics})

	return message, nil
}

// Replay replays the stored messages to the listener.
func (f *FiniteReplayer) Replay(subscription Subscription) error {
	i := findIDInQueue(&f.buf, subscription.LastEventID, f.currentID != nil)
	if i < 0 {
		return nil
	}

	var err error
	f.buf.each(i)(func(_ int, m messageWithTopics) bool {
		if topicsIntersect(subscription.Topics, m.topics) {
			if err = subscription.Client.Send(m.message); err != nil {
				return false
			}
		}
		return true
	})
	if err != nil {
		return err
	}

	return subscription.Client.Flush()
}

// ValidReplayer is a Replayer that replays all the buffered non-expired events.
//
// The replayer removes any expired events when a new event is put and after at least
// a GCInterval period passed.
//
// The events must have an ID unless the replayer is configured to set IDs automatically.
type ValidReplayer struct {
	lastGC time.Time

	// The function used to retrieve the current time. Defaults to time.Now.
	// Useful when testing.
	Now func() time.Time

	currentID *uint64
	messages  queue[messageWithTopicsAndExpiry]

	ttl time.Duration
	// After how long the replayer should attempt to clean up expired events.
	// By default cleanup is done after a fourth of the TTL has passed; this means
	// that messages may be stored for a duration equal to 5/4*TTL. If this is not
	// desired, set the GC interval to a value sensible for your use case or set
	// it to 0 – this disables automatic cleanup, enabling you to do it manually
	// using the GC method.
	GCInterval time.Duration
}

// NewValidReplayer creates a ValidReplayer with the given message
// lifetime duration (time-to-live) and auto ID behavior.
//
// The TTL must be a positive duration. It is technically possible to use a very
// big duration in order to store and replay every message put for the lifetime
// of the program; this is not recommended, as memory usage becomes effectively
// unbounded which might lead to a crash.
func NewValidReplayer(ttl time.Duration, autoIDs bool) (*ValidReplayer, error) {
	if ttl <= 0 {
		return nil, errors.New("event TTL must be greater than zero")
	}

	r := &ValidReplayer{
		Now:        time.Now,
		GCInterval: ttl / 4,
		ttl:        ttl,
	}

	if autoIDs {
		r.currentID = new(uint64)
	}

	return r, nil
}

// Put puts the message into the replayer's buffer.
func (v *ValidReplayer) Put(message *Message, topics []string) (*Message, error) {
	if len(topics) == 0 {
		return nil, ErrNoTopic
	}

	now := v.Now()
	if v.lastGC.IsZero() {
		v.lastGC = now
	}

	if v.shouldGC(now) {
		v.doGC(now)
		v.lastGC = now
	}

	message, err := ensureID(message, v.currentID)
	if err != nil {
		return nil, err
	}

	if v.messages.count == len(v.messages.buf) {
		newCap := len(v.messages.buf) * 2
		if minCap := 4; newCap < minCap {
			newCap = minCap
		}
		v.messages.resize(newCap)
	}

	v.messages.enqueue(messageWithTopicsAndExpiry{messageWithTopics: messageWithTopics{message: message, topics: topics}, exp: now.Add(v.ttl)})

	return message, nil
}

func (v *ValidReplayer) shouldGC(now time.Time) bool {
	return v.GCInterval > 0 && now.Sub(v.lastGC) >= v.GCInterval
}

// GC removes all the expired messages from the replayer's buffer.
func (v *ValidReplayer) GC() {
	v.doGC(v.Now())
}

func (v *ValidReplayer) doGC(now time.Time) {
	for v.messages.count > 0 {
		e := v.messages.buf[v.messages.head]
		if e.exp.After(now) {
			break
		}

		v.messages.dequeue()
	}

	if v.messages.count <= len(v.messages.buf)/4 {
		newCap := len(v.messages.buf) / 2
		if minCap := 4; newCap < minCap {
			newCap = minCap
		}
		v.messages.resize(newCap)
	}
}

// Replay replays all the valid messages to the listener.
func (v *ValidReplayer) Replay(subscription Subscription) error {
	i := findIDInQueue(&v.messages, subscription.LastEventID, v.currentID != nil)
	if i < 0 {
		return nil
	}

	now := v.Now()

	var err error
	v.messages.each(i)(func(_ int, m messageWithTopicsAndExpiry) bool {
		if m.exp.After(now) && topicsIntersect(subscription.Topics, m.topics) {
			if err = subscription.Client.Send(m.message); err != nil {
				return false
			}
		}
		return true
	})
	if err != nil {
		return err
	}

	return subscription.Client.Flush()
}

// topicsIntersect returns true if the given topic slices have at least one topic in common.
func topicsIntersect(a, b []string) bool {
	for _, at := range a {
		for _, bt := range b {
			if at == bt {
				return true
			}
		}
	}

	return false
}

func ensureID(m *Message, currentID *uint64) (*Message, error) {
	if currentID == nil {
		if !m.ID.IsSet() {
			return nil, errors.New("message has no ID")
		}

		return m, nil
	}

	if m.ID.IsSet() {
		return nil, errors.New("message already has an ID, can't use generated ID")
	}

	m = m.Clone()
	m.ID = ID(strconv.FormatUint(*currentID, 10))

	(*currentID)++

	return m, nil
}

type queue[T any] struct {
	buf               []T
	head, tail, count int
}

func (q *queue[T]) each(startAt int) func(func(int, T) bool) {
	return func(yield func(int, T) bool) {
		if startAt < q.tail {
			for i := startAt; i < q.tail; i++ {
				if !yield(i, q.buf[i]) {
					return
				}
			}
		} else {
			for i := startAt; i < len(q.buf); i++ {
				if !yield(i, q.buf[i]) {
					return
				}
			}
			for i := 0; i < q.tail; i++ {
				if !yield(i, q.buf[i]) {
					return
				}
			}
		}
	}
}

func (q *queue[T]) enqueue(v T) {
	q.buf[q.tail] = v

	q.tail++

	overwritten := false
	if q.tail > q.head && q.count == len(q.buf) {
		q.head = q.tail
		overwritten = true
	} else {
		q.count++
	}

	if q.tail == len(q.buf) {
		q.tail = 0
		if overwritten {
			q.head = 0
		}
	}
}

func (q *queue[T]) dequeue() {
	q.buf[q.head] = *new(T)

	q.head++
	if q.head == len(q.buf) {
		q.head = 0
	}

	q.count--
}

func (q *queue[T]) resize(newSize int) {
	buf := make([]T, newSize)
	if q.head < q.tail {
		copy(buf, q.buf[q.head:q.tail])
	} else {
		n := copy(buf, q.buf[q.head:])
		copy(buf[n:], q.buf[:q.tail])
	}

	q.head = 0
	q.tail = q.count
	q.buf = buf
}

func findIDInQueue[M interface{ ID() EventID }](q *queue[M], id EventID, autoID bool) int {
	if q.count == 0 {
		return -1
	}

	if autoID {
		id, err := strconv.ParseUint(id.String(), 10, 64)
		if err != nil {
			return -1
		}

		firstID, _ := strconv.ParseUint(q.buf[q.head].ID().String(), 10, 64)

		pos := -1
		if delta := id - firstID; id >= firstID {
			if delta >= uint64(q.count) { //nolint:gosec // int always positive
				return -1
			}
			pos = int(delta) //nolint:gosec // delta < q.count, which is an int
		}

		i := pos + q.head + 1
		if i >= len(q.buf) {
			i -= len(q.buf)
		}

		return i
	}

	i := -1
	q.each(q.head)(func(j int, m M) bool {
		if m.ID() == id {
			i = j
			return false
		}
		return true
	})

	if i != -1 {
		i++
		if i == len(q.buf) {
			i = 0
		} else if i == q.tail {
			i = -1
		}
	}

	return i
}

func (m messageWithTopics) ID() EventID { return m.message.ID }

type messageWithTopicsAndExpiry struct {
	exp time.Time
	messageWithTopics
}

// noopReplayer is the default replay provider used if none is given. It does nothing.
// It is used to avoid nil checks for the provider each time it is used.
type noopReplayer struct{}

func (n noopReplayer) Put(m *Message, _ []string) (*Message, error) { return m, nil }
func (n noopReplayer) Replay(_ Subscription) error                  { return nil }
