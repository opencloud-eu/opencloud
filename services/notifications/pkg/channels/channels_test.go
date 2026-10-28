package channels_test

import (
	"context"
	"errors"
	"net"
	"net/mail"
	"net/textproto"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/notifications/pkg/channels"
	"github.com/opencloud-eu/opencloud/services/notifications/pkg/config"
)

// startFakeSMTPServer offers STARTTLS but never performs it. It sends reply to
// AUTH and MAIL FROM, and only advertises AUTH when advertiseAuth is set.
func startFakeSMTPServer(advertiseAuth bool, reply string) int {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	Expect(err).ToNot(HaveOccurred())
	DeferCleanup(l.Close)

	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				return
			}
			go func() {
				defer conn.Close()
				tc := textproto.NewConn(conn)
				tc.PrintfLine("220 fake ESMTP")
				for {
					line, err := tc.ReadLine()
					if err != nil {
						return
					}
					cmd, _, _ := strings.Cut(strings.ToUpper(line), " ")
					switch cmd {
					case "EHLO":
						tc.PrintfLine("250-fake")
						if advertiseAuth {
							tc.PrintfLine("250-AUTH PLAIN LOGIN")
						}
						tc.PrintfLine("250 STARTTLS")
					case "AUTH", "MAIL":
						tc.PrintfLine("%s", reply)
					case "QUIT":
						tc.PrintfLine("221 bye")
						return
					default:
						tc.PrintfLine("250 ok")
					}
				}
			}()
		}
	}()

	return l.Addr().(*net.TCPAddr).Port
}

var _ = Describe("Mail", func() {
	Describe("AppendSender", func() {
		var withName, withoutName mail.Address

		BeforeEach(func() {
			a, err := mail.ParseAddress("OpenCloud <noreply@example.com>")
			Expect(err).ToNot(HaveOccurred())
			withName = *a

			a, err = mail.ParseAddress("noreply@example.com")
			Expect(err).ToNot(HaveOccurred())
			withoutName = *a
		})

		DescribeTable("adds the sender to the display name",
			func(sender, wantWithName, wantWithoutName string) {
				Expect(channels.AppendSender(sender, withName)).To(Equal(wantWithName))
				Expect(channels.AppendSender(sender, withoutName)).To(Equal(wantWithoutName))
			},
			Entry("empty sender", "",
				`"OpenCloud" <noreply@example.com>`,
				`<noreply@example.com>`),
			Entry("not empty sender", `Joe Q. Public`,
				`"Joe Q. Public via OpenCloud" <noreply@example.com>`,
				`"Joe Q. Public via" <noreply@example.com>`),
			Entry("sender with comma and semicolon", `Joe, Q; Public:`,
				`"Joe, Q; Public: via OpenCloud" <noreply@example.com>`,
				`"Joe, Q; Public: via" <noreply@example.com>`),
			Entry("sender with quotes", `Joe Q. "Public"`,
				`"Joe Q. \"Public\" via OpenCloud" <noreply@example.com>`,
				`"Joe Q. \"Public\" via" <noreply@example.com>`),
		)
	})

	Describe("SendMessage", func() {
		DescribeTable("returns the server error on an unencrypted connection",
			func(advertiseAuth bool, reply string, wantCode int, wantHint bool) {
				cfg := config.Config{}
				cfg.Notifications.SMTP = config.SMTP{
					Host:           "127.0.0.1",
					Port:           startFakeSMTPServer(advertiseAuth, reply),
					Sender:         "noreply@example.com",
					Username:       "user",
					Password:       "secret",
					Authentication: "auto",
					Encryption:     "none",
				}
				ch, err := channels.NewMailChannel(cfg, log.NopLogger())
				Expect(err).ToNot(HaveOccurred())

				err = ch.SendMessage(context.Background(), &channels.Message{
					Recipient: []string{"user@example.com"},
					Subject:   "subject",
					TextBody:  "body",
				})

				var tpErr *textproto.Error
				Expect(errors.As(err, &tpErr)).To(BeTrue(), "got %v", err)
				Expect(tpErr.Code).To(Equal(wantCode))
				if wantHint {
					Expect(err).To(MatchError(ContainSubstring("NOTIFICATIONS_SMTP_ENCRYPTION")))
				} else {
					Expect(err).ToNot(MatchError(ContainSubstring("NOTIFICATIONS_SMTP_ENCRYPTION")))
				}
			},
			Entry("STARTTLS required, rejected at MAIL FROM", false,
				"530 5.7.0 Must issue a STARTTLS command first", 530, true),
			Entry("STARTTLS required, rejected at AUTH", true,
				"530 5.7.0 Must issue a STARTTLS command first", 530, true),
			Entry("wrong credentials", true,
				"535 5.7.8 Authentication credentials invalid", 535, false),
		)
	})
})
