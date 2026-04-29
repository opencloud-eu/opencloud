package jmap

import (
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/structs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testWsPushListener struct {
	t             *testing.T
	logger        *log.Logger
	username      string
	mailAccountId string
	calls         atomic.Uint32
	m             sync.Mutex
	emailStates   []string
	threadStates  []string
	mailboxStates []string
}

func (l *testWsPushListener) OnNotification(username string, pushState StateChange) {
	assert.Equal(l.t, l.username, username)
	l.calls.Add(1)
	// pushState is currently not supported by Stalwart, let's use the object states instead
	l.logger.Debug().Msgf("received %T: %v", pushState, pushState)
	if changed, ok := pushState.Changed[l.mailAccountId]; ok {
		l.m.Lock()
		if st, ok := changed[EmailName]; ok {
			l.emailStates = append(l.emailStates, st)
		}
		if st, ok := changed[ThreadName]; ok {
			l.threadStates = append(l.threadStates, st)
		}
		if st, ok := changed[MailboxName]; ok {
			l.mailboxStates = append(l.mailboxStates, st)
		}
		l.m.Unlock()

		unsupportedKeys := structs.Filter(structs.Keys(changed), func(o ObjectTypeName) bool { return o != EmailName && o != ThreadName && o != MailboxName })
		assert.Empty(l.t, unsupportedKeys)
	}
	unsupportedAccounts := structs.Filter(structs.Keys(pushState.Changed), func(s string) bool { return s != l.mailAccountId })
	assert.Empty(l.t, unsupportedAccounts)
}

var _ WsPushListener = &testWsPushListener{}

func TestWs(t *testing.T) {
	if skip(t) {
		return
	}

	assert.NoError(t, nil)

	require := require.New(t)

	cotx := t.Context()

	s, err := newStalwartTest(t)
	require.NoError(err)
	defer s.Close()

	user := pickUser()
	session := s.Session(user.name)
	ctx := s.Context(session)

	mailAccountId := session.PrimaryAccounts.Mail
	inboxFolder := ""
	{
		_, inboxFolder = s.findInbox(t, mailAccountId, ctx)
	}

	l := &testWsPushListener{t: t, username: user.name, logger: s.logger, mailAccountId: mailAccountId}
	s.client.AddWsPushListener(l)

	require.Equal(uint32(0), l.calls.Load())
	{
		l.m.Lock()
		require.Len(l.emailStates, 0)
		require.Len(l.mailboxStates, 0)
		require.Len(l.threadStates, 0)
		l.m.Unlock()
	}

	var initialState State
	{
		result, err := s.client.GetEmailChanges(mailAccountId, EmptyState, true, 0, 0, ctx)
		require.NoError(err)
		require.Equal(session.State, result.GetSessionState())
		require.NotEmpty(result.GetState())
		//fmt.Printf("\x1b[45;1;4mChanges [%s]:\x1b[0m\n", state)
		//for _, c := range changes.Created { fmt.Printf("%s %s\n", c.Id, c.Subject) }
		initialState = result.GetState()
		require.Empty(result.Payload.Created)
		require.Empty(result.Payload.Destroyed)
		require.Empty(result.Payload.Updated)
	}
	require.NotEmpty(initialState)

	{
		result, err := s.client.GetEmailChanges(mailAccountId, initialState, true, 0, 0, ctx)
		require.NoError(err)
		require.Equal(session.State, result.GetSessionState())
		require.Equal(initialState, result.GetState())
		require.Equal(initialState, result.Payload.NewState)
		require.Empty(result.Payload.Created)
		require.Empty(result.Payload.Destroyed)
		require.Empty(result.Payload.Updated)
	}

	wsc, err := s.client.EnablePushNotifications(cotx, initialState, func() (*Session, error) { return session, nil })
	require.NoError(err)
	defer wsc.Close()

	require.Equal(uint32(0), l.calls.Load())
	{
		l.m.Lock()
		require.Len(l.emailStates, 0)
		require.Len(l.mailboxStates, 0)
		require.Len(l.threadStates, 0)
		l.m.Unlock()
	}

	emailIds := []string{}

	{
		_, n, err := s.fillEmailsWithImap(inboxFolder, 1, false, user)
		require.NoError(err)
		require.Equal(1, n)
	}

	require.Eventually(func() bool {
		return l.calls.Load() == uint32(1)
	}, 3*time.Second, 200*time.Millisecond, "WS push listener was not called after first email state change")
	{
		l.m.Lock()
		require.Len(l.emailStates, 1)
		require.Len(l.mailboxStates, 1)
		require.Len(l.threadStates, 1)
		l.m.Unlock()
	}
	var lastState State
	{
		result, err := s.client.GetEmailChanges(mailAccountId, initialState, true, 0, 0, ctx)
		require.NoError(err)
		require.Equal(session.State, result.GetSessionState())
		require.NotEqual(initialState, result.GetState())
		require.NotEqual(initialState, result.Payload.NewState)
		require.Equal(result.GetState(), result.Payload.NewState)
		require.Len(result.Payload.Created, 1)
		require.Empty(result.Payload.Destroyed)
		require.Empty(result.Payload.Updated)
		lastState = result.GetState()

		emailIds = append(emailIds, structs.Map(result.Payload.Created, func(e Email) string { return e.Id })...)
	}

	{
		_, n, err := s.fillEmailsWithImap(inboxFolder, 1, false, user)
		require.NoError(err)
		require.Equal(1, n)
	}

	require.Eventually(func() bool {
		return l.calls.Load() == uint32(2)
	}, 3*time.Second, 200*time.Millisecond, "WS push listener was not called after second email state change")
	{
		l.m.Lock()
		require.Len(l.emailStates, 2)
		require.Len(l.mailboxStates, 2)
		require.Len(l.threadStates, 2)
		assert.NotEqual(t, l.emailStates[0], l.emailStates[1])
		assert.NotEqual(t, l.mailboxStates[0], l.mailboxStates[1])
		assert.NotEqual(t, l.threadStates[0], l.threadStates[1])
		l.m.Unlock()
	}
	{
		result, err := s.client.GetEmailChanges(mailAccountId, lastState, true, 0, 0, ctx)
		require.NoError(err)
		require.Equal(session.State, result.GetSessionState())
		require.NotEqual(lastState, result.GetState())
		require.NotEqual(lastState, result.Payload.NewState)
		require.Equal(result.GetState(), result.Payload.NewState)
		require.Len(result.Payload.Created, 1)
		require.Empty(result.Payload.Destroyed)
		require.Empty(result.Payload.Updated)
		lastState = result.GetState()

		emailIds = append(emailIds, structs.Map(result.Payload.Created, func(e Email) string { return e.Id })...)
	}

	{
		_, n, err := s.fillEmailsWithImap(inboxFolder, 0, true, user)
		require.NoError(err)
		require.Equal(0, n)
	}

	require.Eventually(func() bool {
		return l.calls.Load() == uint32(3)
	}, 3*time.Second, 200*time.Millisecond, "WS push listener was not called after third email state change")
	{
		l.m.Lock()
		require.Len(l.emailStates, 3)
		require.Len(l.mailboxStates, 3)
		require.Len(l.threadStates, 3)
		assert.NotEqual(t, l.emailStates[1], l.emailStates[2])
		assert.NotEqual(t, l.mailboxStates[1], l.mailboxStates[2])
		assert.NotEqual(t, l.threadStates[1], l.threadStates[2])
		l.m.Unlock()
	}
	{
		result, err := s.client.GetEmailChanges(mailAccountId, lastState, true, 0, 0, ctx)
		require.NoError(err)
		require.Equal(session.State, result.GetSessionState())
		require.NotEqual(lastState, result.GetState())
		require.NotEqual(lastState, result.Payload.NewState)
		require.Equal(result.GetState(), result.Payload.NewState)
		require.Empty(result.Payload.Created)
		require.Len(result.Payload.Destroyed, 2)
		{
			a := make([]string, len(emailIds))
			copy(a, emailIds)
			slices.Sort(emailIds)
			b := make([]string, len(result.Payload.Destroyed))
			copy(b, result.Payload.Destroyed)
			slices.Sort(b)
			require.EqualValues(a, b)
		}
		require.Empty(result.Payload.Updated)
		lastState = result.GetState()
	}

	err = wsc.DisableNotifications()
	require.NoError(err)
}
