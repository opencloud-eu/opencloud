package jmap

import (
	"context"
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
		if st, ok := changed[EmailType]; ok {
			l.emailStates = append(l.emailStates, st)
		}
		if st, ok := changed[ThreadType]; ok {
			l.threadStates = append(l.threadStates, st)
		}
		if st, ok := changed[MailboxType]; ok {
			l.mailboxStates = append(l.mailboxStates, st)
		}
		l.m.Unlock()

		unsupportedKeys := structs.Filter(structs.Keys(changed), func(o ObjectType) bool { return o != EmailType && o != ThreadType && o != MailboxType })
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

	ctx := context.Background()

	s, err := newStalwartTest(t)
	require.NoError(err)
	defer s.Close()

	user := pickUser()
	session := s.Session(user.name)

	mailAccountId := session.PrimaryAccounts.Mail
	inboxFolder := ""
	{
		_, inboxFolder = s.findInbox(t, mailAccountId, session)
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
		changes, sessionState, state, _, err := s.client.GetEmailChanges(mailAccountId, session, s.ctx, s.logger, "", State(""), true, 0, 0)
		require.NoError(err)
		require.Equal(session.State, sessionState)
		require.NotEmpty(state)
		//fmt.Printf("\x1b[45;1;4mChanges [%s]:\x1b[0m\n", state)
		//for _, c := range changes.Created { fmt.Printf("%s %s\n", c.Id, c.Subject) }
		initialState = state
		require.Empty(changes.Created)
		require.Empty(changes.Destroyed)
		require.Empty(changes.Updated)
	}
	require.NotEmpty(initialState)

	{
		changes, sessionState, state, _, err := s.client.GetEmailChanges(mailAccountId, session, s.ctx, s.logger, "", initialState, true, 0, 0)
		require.NoError(err)
		require.Equal(session.State, sessionState)
		require.Equal(initialState, state)
		require.Equal(initialState, changes.NewState)
		require.Empty(changes.Created)
		require.Empty(changes.Destroyed)
		require.Empty(changes.Updated)
	}

	wsc, err := s.client.EnablePushNotifications(ctx, initialState, func() (*Session, error) { return session, nil })
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
		changes, sessionState, state, _, err := s.client.GetEmailChanges(mailAccountId, session, s.ctx, s.logger, "", initialState, true, 0, 0)
		require.NoError(err)
		require.Equal(session.State, sessionState)
		require.NotEqual(initialState, state)
		require.NotEqual(initialState, changes.NewState)
		require.Equal(state, changes.NewState)
		require.Len(changes.Created, 1)
		require.Empty(changes.Destroyed)
		require.Empty(changes.Updated)
		lastState = state

		emailIds = append(emailIds, structs.Map(changes.Created, func(e Email) string { return e.Id })...)
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
		changes, sessionState, state, _, err := s.client.GetEmailChanges(mailAccountId, session, s.ctx, s.logger, "", lastState, true, 0, 0)
		require.NoError(err)
		require.Equal(session.State, sessionState)
		require.NotEqual(lastState, state)
		require.NotEqual(lastState, changes.NewState)
		require.Equal(state, changes.NewState)
		require.Len(changes.Created, 1)
		require.Empty(changes.Destroyed)
		require.Empty(changes.Updated)
		lastState = state

		emailIds = append(emailIds, structs.Map(changes.Created, func(e Email) string { return e.Id })...)
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
		changes, sessionState, state, _, err := s.client.GetEmailChanges(mailAccountId, session, s.ctx, s.logger, "", lastState, true, 0, 0)
		require.NoError(err)
		require.Equal(session.State, sessionState)
		require.NotEqual(lastState, state)
		require.NotEqual(lastState, changes.NewState)
		require.Equal(state, changes.NewState)
		require.Empty(changes.Created)
		require.Len(changes.Destroyed, 2)
		require.EqualValues(emailIds, changes.Destroyed)
		require.Empty(changes.Updated)
		lastState = state
	}

	err = wsc.DisableNotifications()
	require.NoError(err)
}
