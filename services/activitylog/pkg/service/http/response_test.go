package http_test

import (
	"context"
	"time"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	user "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/opencloud-eu/opencloud/pkg/l10n"
	httpsvc "github.com/opencloud-eu/opencloud/services/activitylog/pkg/service/http"
)

var _ = Describe("Response", func() {
	Describe("NewActivity", func() {
		It("creates an activity with the given parameters", func() {
			ts := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
			vars := map[string]any{
				"user":     "testuser",
				"resource": "testfile.txt",
			}

			act := httpsvc.NewActivity("Test message", ts, "event-123", vars)

			Expect(act.Id).To(Equal("event-123"))
			Expect(act.Times.RecordedTime).To(Equal(ts))
			Expect(act.Template.Message).To(Equal("Test message"))
			Expect(act.Template.Variables).To(HaveKeyWithValue("user", "testuser"))
			Expect(act.Template.Variables).To(HaveKeyWithValue("resource", "testfile.txt"))
		})

		It("handles empty variables map", func() {
			act := httpsvc.NewActivity("", time.Time{}, "", map[string]any{})

			Expect(act.Id).To(BeEmpty())
			Expect(act.Times.RecordedTime).To(Equal(time.Time{}))
			Expect(act.Template.Message).To(BeEmpty())
			Expect(act.Template.Variables).To(BeEmpty())
		})
	})

	Describe("WithOldResource", func() {
		It("sets the oldResource variable from reference path", func() {
			ref := &provider.Reference{
				Path: "/old/path/oldname.txt",
			}
			vars := make(map[string]any)

			opt := httpsvc.WithOldResource(ref)
			err := opt(context.Background(), nil, vars)
			Expect(err).ToNot(HaveOccurred())
			Expect(vars).To(HaveKey("oldResource"))
		})
	})

	Describe("WithUser", func() {
		It("returns error when no user is provided", func() {
			opt := httpsvc.WithUser(nil, nil, nil)
			err := opt(context.Background(), nil, make(map[string]any))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("no user provided"))
		})

		It("uses impersonator when provided", func() {
			impersonator := &user.User{
				Id: &user.UserId{
					OpaqueId: "imp-user-id",
				},
				DisplayName: "Impersonated User",
			}
			vars := make(map[string]any)

			opt := httpsvc.WithUser(nil, nil, impersonator)
			err := opt(context.Background(), nil, vars)
			Expect(err).ToNot(HaveOccurred())
			Expect(vars).To(HaveKey("user"))
		})

		It("uses executing user when no impersonator", func() {
			execUser := &user.User{
				Id: &user.UserId{
					OpaqueId: "exec-user-id",
				},
				DisplayName: "Executing User",
			}
			vars := make(map[string]any)

			opt := httpsvc.WithUser(nil, execUser, nil)
			err := opt(context.Background(), nil, vars)
			Expect(err).ToNot(HaveOccurred())
			Expect(vars).To(HaveKey("user"))
		})
	})

	Describe("WithVar", func() {
		It("sets a simple key-value variable", func() {
			vars := make(map[string]any)

			opt := httpsvc.WithVar("token", "id123", "My Token")
			err := opt(context.Background(), nil, vars)
			Expect(err).ToNot(HaveOccurred())
			Expect(vars).To(HaveKey("token"))
		})
	})

	Describe("WithTranslation", func() {
		It("sets translated field variable", func() {
			var t l10n.Translator
			vars := make(map[string]any)

			opt := httpsvc.WithTranslation(&t, "en", "field", []string{"permission"})
			err := opt(context.Background(), nil, vars)
			Expect(err).ToNot(HaveOccurred())
			Expect(vars).To(HaveKey("field"))
		})

		It("handles empty values slice", func() {
			var t l10n.Translator
			vars := make(map[string]any)

			opt := httpsvc.WithTranslation(&t, "en", "field", []string{})
			err := opt(context.Background(), nil, vars)
			Expect(err).ToNot(HaveOccurred())
			Expect(vars).To(HaveKey("field"))
		})
	})

	Describe("ActivityOption type", func() {
		It("allows composing multiple options", func() {
			vars := make(map[string]any)
			ctx := context.Background()
			var gwc gateway.GatewayAPIClient

			options := []httpsvc.ActivityOption{
				httpsvc.WithVar("key1", "id1", "name1"),
				httpsvc.WithVar("key2", "id2", "name2"),
			}

			for _, opt := range options {
				err := opt(ctx, gwc, vars)
				Expect(err).ToNot(HaveOccurred())
			}

			Expect(vars).To(HaveKey("key1"))
			Expect(vars).To(HaveKey("key2"))
		})
	})
})
