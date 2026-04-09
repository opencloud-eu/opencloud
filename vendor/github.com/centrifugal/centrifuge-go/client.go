package centrifuge

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/centrifugal/protocol"
	"google.golang.org/protobuf/encoding/protojson"
)

// State of client connection.
type State string

// Describe client connection states.
const (
	StateDisconnected State = "disconnected"
	StateConnecting   State = "connecting"
	StateConnected    State = "connected"
	StateClosed       State = "closed"
)

// Client represents client connection to Centrifugo or Centrifuge
// library based server. It provides methods to set various event
// handlers, subscribe channels, call RPC commands etc. Call client
// Connect method to trigger actual connection with a server. Call client
// Close method to clean up state when you don't need client instance
// anymore.
type Client struct {
	futureID          uint64
	cmdID             uint32
	mu                sync.RWMutex
	endpoints         []string
	round             int
	protocolType      protocol.Type
	config            Config
	token             string
	data              protocol.Raw
	transport         transport
	disconnectedCh    chan struct{}
	state             State
	subs              map[string]*Subscription
	serverSubs        map[string]*serverSub
	requestsMu        sync.RWMutex
	requests          map[uint32]request
	receive           chan []byte
	reconnectAttempts int
	reconnectStrategy reconnectStrategy
	events            *eventHub
	sendPong          bool
	delayPing         chan struct{}
	closeCh           chan struct{}
	connectFutures    map[uint64]connectFuture
	cbQueue           *cbQueue
	reconnectTimer    *time.Timer
	refreshTimer      *time.Timer
	refreshRequired   bool
	logCh             chan LogEntry
	logCloseCh        chan struct{}
	logCloseOnce      sync.Once
}

// NewJsonClient initializes Client which uses JSON-based protocol internally.
// After client initialized call Client.Connect method. Use Client.NewSubscription to
// create Subscription objects.
// The provided endpoint must be a valid URL with ws:// or wss:// scheme – otherwise
// NewJsonClient will panic.
func NewJsonClient(endpoint string, config Config) *Client {
	return newClient(endpoint, false, config)
}

// NewProtobufClient initializes Client which uses Protobuf-based protocol internally.
// After client initialized call Client.Connect method. Use Client.NewSubscription to
// create Subscription objects.
// The provided endpoint must be a valid URL with ws:// or wss:// scheme – otherwise
// NewProtobufClient will panic.
func NewProtobufClient(endpoint string, config Config) *Client {
	return newClient(endpoint, true, config)
}

func (c *Client) logLevelEnabled(level LogLevel) bool {
	return c.config.LogLevel > LogLevelNone && level >= c.config.LogLevel
}

func (c *Client) log(level LogLevel, message string, fields map[string]string) {
	logEntry := LogEntry{
		Level:   level,
		Message: message,
		Fields:  fields,
	}
	select {
	case c.logCh <- logEntry:
	default:
		// If log channel is full, drop the log entry.
	}
}

func (c *Client) handleLogs() {
	for {
		select {
		case entry := <-c.logCh:
			c.config.LogHandler(entry)
		case <-c.logCloseCh:
			return
		}
	}
}

func (c *Client) traceOutCmd(cmd *protocol.Command) {
	jsonBytes, err := json.Marshal(cmd)
	if err != nil {
		jsonBytes, _ = protojson.Marshal(cmd)
	}
	c.log(LogLevelTrace, "-out->", map[string]string{"command": string(jsonBytes)})
}

func (c *Client) traceInReply(rep *protocol.Reply) {
	jsonBytes, err := json.Marshal(rep)
	if err != nil {
		jsonBytes, _ = protojson.Marshal(rep)
	}
	c.log(LogLevelTrace, "<-in--", map[string]string{"reply": string(jsonBytes)})
}

func (c *Client) traceInPush(push *protocol.Push) {
	jsonBytes, err := json.Marshal(push)
	if err != nil {
		jsonBytes, _ = protojson.Marshal(push)
	}
	c.log(LogLevelTrace, "<-in--", map[string]string{"push": string(jsonBytes)})
}

func newClient(endpoint string, isProtobuf bool, config Config) *Client {
	if config.ReadTimeout == 0 {
		config.ReadTimeout = 5 * time.Second
	}
	if config.WriteTimeout == 0 {
		config.WriteTimeout = time.Second
	}
	if config.HandshakeTimeout == 0 {
		config.HandshakeTimeout = time.Second
	}
	if config.MaxServerPingDelay == 0 {
		config.MaxServerPingDelay = 10 * time.Second
	}
	if config.Header == nil {
		config.Header = http.Header{}
	}
	if config.Name == "" {
		config.Name = "go"
	}

	// We support setting multiple endpoints to try in round-robin fashion. But
	// for now this feature is not documented and used for internal tests. In most
	// cases there should be a single public server WS endpoint.
	endpoints := strings.Split(endpoint, ",")
	if len(endpoints) == 0 {
		panic("connection endpoint required")
	}
	rand.Shuffle(len(endpoints), func(i, j int) {
		endpoints[i], endpoints[j] = endpoints[j], endpoints[i]
	})
	for _, e := range endpoints {
		if !strings.HasPrefix(e, "ws") {
			panic(fmt.Sprintf("unsupported connection endpoint: %s", e))
		}
	}

	protocolType := protocol.TypeJSON
	if isProtobuf {
		protocolType = protocol.TypeProtobuf
	}

	client := &Client{
		endpoints:         endpoints,
		config:            config,
		state:             StateDisconnected,
		protocolType:      protocolType,
		subs:              make(map[string]*Subscription),
		serverSubs:        make(map[string]*serverSub),
		requests:          make(map[uint32]request),
		reconnectStrategy: newBackoffReconnect(config.MinReconnectDelay, config.MaxReconnectDelay),
		delayPing:         make(chan struct{}, 32),
		events:            newEventHub(),
		connectFutures:    make(map[uint64]connectFuture),
		token:             config.Token,
		data:              config.Data,
		logCh:             make(chan LogEntry, 256),
		logCloseCh:        make(chan struct{}),
	}

	// Queue to run callbacks on.
	client.cbQueue = &cbQueue{
		closeCh: make(chan struct{}),
	}
	client.cbQueue.cond = sync.NewCond(&client.cbQueue.mu)
	go client.cbQueue.dispatch()
	if client.config.LogLevel > 0 {
		go client.handleLogs()
	}
	return client
}

// Connect dials to server and sends connect message. Will return an error if first
// dial with a server failed. In case of failure client will automatically reconnect.
// To temporary disconnect from a server call Client.Disconnect.
func (c *Client) Connect() error {
	return c.startConnecting()
}

// Disconnect client from server. It's still possible to connect again later. If
// you don't need Client anymore – use Client.Close.
func (c *Client) Disconnect() error {
	if c.isClosed() {
		return ErrClientClosed
	}
	c.moveToDisconnected(disconnectedDisconnectCalled, "disconnect called")
	return nil
}

// Close closes Client and cleanups resources. Client is unusable after this. Use this
// method if you don't need client anymore, otherwise look at Client.Disconnect.
func (c *Client) Close() {
	if c.isClosed() {
		return
	}
	c.moveToDisconnected(disconnectedDisconnectCalled, "disconnect called")
	c.moveToClosed()
	c.logCloseOnce.Do(func() {
		close(c.logCloseCh)
	})
}

// State returns current Client state. Note that while you are processing
// this state - Client can move to a new one.
func (c *Client) State() State {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.state
}

// SetToken allows updating Client's connection token.
func (c *Client) SetToken(token string) {
	c.mu.Lock()
	c.token = token
	c.mu.Unlock()
}

