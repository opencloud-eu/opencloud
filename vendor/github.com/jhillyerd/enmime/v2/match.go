package enmime

import (
	"container/list"
)

// PartMatcher is a function type that you must implement to search for Parts using the
// BreadthMatch* functions.  Implementators should inspect the provided Part and return true if it
// matches your criteria.
type PartMatcher func(part *Part) bool

// BreadthMatchFirst performs a breadth first search of the Part tree and returns the first part
// that causes the given matcher to return true
func (p *Part) BreadthMatchFirst(matcher PartMatcher) *Part {
	q := list.New()
	q.PushBack(p)

	// Push children onto queue and attempt to match in that order
	for q.Len() > 0 {
		e := q.Front()
		p := e.Value.(*Part)
		if matcher(p) {
			return p
		}
		q.Remove(e)
		c := p.FirstChild
		for c != nil {
			q.PushBack(c)
			c = c.NextSibling
		}
	}

	return nil
}

// BreadthMatchAll performs a breadth first search of the Part tree and returns all parts that cause
// the given matcher to return true
func (p *Part) BreadthMatchAll(matcher PartMatcher) []*Part {
	q := list.New()
	q.PushBack(p)

	matches := make([]*Part, 0, 10)

	// Push children onto queue and attempt to match in that order
	for q.Len() > 0 {
		e := q.Front()
		p := e.Value.(*Part)
		if matcher(p) {
			matches = append(matches, p)
		}
		q.Remove(e)
		c := p.FirstChild
		for c != nil {
			q.PushBack(c)
			c = c.NextSibling
		}
	}

	return matches
}

// DepthMatchFirst performs a depth first search of the Part tree and returns the first part that
// causes the given matcher to return true
func (p *Part) DepthMatchFirst(matcher PartMatcher) *Part {
	root := p
	for {
		if matcher(p) {
			return p
		}
		c := p.FirstChild
		if c != nil {
			p = c
		} else {
			for p.NextSibling == nil {
				if p == root {
					return nil
				}
				p = p.Parent
			}
			p = p.NextSibling
		}
	}
}

// DepthMatchAll performs a depth first search of the Part tree and returns all parts that causes
// the given matcher to return true
func (p *Part) DepthMatchAll(matcher PartMatcher) []*Part {
	root := p
	matches := make([]*Part, 0, 10)
	for {
		if matcher(p) {
			matches = append(matches, p)
		}
		c := p.FirstChild
		if c != nil {
			p = c
		} else {
			for p.NextSibling == nil {
				if p == root {
					return matches
				}
				p = p.Parent
			}
			p = p.NextSibling
		}
	}
}
