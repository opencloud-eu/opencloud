package groupware

import (
	"time"

	g "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/opencloud-eu/opencloud/pkg/jmap"
)

var _ = g.Describe("Groupware", func() {

	g.Describe("message", func() {
		g.It("copies a JMAP Email to a Graph Message correctly", func() {
			now := time.Now()
			email := jmap.Email{
				Id:        "id",
				MessageId: []string{"123.456@example.com"},
				BlobId:    "918e929e-a296-4078-915a-2c2abc580a8d",
				ThreadId:  "t",
				Size:      12345,
				From: []jmap.EmailAddress{
					{Name: "Bobbie Draper", Email: "bobbie@mcrn.mars"},
				},
				To: []jmap.EmailAddress{
					{Name: "Camina Drummer", Email: "camina@opa.org"},
				},
				Subject:        "test subject",
				HasAttachments: true,
				ReceivedAt:     now,
				Preview:        "the preview",
				TextBody: []jmap.EmailBodyRef{
					{PartId: "0", Type: "text/plain"},
				},
				BodyValues: map[string]jmap.EmailBody{
					"0": {
						IsEncodingProblem: false,
						IsTruncated:       false,
						Value:             "the body",
					},
				},
			}

			msg := message(email, "aaa")
			Expect(msg.Body.ContentType).To(Equal("text/plain"))
			Expect(msg.Body.Content).To(Equal("the body"))
			Expect(msg.Subject).To(Equal("test subject"))
		})
	})
})