// NewSubscription allocates new Subscription on a channel. As soon as Subscription
// successfully created Client keeps reference to it inside internal map registry to
// manage automatic resubscribe on reconnect. After creating Subscription call its
// Subscription.Subscribe method to actually start subscribing process. To temporarily
// unsubscribe call Subscription.Unsubscribe. If you finished with Subscription then
// you can remove it from the internal registry by calling Client.RemoveSubscription
// method.
func (c *Client) NewSubscription(channel string, config ...SubscriptionConfig) (*Subscription, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	var sub *Subscription
	if _, ok := c.subs[channel]; ok {
		return nil, ErrDuplicateSubscription
	}
	sub = newSubscription(c, channel, config...)
	c.subs[channel] = sub
	return sub, nil
}

// RemoveSubscription removes Subscription from the internal client registry.
// Make sure Subscription is in unsubscribed state before removing it.
func (c *Client) RemoveSubscription(sub *Subscription) error {
	if sub.State() != SubStateUnsubscribed {
		return errors.New("subscription must be unsubscribed to be removed")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.subs, sub.Channel)
	return nil
}

// GetSubscription allows getting Subscription from the internal client registry.
func (c *Client) GetSubscription(channel string) (*Subscription, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	s, ok := c.subs[channel]
	return s, ok
}

// Subscriptions returns a map with all currently registered client-side subscriptions.
func (c *Client) Subscriptions() map[string]*Subscription {
	subs := make(map[string]*Subscription)
	c.mu.Lock()
	defer c.mu.Unlock()
	for k, v := range c.subs {
		subs[k] = v
	}
	return subs
}

// Send message to server without waiting for response.
// Message handler must be registered on server.
func (c *Client) Send(ctx context.Context, data []byte) error {
	if c.isClosed() {
		return ErrClientClosed
	}
	errCh := make(chan error, 1)
	c.onConnect(func(err error) {
		if err != nil {
			errCh <- err
			return
		}
		select {
		case <-ctx.Done():
			errCh <- ctx.Err()
			return
		default:
		}
		cmd := &protocol.Command{}
		params := &protocol.SendRequest{
			Data: data,
		}
		cmd.Send = params
		errCh <- c.send(cmd)
	})

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errCh:
		return err
	}
}

// RPCResult contains data returned from server as RPC result.
type RPCResult struct {
	Data []byte
}

// RPC allows sending data to a server and waiting for a response.
// RPC handler must be registered on server.
func (c *Client) RPC(ctx context.Context, method string, data []byte) (RPCResult, error) {
	if c.isClosed() {
		return RPCResult{}, ErrClientClosed
	}
	resCh := make(chan RPCResult, 1)
	errCh := make(chan error, 1)
	c.sendRPC(ctx, method, data, func(result RPCResult, err error) {
		resCh <- result
		errCh <- err
	})

	select {
	case <-ctx.Done():
		return RPCResult{}, ctx.Err()
	case res := <-resCh:
		return res, <-errCh
	}
}

func (c *Client) nextCmdID() uint32 {
	return atomic.AddUint32(&c.cmdID, 1)
}

func (c *Client) isConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.state == StateConnected
}

func (c *Client) isClosed() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.state == StateClosed
}

func (c *Client) isSubscribed(channel string) bool {
	c.mu.RLock()
	_, ok := c.subs[channel]
	c.mu.RUnlock()
	return ok
}

func (c *Client) sendRPC(ctx context.Context, method string, data []byte, fn func(RPCResult, error)) {
	c.onConnect(func(err error) {
		select {
		case <-ctx.Done():
			fn(RPCResult{}, ctx.Err())
			return
		default:
		}
		if err != nil {
			fn(RPCResult{}, err)
			return
		}
		cmd := &protocol.Command{
			Id: c.nextCmdID(),
		}

		params := &protocol.RPCRequest{
			Data:   data,
			Method: method,
		}

		cmd.Rpc = params

		err = c.sendAsync(cmd, func(r *protocol.Reply, err error) {
			if err != nil {
				fn(RPCResult{}, err)
				return
			}
			if r.Error != nil {
				fn(RPCResult{}, errorFromProto(r.Error))
				return
			}
			fn(RPCResult{Data: r.Rpc.Data}, nil)
		})
		if err != nil {
			fn(RPCResult{}, err)
			return
		}
	})
}

func (c *Client) moveToDisconnected(code uint32, reason string) {
	c.mu.Lock()
	if c.state == StateDisconnected || c.state == StateClosed {
		c.mu.Unlock()
		return
	}
	if c.transport != nil {
		_ = c.transport.Close()
		c.transport = nil
	}

	prevState := c.state
	c.state = StateDisconnected
	c.clearConnectedState()
	c.resolveConnectFutures(ErrClientDisconnected)

	subsToUnsubscribe := make([]*Subscription, 0, len(c.subs))
	for _, s := range c.subs {
		s.mu.Lock()
		if s.state == SubStateUnsubscribed {
			s.mu.Unlock()
			continue
		}
		s.mu.Unlock()
		subsToUnsubscribe = append(subsToUnsubscribe, s)
	}
	serverSubsToUnsubscribe := make([]string, 0, len(c.serverSubs))
	for ch := range c.serverSubs {
		serverSubsToUnsubscribe = append(serverSubsToUnsubscribe, ch)
	}
	c.mu.Unlock()

	for _, s := range subsToUnsubscribe {
		s.moveToSubscribing(subscribingTransportClosed, "transport closed")
	}

	if prevState == StateConnected {
		var serverSubscribingHandler ServerSubscribingHandler
		if c.events != nil && c.events.onServerSubscribing != nil {
			serverSubscribingHandler = c.events.onServerSubscribing
		}
		if serverSubscribingHandler != nil {
			c.runHandlerAsync(func() {
				for _, ch := range serverSubsToUnsubscribe {
					serverSubscribingHandler(ServerSubscribingEvent{Channel: ch})
				}
			})
		}
	}

	var handler DisconnectHandler
	if c.events != nil && c.events.onDisconnected != nil {
		handler = c.events.onDisconnected
	}
	if handler != nil {
		c.runHandlerAsync(func() {
			event := DisconnectedEvent{Code: code, Reason: reason}
			handler(event)
		})
	}
}

