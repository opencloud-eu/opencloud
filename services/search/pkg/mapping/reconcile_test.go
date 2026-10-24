package mapping

import (
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/opencloud-eu/opencloud/pkg/log"
)

type fakeReconciler struct {
	classification Classification
	classifyErr    error
	persisted      bool
	applyErr       error
	applyCalls     int
}

func (f *fakeReconciler) Classify() (Classification, error) {
	return f.classification, f.classifyErr
}

func (f *fakeReconciler) ApplyAdditive() (bool, error) {
	f.applyCalls++
	return f.persisted, f.applyErr
}

var _ = Describe("Reconcile", func() {
	logger := log.NopLogger()

	It("does nothing on an equal schema", func() {
		r := &fakeReconciler{classification: Classification{Verdict: VerdictEqual}}

		c, err := Reconcile("idx", r, logger)
		Expect(err).ToNot(HaveOccurred())
		Expect(c.Verdict).To(Equal(VerdictEqual))
		Expect(r.applyCalls).To(BeZero())
	})

	It("refuses a breaking schema without applying it", func() {
		r := &fakeReconciler{classification: Classification{Verdict: VerdictBreaking, Reasons: []string{"Name changed"}}}

		_, err := Reconcile("idx", r, logger)
		Expect(err).To(MatchError(ErrManualActionRequired))
		Expect(err.Error()).To(ContainSubstring("Name changed"))
		Expect(r.applyCalls).To(BeZero())
	})

	It("applies an additive schema", func() {
		r := &fakeReconciler{classification: Classification{Verdict: VerdictAdditive, NewFields: []string{"Size"}}, persisted: true}

		c, err := Reconcile("idx", r, logger)
		Expect(err).ToNot(HaveOccurred())
		Expect(c.Verdict).To(Equal(VerdictAdditive))
		Expect(r.applyCalls).To(Equal(1))
	})

	It("surfaces an apply error even when the schema was persisted", func() {
		r := &fakeReconciler{
			classification: Classification{Verdict: VerdictAdditive, NewFields: []string{"Size"}},
			persisted:      true,
			applyErr:       errors.New("reopen failed"),
		}

		_, err := Reconcile("idx", r, logger)
		Expect(err).To(MatchError(ContainSubstring("reopen failed")))
		Expect(r.applyCalls).To(Equal(1))
	})

	It("propagates a classify error", func() {
		r := &fakeReconciler{classifyErr: errors.New("read schema failed")}

		_, err := Reconcile("idx", r, logger)
		Expect(err).To(MatchError(ContainSubstring("read schema failed")))
		Expect(r.applyCalls).To(BeZero())
	})
})
