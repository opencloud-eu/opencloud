// Copyright 2026 OpenCloud GmbH <mail@opencloud.eu>
// SPDX-License-Identifier: Apache-2.0

package revisions

type SimplePlan struct {
	retained    []Revision
	deleted     []Revision
	safeguarded []Revision
	reasons     map[string]string // by revision key
}

var _ Plan = &SimplePlan{}

func NewSimplePlan() Plan {
	return &SimplePlan{
		retained:    []Revision{},
		deleted:     []Revision{},
		safeguarded: []Revision{},
		reasons:     map[string]string{},
	}
}

func (p *SimplePlan) Delete(reason string, revs ...Revision) {
	p.deleted = append(p.deleted, revs...)
	p.setReason(reason, revs)
}

func (p *SimplePlan) Retain(reason string, revs ...Revision) {
	p.retained = append(p.retained, revs...)
	p.setReason(reason, revs)
}

func (p *SimplePlan) Safeguard(reason string, revs ...Revision) {
	p.safeguarded = append(p.safeguarded, revs...)
	p.setReason(reason, revs)
}

func (p *SimplePlan) setReason(reason string, revs []Revision) {
	for _, rev := range revs {
		p.reasons[rev.Key()] = reason
	}
}

func (p *SimplePlan) Deleted() []Revision {
	return p.deleted
}

func (p *SimplePlan) Retained() []Revision {
	return p.retained
}

func (p *SimplePlan) Safeguarded() []Revision {
	return p.safeguarded
}

func (p *SimplePlan) Reason(key string) string {
	return p.reasons[key]
}