func (c *Client) moveToConnecting(code uint32, reason string) {
	if c.logLevelEnabled(LogLevelDebug) {
		c.log(LogLevelDebug, "moving client to connecting state", map[string]string{
			"code":   strconv.Itoa(int(code)),
			"reason": reason,
		})
	}
	c.mu.Lock()
	if c.state == StateDisconnected || c.state == StateClosed || c.state == StateConnecting {
		if c.logLevelEnabled(LogLevelDebug) {
			c.log(LogLevelDebug, "client already in state that does not require extra work", map[string]string{
				"state": string(c.state),
			})
		}
		c.mu.Unlock()
		return
	}
	if c.transport != nil {
		if c.logLevelEnabled(LogLevelDebug) {
			c.log(LogLevelDebug, "closing non-nil transport", nil)
		}
		_ = c.transport.Close()
		c.transport = nil
	}

	c.state = StateConnecting
	if c.logLevelEnabled(LogLevelDebug) {
		c.log(LogLevelDebug, "client moved to connecting state", nil)
	}
	c.clearConnectedState()
	if c.logLevelEnabled(LogLevelDebug) {
		c.log(LogLevelDebug, "cleared connected state", nil)
	}
	c.resolveConnectFutures(ErrClientDisconnected)
	if c.logLevelEnabled(LogLevelDebug) {
		c.log(LogLevelDebug, "resolved connect futures", nil)
	}

	subsToUnsubscribe := make([]*Subscription, 0, len(c.subs))
	for _, s := range c.subs {
		s.mu.Lock()
		if s.state == SubStateUnsubscribed {
			s.mu.Unlock()
			continue
		}
		s.mu.Unlock()
		subsToUnsubscribe = append(subsToUnsubscribe, s)
	}
	serverSubsToUnsubscribe := make([]string, 0, len(c.serverSubs))
	for ch := range c.serverSubs {
		serverSubsToUnsubscribe = append(serverSubsToUnsubscribe, ch)
	}
	c.mu.Unlock()

	for _, s := range subsToUnsubscribe {
		s.moveToSubscribing(subscribingTransportClosed, "transport closed")
	}
	if c.logLevelEnabled(LogLevelDebug) {
		c.log(LogLevelDebug, "client-side subs unsubscribe events called", map[string]string{
			"num_subs": strconv.Itoa(len(subsToUnsubscribe)),
		})
	}

	var serverSubscribingHandler ServerSubscribingHandler
	if c.events != nil && c.events.onServerSubscribing != nil {
		serverSubscribingHandler = c.events.onServerSubscribing
	}
	if serverSubscribingHandler != nil {
		c.runHandlerSync(func() {
			for _, ch := range serverSubsToUnsubscribe {
				serverSubscribingHandler(ServerSubscribingEvent{Channel: ch})
			}
		})
	}
	if c.logLevelEnabled(LogLevelDebug) {
		c.log(LogLevelDebug, "server-side subs unsubscribe events called", map[string]string{
			"num_subs": strconv.Itoa(len(serverSubsToUnsubscribe)),
		})
	}

	var handler ConnectingHandler
	if c.events != nil && c.events.onConnecting != nil {
		handler = c.events.onConnecting
	}
	if handler != nil {
		c.runHandlerSync(func() {
			event := ConnectingEvent{Code: code, Reason: reason}
			handler(event)
		})
	}
	if c.logLevelEnabled(LogLevelDebug) {
		c.log(LogLevelDebug, "connecting event called", nil)
	}

	c.mu.Lock()
	if c.state != StateConnecting {
		if c.logLevelEnabled(LogLevelDebug) {
			c.log(LogLevelDebug, "not in connecting state, no need to reconnect", map[string]string{
				"state": string(c.state),
			})
		}
		c.mu.Unlock()
		return
	}
	c.scheduleReconnectLocked()
	c.mu.Unlock()
}

func (c *Client) scheduleReconnectLocked() {
	c.reconnectAttempts++
	if c.reconnectTimer != nil {
		if c.logLevelEnabled(LogLevelDebug) {
			c.log(LogLevelDebug, "stopping previous reconnect timer", nil)
		}
		c.reconnectTimer.Stop()
		c.reconnectTimer = nil
	}
	reconnectDelay := c.getReconnectDelay()
	if c.logLevelEnabled(LogLevelDebug) {
		c.log(LogLevelDebug, "reconnect with delay", map[string]string{
			"delay": reconnectDelay.String(),
		})
	}
	c.reconnectTimer = time.AfterFunc(reconnectDelay, func() {
		if c.logLevelEnabled(LogLevelDebug) {
			c.log(LogLevelDebug, "reconnect timer fired, start reconnecting", nil)
		}
		_ = c.startReconnecting()
	})
}

