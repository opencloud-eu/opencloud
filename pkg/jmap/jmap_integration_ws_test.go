package jmap

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testWsPushListener struct {
	logger        *log.Logger
	mailAccountId string
	calls         atomic.Uint32
	m             sync.Mutex
	emailStates   []string
	threadStates  []string
	mailboxStates []string
}

func (l *testWsPushListener) OnNotification(pushState StateChange) {
	l.calls.Add(1)
	// pushState is currently not supported by Stalwart, let's use the Email state instead
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
	}
}

var _ WsPushListener = &testWsPushListener{}

func TestWs(t *testing.T) {
	if skip(t) {
		return
	}

	require := require.New(t)

	s, err := newStalwartTest(t)
	require.NoError(err)
	defer s.Close()

	mailAccountId := s.session.PrimaryAccounts.Mail
	inboxFolder := ""
	{
		_, inboxFolder = s.findInbox(t, mailAccountId)
	}

	l := &testWsPushListener{logger: s.logger, mailAccountId: mailAccountId}
	s.client.AddWsPushListener(l)
	require.Equal(uint32(0), l.calls.Load())

	wsc, err := s.client.EnableNotifications("", func() (*Session, error) { return s.session, nil })
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

	{
		_, n, err := s.fillEmailsWithImap(inboxFolder, 1)
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

	{
		_, n, err := s.fillEmailsWithImap(inboxFolder, 1)
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

	err = wsc.DisableNotifications()
	require.NoError(err)
}