func (c *Client) moveToClosed() {
	c.mu.Lock()
	if c.state == StateClosed {
		c.mu.Unlock()
		return
	}
	c.state = StateClosed

	subsToUnsubscribe := make([]*Subscription, 0, len(c.subs))
	for _, s := range c.subs {
		s.mu.Lock()
		if s.state == SubStateUnsubscribed {
			s.mu.Unlock()
			continue
		}
		s.mu.Unlock()
		subsToUnsubscribe = append(subsToUnsubscribe, s)
	}
	serverSubsToUnsubscribe := make([]string, 0, len(c.serverSubs))
	for ch := range c.serverSubs {
		serverSubsToUnsubscribe = append(serverSubsToUnsubscribe, ch)
	}
	c.mu.Unlock()

	for _, s := range subsToUnsubscribe {
		s.moveToUnsubscribed(unsubscribedClientClosed, "client closed")
	}

	var serverUnsubscribedHandler ServerUnsubscribedHandler
	if c.events != nil && c.events.onServerUnsubscribed != nil {
		serverUnsubscribedHandler = c.events.onServerUnsubscribed
	}
	if serverUnsubscribedHandler != nil {
		c.runHandlerAsync(func() {
			for _, ch := range serverSubsToUnsubscribe {
				serverUnsubscribedHandler(ServerUnsubscribedEvent{Channel: ch})
			}
		})
	}

	c.mu.RLock()
	disconnectedCh := c.disconnectedCh
	c.mu.RUnlock()
	// At this point connection close was issued, so we wait until the reader goroutine
	// finishes its work, after that it's safe to close the callback queue.
	if disconnectedCh != nil {
		<-disconnectedCh
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	c.disconnectedCh = nil
	c.cbQueue.close()
	c.cbQueue = nil
}

func (c *Client) handleError(err error) {
	var handler ErrorHandler
	if c.events != nil && c.events.onError != nil {
		handler = c.events.onError
	}
	if handler != nil {
		c.runHandlerSync(func() {
			handler(ErrorEvent{Error: err})
		})
	}
}

// Lock must be held outside.
func (c *Client) clearConnectedState() {
	if c.reconnectTimer != nil {
		c.reconnectTimer.Stop()
		c.reconnectTimer = nil
	}
	if c.refreshTimer != nil {
		c.refreshTimer.Stop()
		c.refreshTimer = nil
	}
	if c.closeCh != nil {
		close(c.closeCh)
		c.closeCh = nil
	}

	c.requestsMu.Lock()
	reqs := make(map[uint32]request, len(c.requests))
	for uid, req := range c.requests {
		reqs[uid] = req
	}
	c.requests = make(map[uint32]request)
	c.requestsMu.Unlock()

	for _, req := range reqs {
		if req.cb != nil {
			go req.cb(nil, ErrClientDisconnected)
		}
	}
}

func (c *Client) handleDisconnect(d *disconnect) {
	if d == nil {
		d = &disconnect{
			Code:      connectingTransportClosed,
			Reason:    "transport closed",
			Reconnect: true,
		}
	}
	if d.Reconnect {
		c.moveToConnecting(d.Code, d.Reason)
	} else {
		c.moveToDisconnected(d.Code, d.Reason)
	}
}

func (c *Client) waitServerPing(disconnectCh chan struct{}, pingInterval uint32) {
	timeout := c.config.MaxServerPingDelay + time.Duration(pingInterval)*time.Second
	for {
		select {
		case <-c.delayPing:
		case <-time.After(timeout):
			go c.handleDisconnect(&disconnect{Code: connectingNoPing, Reason: "no ping", Reconnect: true})
		case <-disconnectCh:
			return
		}
	}
}

func (c *Client) readOnce(t transport) error {
	reply, disconnect, err := t.Read()
	if err != nil {
		go c.handleDisconnect(disconnect)
		return err
	}
	c.handle(reply)
	return nil
}

func (c *Client) reader(t transport, disconnectCh chan struct{}) {
	defer close(disconnectCh)
	for {
		err := c.readOnce(t)
		if err != nil {
			return
		}
	}
}

func (c *Client) runHandlerSync(fn func()) {
	c.mu.RLock()
	queue := c.cbQueue
	c.mu.RUnlock()
	if queue == nil {
		return
	}
	waitCh := make(chan struct{})
	queue.push(func(delay time.Duration) {
		defer close(waitCh)
		fn()
	})
	<-waitCh
}

func (c *Client) runHandlerAsync(fn func()) {
	c.mu.RLock()
	queue := c.cbQueue
	c.mu.RUnlock()
	if queue == nil {
		return
	}
	queue.push(func(delay time.Duration) {
		fn()
	})
}

func (c *Client) handle(reply *protocol.Reply) {
	if reply.Id > 0 {
		if c.logLevelEnabled(LogLevelTrace) {
			c.traceInReply(reply)
		}
		c.requestsMu.RLock()
		req, ok := c.requests[reply.Id]
		c.requestsMu.RUnlock()
		if ok {
			if req.cb != nil {
				req.cb(reply, nil)
			}
		}
		c.removeRequest(reply.Id)
	} else {
		if reply.Push == nil {
			if c.logLevelEnabled(LogLevelTrace) {
				c.traceInReply(reply)
			}
			// Ping from server, send pong if needed.
			select {
			case c.delayPing <- struct{}{}:
			default:
			}
			c.mu.RLock()
			sendPong := c.sendPong
			c.mu.RUnlock()
			if sendPong {
				cmd := &protocol.Command{}
				_ = c.send(cmd)
			}
			return
		}
		if c.logLevelEnabled(LogLevelTrace) {
			c.traceInPush(reply.Push)
		}
		c.mu.Lock()
		if c.state != StateConnected {
			c.mu.Unlock()
			return
		}
		c.mu.Unlock()
		c.handlePush(reply.Push)
	}
}

func (c *Client) handleMessage(msg *protocol.Message) error {
	var handler MessageHandler
	if c.events != nil && c.events.onMessage != nil {
		handler = c.events.onMessage
	}
	if handler != nil {
		event := MessageEvent{Data: msg.Data}
		c.runHandlerSync(func() {
			handler(event)
		})
	}
	return nil
}

func (c *Client) handlePush(push *protocol.Push) {
	channel := push.Channel
	c.mu.RLock()
	sub, ok := c.subs[channel]
	c.mu.RUnlock()
	switch {
	case push.Message != nil:
		_ = c.handleMessage(push.Message)
	case push.Unsubscribe != nil:
		if !ok {
			c.handleServerUnsub(channel, push.Unsubscribe)
			return
		}
		sub.handleUnsubscribe(push.Unsubscribe)
	case push.Pub != nil:
		if !ok {
			c.handleServerPublication(channel, push.Pub)
			return
		}
		sub.handlePublication(push.Pub)
	case push.Join != nil:
		if !ok {
			c.handleServerJoin(channel, push.Join)
			return
		}
		sub.handleJoin(push.Join.Info)
	case push.Leave != nil:
		if !ok {
			c.handleServerLeave(channel, push.Leave)
			return
		}
		sub.handleLeave(push.Leave.Info)
	case push.Subscribe != nil:
		if ok {
			// Client-side subscription exists.
			return
		}
		c.handleServerSub(channel, push.Subscribe)
		return
	case push.Disconnect != nil:
		code := push.Disconnect.Code
		reconnect := code < 3500 || code >= 5000 || (code >= 4000 && code < 4500)
		if reconnect {
			c.moveToConnecting(code, push.Disconnect.Reason)
		} else {
			c.moveToDisconnected(code, push.Disconnect.Reason)
		}
	default:
	}
}

func (c *Client) handleServerPublication(channel string, pub *protocol.Publication) {
	c.mu.Lock()
	serverSub, ok := c.serverSubs[channel]
	if !ok {
		c.mu.Unlock()
		return
	}
	if serverSub.Recoverable && pub.Offset > 0 {
		serverSub.Offset = pub.Offset
	}
	c.mu.Unlock()

	var handler ServerPublicationHandler
	if c.events != nil && c.events.onServerPublication != nil {
		handler = c.events.onServerPublication
	}
	if handler != nil {
		c.runHandlerSync(func() {
			handler(ServerPublicationEvent{Channel: channel, Publication: pubFromProto(pub)})
		})
	}
}

func (c *Client) handleServerJoin(channel string, join *protocol.Join) {
	c.mu.Lock()
	_, ok := c.serverSubs[channel]
	if !ok {
		c.mu.Unlock()
		return
	}
	c.mu.Unlock()

	var handler ServerJoinHandler
	if c.events != nil && c.events.onServerJoin != nil {
		handler = c.events.onServerJoin
	}
	if handler != nil {
		c.runHandlerSync(func() {
			handler(ServerJoinEvent{Channel: channel, ClientInfo: infoFromProto(join.Info)})
		})
	}
}

func (c *Client) handleServerLeave(channel string, leave *protocol.Leave) {
	c.mu.Lock()
	_, ok := c.serverSubs[channel]
	if !ok {
		c.mu.Unlock()
		return
	}
	c.mu.Unlock()

	var handler ServerLeaveHandler
	if c.events != nil && c.events.onServerLeave != nil {
		handler = c.events.onServerLeave
	}
	if handler != nil {
		c.runHandlerSync(func() {
			handler(ServerLeaveEvent{Channel: channel, ClientInfo: infoFromProto(leave.Info)})
		})
	}
}

func (c *Client) handleServerSub(channel string, sub *protocol.Subscribe) {
	c.mu.Lock()
	_, ok := c.serverSubs[channel]
	if ok {
		c.mu.Unlock()
		return
	}
	c.serverSubs[channel] = &serverSub{
		Offset:      sub.Offset,
		Epoch:       sub.Epoch,
		Recoverable: sub.Recoverable,
	}
	c.mu.Unlock()

	var handler ServerSubscribedHandler
	if c.events != nil && c.events.onServerSubscribe != nil {
		handler = c.events.onServerSubscribe
	}
	if handler != nil {
		c.runHandlerSync(func() {
			ev := ServerSubscribedEvent{
				Channel:     channel,
				Positioned:  sub.GetPositioned(),
				Recoverable: sub.GetRecoverable(),
				Data:        sub.GetData(),
			}
			if ev.Positioned || ev.Recoverable {
				ev.StreamPosition = &StreamPosition{
					Epoch:  sub.GetEpoch(),
					Offset: sub.GetOffset(),
				}
			}
			handler(ev)
		})
	}
}

func (c *Client) handleServerUnsub(channel string, _ *protocol.Unsubscribe) {
	c.mu.Lock()
	_, ok := c.serverSubs[channel]
	if ok {
		delete(c.serverSubs, channel)
	}
	c.mu.Unlock()
	if !ok {
		return
	}

	var handler ServerUnsubscribedHandler
	if c.events != nil && c.events.onServerUnsubscribed != nil {
		handler = c.events.onServerUnsubscribed
	}
	if handler != nil {
		c.runHandlerSync(func() {
			handler(ServerUnsubscribedEvent{Channel: channel})
		})
	}
}

func (c *Client) getReconnectDelay() time.Duration {
	return c.reconnectStrategy.timeBeforeNextAttempt(c.reconnectAttempts)
}

func (c *Client) startReconnecting() error {
	c.mu.Lock()
	c.round++
	round := c.round
	if c.state != StateConnecting {
		if c.logLevelEnabled(LogLevelDebug) {
			c.log(LogLevelDebug, "not in connecting state, no need to reconnect", map[string]string{
				"state": string(c.state),
			})
		}
		c.mu.Unlock()
		return nil
	}
	refreshRequired := c.refreshRequired
	token := c.token
	getTokenFunc := c.config.GetToken
	c.mu.Unlock()

	wsConfig := websocketConfig{
		Proxy:             c.config.Proxy,
		NetDialContext:    c.config.NetDialContext,
		TLSConfig:         c.config.TLSConfig,
		HandshakeTimeout:  c.config.HandshakeTimeout,
		EnableCompression: c.config.EnableCompression,
		CookieJar:         c.config.CookieJar,
		Header:            c.config.Header,
	}

	u := c.endpoints[round%len(c.endpoints)]
	if c.logLevelEnabled(LogLevelDebug) {
		c.log(LogLevelDebug, "creating new transport", nil)
	}
	t, err := newWebsocketTransport(u, c.protocolType, wsConfig)
	if err != nil {
		if c.logLevelEnabled(LogLevelDebug) {
			c.log(LogLevelDebug, "error creating new transport", map[string]string{
				"error": err.Error(),
			})
		}
		c.handleError(TransportError{err})
		c.mu.Lock()
		if c.state != StateConnecting {
			if c.logLevelEnabled(LogLevelDebug) {
				c.log(LogLevelDebug, "not in connecting state, no need to reconnect", map[string]string{
					"state": string(c.state),
				})
			}
			c.mu.Unlock()
			return nil
		}
		c.scheduleReconnectLocked()
		c.mu.Unlock()
		return err
	}
	if c.logLevelEnabled(LogLevelDebug) {
		c.log(LogLevelDebug, "new transport created", nil)
	}

	if refreshRequired || (token == "" && getTokenFunc != nil) {
		// Try to refresh token.
		if c.logLevelEnabled(LogLevelDebug) {
			c.log(LogLevelDebug, "refreshing token", nil)
		}
		newToken, err := c.refreshToken()
		if err != nil {
			if errors.Is(err, ErrUnauthorized) {
				if c.logLevelEnabled(LogLevelDebug) {
					c.log(LogLevelDebug, "unauthorized error, move to disconnected", nil)
				}
				c.moveToDisconnected(disconnectedUnauthorized, "unauthorized")
				return nil
			}
			if c.logLevelEnabled(LogLevelDebug) {
				c.log(LogLevelDebug, "error refreshing token", map[string]string{
					"error": err.Error(),
				})
			}
			c.handleError(RefreshError{err})
			c.mu.Lock()
			if c.state != StateConnecting {
				if c.logLevelEnabled(LogLevelDebug) {
					c.log(LogLevelDebug, "not in connecting state, no need to continue", map[string]string{
						"state": string(c.state),
					})
				}
				_ = t.Close()
				c.mu.Unlock()
				return nil
			}
			c.scheduleReconnectLocked()
			c.mu.Unlock()
			return err
		} else {
			c.mu.Lock()
			c.token = newToken
			if c.state != StateConnecting {
				if c.logLevelEnabled(LogLevelDebug) {
					c.log(LogLevelDebug, "got token, but not in connecting state anymore", map[string]string{
						"state": string(c.state),
					})
				}
				c.mu.Unlock()
				return nil
			}
			c.mu.Unlock()
		}
	}

	c.mu.Lock()
	if c.state != StateConnecting {
		if c.logLevelEnabled(LogLevelDebug) {
			c.log(LogLevelDebug, "not in connecting state, no need to reconnect", map[string]string{
				"state": string(c.state),
			})
		}
		_ = t.Close()
		c.mu.Unlock()
		return nil
	}
	c.refreshRequired = false
	disconnectCh := make(chan struct{})
	c.receive = make(chan []byte, 64)
	c.transport = t
	c.disconnectedCh = disconnectCh

	go c.reader(t, disconnectCh)
	if c.logLevelEnabled(LogLevelDebug) {
		c.log(LogLevelDebug, "started reader loop, sending connect frame", nil)
	}
	err = c.sendConnect(func(res *protocol.ConnectResult, err error) {
		if c.logLevelEnabled(LogLevelDebug) {
			c.log(LogLevelDebug, "connect result received", nil)
		}
		c.mu.Lock()
		if c.state != StateConnecting {
			if c.logLevelEnabled(LogLevelDebug) {
				c.log(LogLevelDebug, "not in connecting state, no need to continue", map[string]string{
					"state": string(c.state),
				})
			}
			c.mu.Unlock()
			return
		}
		c.mu.Unlock()
		if err != nil {
			if c.logLevelEnabled(LogLevelDebug) {
				c.log(LogLevelDebug, "connect error", map[string]string{
					"error": err.Error(),
				})
			}
			c.handleError(ConnectError{err})
			_ = t.Close()
			if isTokenExpiredError(err) {
				c.mu.Lock()
				defer c.mu.Unlock()
				if c.state != StateConnecting {
					if c.logLevelEnabled(LogLevelDebug) {
						c.log(LogLevelDebug, "not in connecting state, no need to continue", map[string]string{
							"state": string(c.state),
						})
					}
					return
				}
				c.refreshRequired = true
				c.scheduleReconnectLocked()
				return
			} else if isServerError(err) && !isTemporaryError(err) {
				var serverError *Error
				if errors.As(err, &serverError) {
					if c.logLevelEnabled(LogLevelDebug) {
						c.log(LogLevelDebug, "server error, move to disconnected", map[string]string{
							"code":    strconv.Itoa(int(serverError.Code)),
							"message": serverError.Message,
						})
					}
					c.moveToDisconnected(serverError.Code, serverError.Message)
				} else {
					// Should not happen, but just in case.
					if c.logLevelEnabled(LogLevelDebug) {
						c.log(LogLevelDebug, "not a server error", map[string]string{
							"error": err.Error(),
						})
					}
				}
				return
			} else {
				c.mu.Lock()
				defer c.mu.Unlock()
				if c.state != StateConnecting {
					if c.logLevelEnabled(LogLevelDebug) {
						c.log(LogLevelDebug, "not in connecting state, no need to continue", map[string]string{
							"state": string(c.state),
						})
					}
					return
				}
				c.scheduleReconnectLocked()
				return
			}
		}
		c.mu.Lock()
		if c.state != StateConnecting {
			if c.logLevelEnabled(LogLevelDebug) {
				c.log(LogLevelDebug, "not in connecting state, no need to continue", map[string]string{
					"state": string(c.state),
				})
			}
			_ = t.Close()
			c.mu.Unlock()
			return
		}
		if c.logLevelEnabled(LogLevelDebug) {
			c.log(LogLevelDebug, "connect ok, move to connected", map[string]string{
				"client_id": res.Client,
			})
		}
		c.state = StateConnected

		if res.Expires {
			c.refreshTimer = time.AfterFunc(time.Duration(res.Ttl)*time.Second, c.sendRefresh)
		}
		c.resolveConnectFutures(nil)
		if c.logLevelEnabled(LogLevelDebug) {
			c.log(LogLevelDebug, "resolved connect futures", nil)
		}
		c.mu.Unlock()

		if c.events != nil && c.events.onConnected != nil {
			handler := c.events.onConnected
			ev := ConnectedEvent{
				ClientID: res.Client,
				Version:  res.Version,
				Data:     res.Data,
			}
			c.runHandlerSync(func() {
				handler(ev)
			})
		}
		if c.logLevelEnabled(LogLevelDebug) {
			c.log(LogLevelDebug, "connected event called", nil)
		}

		var subscribeHandler ServerSubscribedHandler
		if c.events != nil && c.events.onServerSubscribe != nil {
			subscribeHandler = c.events.onServerSubscribe
		}

		var publishHandler ServerPublicationHandler
		if c.events != nil && c.events.onServerPublication != nil {
			publishHandler = c.events.onServerPublication
		}

		for channel, subRes := range res.Subs {
			c.mu.Lock()
			sub, ok := c.serverSubs[channel]
			if ok {
				sub.Epoch = subRes.Epoch
				sub.Recoverable = subRes.Recoverable
			} else {
				sub = &serverSub{
					Epoch:       subRes.Epoch,
					Offset:      subRes.Offset,
					Recoverable: subRes.Recoverable,
				}
			}
			if len(subRes.Publications) == 0 {
				sub.Offset = subRes.Offset
			}
			c.serverSubs[channel] = sub
			c.mu.Unlock()

			if subscribeHandler != nil {
				c.runHandlerSync(func() {
					ev := ServerSubscribedEvent{
						Channel:       channel,
						Data:          subRes.GetData(),
						Recovered:     subRes.GetRecovered(),
						WasRecovering: subRes.GetWasRecovering(),
						Positioned:    subRes.GetPositioned(),
						Recoverable:   subRes.GetRecoverable(),
					}
					if ev.Positioned || ev.Recoverable {
						ev.StreamPosition = &StreamPosition{
							Epoch:  subRes.GetEpoch(),
							Offset: subRes.GetOffset(),
						}
					}
					subscribeHandler(ev)
				})
			}
			if publishHandler != nil {
				c.runHandlerSync(func() {
					for _, pub := range subRes.Publications {
						c.mu.Lock()
						if sub, ok := c.serverSubs[channel]; ok {
							sub.Offset = pub.Offset
						}
						c.serverSubs[channel] = sub
						c.mu.Unlock()
						publishHandler(ServerPublicationEvent{Channel: channel, Publication: pubFromProto(pub)})
					}
				})
			}
		}
		if c.logLevelEnabled(LogLevelDebug) {
			c.log(LogLevelDebug, "connect server-side subscriptions processed", nil)
		}

		for ch := range c.serverSubs {
			if _, ok := res.Subs[ch]; !ok {
				var serverUnsubscribedHandler ServerUnsubscribedHandler
				if c.events != nil && c.events.onServerUnsubscribed != nil {
					serverUnsubscribedHandler = c.events.onServerUnsubscribed
				}
				if serverUnsubscribedHandler != nil {
					c.runHandlerSync(func() {
						serverUnsubscribedHandler(ServerUnsubscribedEvent{Channel: ch})
					})
				}
			}
		}
		if c.logLevelEnabled(LogLevelDebug) {
			c.log(LogLevelDebug, "connect server-side unsubscriptions processed", nil)
		}

		c.mu.Lock()
		defer c.mu.Unlock()
		// Successfully connected – can reset reconnect attempts.
		c.reconnectAttempts = 0
		if c.logLevelEnabled(LogLevelDebug) {
			c.log(LogLevelDebug, "reset reconnect attempts counter", nil)
		}

		if c.state != StateConnected {
			if c.logLevelEnabled(LogLevelDebug) {
				c.log(LogLevelDebug, "not in connected state, no need to continue", map[string]string{
					"state": string(c.state),
				})
			}
			return
		}

		if res.Ping > 0 {
			if c.logLevelEnabled(LogLevelDebug) {
				c.log(LogLevelDebug, "start waiting server ping", map[string]string{
					"ping": strconv.Itoa(int(res.Ping)),
				})
			}
			c.sendPong = res.Pong
			go c.waitServerPing(disconnectCh, res.Ping)
		}
		c.resubscribe()
		if c.logLevelEnabled(LogLevelDebug) {
			c.log(LogLevelDebug, "client-side subscriptions resubscribe called", nil)
		}
	})
	if err != nil {
		if c.logLevelEnabled(LogLevelDebug) {
			c.log(LogLevelDebug, "error sending connect frame", map[string]string{
				"error": err.Error(),
			})
		}
		_ = t.Close()
		c.scheduleReconnectLocked()
	} else {
		if c.logLevelEnabled(LogLevelDebug) {
			c.log(LogLevelDebug, "connect frame successfully sent", nil)
		}
	}
	c.mu.Unlock()
	if err != nil {
		c.handleError(ConnectError{err})
	}
	return err
}

func (c *Client) startConnecting() error {
	c.mu.Lock()
	if c.state == StateClosed {
		c.mu.Unlock()
		return ErrClientClosed
	}
	if c.state == StateConnected || c.state == StateConnecting {
		c.mu.Unlock()
		return nil
	}
	if c.closeCh == nil {
		c.closeCh = make(chan struct{})
	}
	c.state = StateConnecting
	c.mu.Unlock()

	var handler ConnectingHandler
	if c.events != nil && c.events.onConnecting != nil {
		handler = c.events.onConnecting
	}
	if handler != nil {
		c.runHandlerSync(func() {
			event := ConnectingEvent{Code: connectingConnectCalled, Reason: "connect called"}
			handler(event)
		})
	}

	return c.startReconnecting()
}

func (c *Client) resubscribe() {
	for _, sub := range c.subs {
		sub.resubscribe()
	}
}

func isTokenExpiredError(err error) bool {
	if e, ok := err.(*Error); ok && e.Code == 109 {
		return true
	}
	return false
}

func isServerError(err error) bool {
	if _, ok := err.(*Error); ok {
		return true
	}
	return false
}

func isTemporaryError(err error) bool {
	if e, ok := err.(*Error); ok && e.Temporary {
		return true
	}
	return false
}

func (c *Client) refreshToken() (string, error) {
	handler := c.config.GetToken
	if handler == nil {
		c.handleError(ConfigurationError{Err: errors.New("GetToken must be set to handle expired token")})
		return "", ErrUnauthorized
	}
	return handler(ConnectionTokenEvent{})
}

func (c *Client) sendRefresh() {
	token, err := c.refreshToken()
	if err != nil {
		if errors.Is(err, ErrUnauthorized) {
			c.moveToDisconnected(disconnectedUnauthorized, "unauthorized")
			return
		}
		c.handleError(RefreshError{err})
		c.mu.Lock()
		defer c.mu.Unlock()
		c.handleRefreshError()
		return
	}
	c.mu.Lock()
	c.token = token
	c.mu.Unlock()

	cmd := &protocol.Command{
		Id: c.nextCmdID(),
	}
	params := &protocol.RefreshRequest{
		Token: c.token,
	}
	cmd.Refresh = params

	_ = c.sendAsync(cmd, func(r *protocol.Reply, err error) {
		if err != nil {
			c.handleError(RefreshError{err})
			c.mu.Lock()
			defer c.mu.Unlock()
			c.handleRefreshError()
			return
		}
		if r.Error != nil {
			c.mu.Lock()
			if c.state != StateConnected {
				c.mu.Unlock()
				return
			}
			if r.Error.Temporary {
				c.handleError(RefreshError{err})
				c.refreshTimer = time.AfterFunc(10*time.Second, c.sendRefresh)
				c.mu.Unlock()
			} else {
				c.mu.Unlock()
				c.moveToDisconnected(r.Error.Code, r.Error.Message)
			}
			return
		}
		expires := r.Refresh.Expires
		ttl := r.Refresh.Ttl
		if expires {
			c.mu.Lock()
			if c.state == StateConnected {
				c.refreshTimer = time.AfterFunc(time.Duration(ttl)*time.Second, c.sendRefresh)
			}
			c.mu.Unlock()
		}
	})
}

// Lock must be held outside.
func (c *Client) handleRefreshError() {
	if c.state != StateConnected {
		return
	}
	c.refreshTimer = time.AfterFunc(10*time.Second, c.sendRefresh)
}

func (c *Client) sendSubRefresh(channel string, token string, fn func(*protocol.SubRefreshResult, error)) {
	cmd := &protocol.Command{
		Id: c.nextCmdID(),
	}
	params := &protocol.SubRefreshRequest{
		Channel: channel,
		Token:   token,
	}
	cmd.SubRefresh = params

	_ = c.sendAsync(cmd, func(r *protocol.Reply, err error) {
		if err != nil {
			fn(nil, err)
			return
		}
		if r.Error != nil {
			fn(nil, errorFromProto(r.Error))
			return
		}
		fn(r.SubRefresh, nil)
	})
}

func (c *Client) sendConnect(fn func(*protocol.ConnectResult, error)) error {
	cmd := &protocol.Command{
		Id: c.nextCmdID(),
	}

	req := &protocol.ConnectRequest{}
	req.Token = c.token
	req.Name = c.config.Name
	req.Version = c.config.Version
	req.Data = c.data

	if len(c.serverSubs) > 0 {
		subs := make(map[string]*protocol.SubscribeRequest)
		for channel, serverSub := range c.serverSubs {
			if !serverSub.Recoverable {
				continue
			}
			subs[channel] = &protocol.SubscribeRequest{
				Recover: true,
				Epoch:   serverSub.Epoch,
				Offset:  serverSub.Offset,
			}
		}
		req.Subs = subs
	}
	cmd.Connect = req

	return c.sendAsync(cmd, func(reply *protocol.Reply, err error) {
		if err != nil {
			fn(nil, err)
			return
		}
		if reply.Error != nil {
			fn(nil, errorFromProto(reply.Error))
			return
		}
		fn(reply.Connect, nil)
	})
}

type StreamPosition struct {
	Offset uint64
	Epoch  string
}

func (c *Client) sendSubscribe(
	channel string, data []byte, recover bool, streamPos StreamPosition, token string,
	positioned bool, recoverable bool, joinLeave bool, deltaType DeltaType,
	fn func(res *protocol.SubscribeResult, err error),
) error {
	params := &protocol.SubscribeRequest{
		Channel: channel,
	}

	if recover {
		params.Recover = true
		if streamPos.Offset > 0 {
			params.Offset = streamPos.Offset
		}
		params.Epoch = streamPos.Epoch
	}
	params.Token = token
	params.Data = data
	params.Positioned = positioned
	params.Recoverable = recoverable
	params.JoinLeave = joinLeave

	if deltaType != DeltaTypeNone {
		params.Delta = string(deltaType)
	}

	cmd := &protocol.Command{
		Id: c.nextCmdID(),
	}
	cmd.Subscribe = params

	return c.sendAsync(cmd, func(reply *protocol.Reply, err error) {
		if err != nil {
			fn(nil, err)
			return
		}
		if reply.Error != nil {
			fn(nil, errorFromProto(reply.Error))
			return
		}
		fn(reply.Subscribe, nil)
	})
}

func (c *Client) nextFutureID() uint64 {
	return atomic.AddUint64(&c.futureID, 1)
}

type connectFuture struct {
	fn      func(error)
	closeCh chan struct{}
}

func newConnectFuture(fn func(error)) connectFuture {
	return connectFuture{fn: fn, closeCh: make(chan struct{})}
}

// lock must be held outside.
func (c *Client) resolveConnectFutures(err error) {
	for _, fut := range c.connectFutures {
		fut.fn(err)
		close(fut.closeCh)
	}
	c.connectFutures = make(map[uint64]connectFuture)
}

func (c *Client) onConnect(fn func(err error)) {
	c.mu.Lock()
	if c.state == StateConnected {
		c.mu.Unlock()
		fn(nil)
	} else if c.state == StateDisconnected {
		c.mu.Unlock()
		fn(ErrClientDisconnected)
	} else {
		defer c.mu.Unlock()
		id := c.nextFutureID()
		fut := newConnectFuture(fn)
		c.connectFutures[id] = fut
		go func() {
			select {
			case <-fut.closeCh:
			case <-time.After(c.config.ReadTimeout):
				c.mu.Lock()
				defer c.mu.Unlock()
				fut, ok := c.connectFutures[id]
				if !ok {
					return
				}
				delete(c.connectFutures, id)
				fut.fn(ErrTimeout)
			}
		}()
	}
}

// PublishResult contains the result of publish.
type PublishResult struct{}

// Publish data into channel.
func (c *Client) Publish(ctx context.Context, channel string, data []byte) (PublishResult, error) {
	if c.isClosed() {
		return PublishResult{}, ErrClientClosed
	}
	resCh := make(chan PublishResult, 1)
	errCh := make(chan error, 1)
	c.publish(ctx, channel, data, func(result PublishResult, err error) {
		resCh <- result
		errCh <- err
	})
	select {
	case <-ctx.Done():
		return PublishResult{}, ctx.Err()
	case res := <-resCh:
		return res, <-errCh
	}
}

func (c *Client) publish(ctx context.Context, channel string, data []byte, fn func(PublishResult, error)) {
	c.onConnect(func(err error) {
		select {
		case <-ctx.Done():
			fn(PublishResult{}, ctx.Err())
			return
		default:
		}
		if err != nil {
			fn(PublishResult{}, err)
			return
		}
		c.sendPublish(channel, data, fn)
	})
}

func (c *Client) sendPublish(channel string, data []byte, fn func(PublishResult, error)) {
	params := &protocol.PublishRequest{
		Channel: channel,
		Data:    protocol.Raw(data),
	}

	cmd := &protocol.Command{
		Id: c.nextCmdID(),
	}
	cmd.Publish = params
	err := c.sendAsync(cmd, func(r *protocol.Reply, err error) {
		if err != nil {
			fn(PublishResult{}, err)
			return
		}
		if r.Error != nil {
			fn(PublishResult{}, errorFromProto(r.Error))
			return
		}
		fn(PublishResult{}, nil)
	})
	if err != nil {
		fn(PublishResult{}, err)
	}
}

// HistoryResult contains the result of history op.
type HistoryResult struct {
	Publications []Publication
	Offset       uint64
	Epoch        string
}

// History for a channel without being subscribed.
func (c *Client) History(ctx context.Context, channel string, opts ...HistoryOption) (HistoryResult, error) {
	if c.isClosed() {
		return HistoryResult{}, ErrClientClosed
	}
	resCh := make(chan HistoryResult, 1)
	errCh := make(chan error, 1)
	historyOpts := &HistoryOptions{}
	for _, opt := range opts {
		opt(historyOpts)
	}
	c.history(ctx, channel, *historyOpts, func(result HistoryResult, err error) {
		resCh <- result
		errCh <- err
	})
	select {
	case <-ctx.Done():
		return HistoryResult{}, ctx.Err()
	case res := <-resCh:
		return res, <-errCh
	}
}

func (c *Client) history(ctx context.Context, channel string, opts HistoryOptions, fn func(HistoryResult, error)) {
	c.onConnect(func(err error) {
		select {
		case <-ctx.Done():
			fn(HistoryResult{}, ctx.Err())
			return
		default:
		}
		if err != nil {
			fn(HistoryResult{}, err)
			return
		}
		c.sendHistory(channel, opts, fn)
	})
}

func (c *Client) sendHistory(channel string, opts HistoryOptions, fn func(HistoryResult, error)) {
	params := &protocol.HistoryRequest{
		Channel: channel,
		Limit:   opts.Limit,
		Reverse: opts.Reverse,
	}
	if opts.Since != nil {
		params.Since = &protocol.StreamPosition{
			Offset: opts.Since.Offset,
			Epoch:  opts.Since.Epoch,
		}
	}

	cmd := &protocol.Command{
		Id: c.nextCmdID(),
	}
	cmd.History = params

	err := c.sendAsync(cmd, func(r *protocol.Reply, err error) {
		if err != nil {
			fn(HistoryResult{}, err)
			return
		}
		if r.Error != nil {
			fn(HistoryResult{}, errorFromProto(r.Error))
			return
		}

		publications := r.History.Publications
		offset := r.History.Offset
		epoch := r.History.Epoch

		pubs := make([]Publication, len(publications))
		for i, m := range publications {
			pubs[i] = pubFromProto(m)
		}
		fn(HistoryResult{
			Publications: pubs,
			Offset:       offset,
			Epoch:        epoch,
		}, nil)
	})
	if err != nil {
		fn(HistoryResult{}, err)
		return
	}
}

// PresenceResult contains the result of presence op.
type PresenceResult struct {
	Clients map[string]ClientInfo
}

// Presence for a channel without being subscribed.
func (c *Client) Presence(ctx context.Context, channel string) (PresenceResult, error) {
	if c.isClosed() {
		return PresenceResult{}, ErrClientClosed
	}
	resCh := make(chan PresenceResult, 1)
	errCh := make(chan error, 1)
	c.presence(ctx, channel, func(result PresenceResult, err error) {
		resCh <- result
		errCh <- err
	})
	select {
	case <-ctx.Done():
		return PresenceResult{}, ctx.Err()
	case res := <-resCh:
		return res, <-errCh
	}
}

func (c *Client) presence(ctx context.Context, channel string, fn func(PresenceResult, error)) {
	c.onConnect(func(err error) {
		select {
		case <-ctx.Done():
			fn(PresenceResult{}, ctx.Err())
			return
		default:
		}
		if err != nil {
			fn(PresenceResult{}, err)
			return
		}
		c.sendPresence(channel, fn)
	})
}

func (c *Client) sendPresence(channel string, fn func(PresenceResult, error)) {
	params := &protocol.PresenceRequest{
		Channel: channel,
	}

	cmd := &protocol.Command{
		Id: c.nextCmdID(),
	}
	cmd.Presence = params

	err := c.sendAsync(cmd, func(r *protocol.Reply, err error) {
		if err != nil {
			fn(PresenceResult{}, err)
			return
		}
		if r.Error != nil {
			fn(PresenceResult{}, errorFromProto(r.Error))
			return
		}

		p := make(map[string]ClientInfo)

		for uid, info := range r.Presence.Presence {
			p[uid] = infoFromProto(info)
		}
		fn(PresenceResult{Clients: p}, nil)
	})
	if err != nil {
		fn(PresenceResult{}, err)
	}
}

// PresenceStats represents short presence information.
type PresenceStats struct {
	NumClients int
	NumUsers   int
}

// PresenceStatsResult wraps presence stats.
type PresenceStatsResult struct {
	PresenceStats
}

// PresenceStats for a channel without being subscribed.
func (c *Client) PresenceStats(ctx context.Context, channel string) (PresenceStatsResult, error) {
	if c.isClosed() {
		return PresenceStatsResult{}, ErrClientClosed
	}
	resCh := make(chan PresenceStatsResult, 1)
	errCh := make(chan error, 1)
	c.presenceStats(ctx, channel, func(result PresenceStatsResult, err error) {
		resCh <- result
		errCh <- err
	})
	select {
	case <-ctx.Done():
		return PresenceStatsResult{}, ctx.Err()
	case res := <-resCh:
		return res, <-errCh
	}
}

func (c *Client) presenceStats(ctx context.Context, channel string, fn func(PresenceStatsResult, error)) {
	c.onConnect(func(err error) {
		select {
		case <-ctx.Done():
			fn(PresenceStatsResult{}, ctx.Err())
			return
		default:
		}
		if err != nil {
			fn(PresenceStatsResult{}, err)
			return
		}
		c.sendPresenceStats(channel, fn)
	})
}

func (c *Client) sendPresenceStats(channel string, fn func(PresenceStatsResult, error)) {
	params := &protocol.PresenceStatsRequest{
		Channel: channel,
	}

	cmd := &protocol.Command{
		Id: c.nextCmdID(),
	}
	cmd.PresenceStats = params

	err := c.sendAsync(cmd, func(r *protocol.Reply, err error) {
		if err != nil {
			fn(PresenceStatsResult{}, err)
			return
		}
		if r.Error != nil {
			fn(PresenceStatsResult{}, errorFromProto(r.Error))
			return
		}
		fn(PresenceStatsResult{PresenceStats{
			NumClients: int(r.PresenceStats.NumClients),
			NumUsers:   int(r.PresenceStats.NumUsers),
		}}, nil)
	})
	if err != nil {
		fn(PresenceStatsResult{}, err)
		return
	}
}

type UnsubscribeResult struct{}

func (c *Client) unsubscribe(channel string, fn func(UnsubscribeResult, error)) {
	if !c.isSubscribed(channel) {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.state != StateConnected {
		return
	}
	c.sendUnsubscribe(channel, fn)
}

func (c *Client) sendUnsubscribe(channel string, fn func(UnsubscribeResult, error)) {
	params := &protocol.UnsubscribeRequest{
		Channel: channel,
	}

	cmd := &protocol.Command{
		Id: c.nextCmdID(),
	}
	cmd.Unsubscribe = params

	err := c.sendAsync(cmd, func(r *protocol.Reply, err error) {
		if err != nil {
			fn(UnsubscribeResult{}, err)
			return
		}
		if r.Error != nil {
			fn(UnsubscribeResult{}, errorFromProto(r.Error))
			return
		}
		fn(UnsubscribeResult{}, nil)
	})
	if err != nil {
		fn(UnsubscribeResult{}, err)
	}
}

func (c *Client) sendAsync(cmd *protocol.Command, cb func(*protocol.Reply, error)) error {
	c.addRequest(cmd.Id, cb)

	err := c.send(cmd)
	if err != nil {
		return err
	}
	go func() {
		c.mu.Lock()
		closeCh := c.closeCh
		c.mu.Unlock()
		defer c.removeRequest(cmd.Id)
		select {
		case <-time.After(c.config.ReadTimeout):
			c.requestsMu.RLock()
			req, ok := c.requests[cmd.Id]
			c.requestsMu.RUnlock()
			if !ok {
				return
			}
			req.cb(nil, ErrTimeout)
		case <-closeCh:
			c.requestsMu.RLock()
			req, ok := c.requests[cmd.Id]
			c.requestsMu.RUnlock()
			if !ok {
				return
			}
			req.cb(nil, ErrClientDisconnected)
		}
	}()
	return nil
}

func (c *Client) send(cmd *protocol.Command) error {
	transport := c.transport
	if transport == nil {
		return ErrClientDisconnected
	}
	if c.logLevelEnabled(LogLevelTrace) {
		c.traceOutCmd(cmd)
	}
	err := transport.Write(cmd, c.config.WriteTimeout)
	if err != nil {
		go c.handleDisconnect(&disconnect{Code: connectingTransportClosed, Reason: "write error", Reconnect: true})
		return io.EOF
	}
	return nil
}

type request struct {
	cb func(*protocol.Reply, error)
}

func (c *Client) addRequest(id uint32, cb func(*protocol.Reply, error)) {
	c.requestsMu.Lock()
	defer c.requestsMu.Unlock()
	c.requests[id] = request{cb}
}

func (c *Client) removeRequest(id uint32) {
	c.requestsMu.Lock()
	defer c.requestsMu.Unlock()
	delete(c.requests, id)
}

type disconnect struct {
	Code      uint32
	Reason    string
	Reconnect bool
}

type serverSub struct {
	Offset      uint64
	Epoch       string
	Recoverable bool
}
