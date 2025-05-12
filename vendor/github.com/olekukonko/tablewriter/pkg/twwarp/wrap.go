// Copyright 2014 Oleku Konko All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

// This module is a Table Writer  API for the Go Programming Language.
// The protocols were written in pure Go and works on windows and unix systems

package twwarp

import (
	"math"
	"strings"
	"unicode"

	"github.com/mattn/go-runewidth"
)

const (
	nl = "\n"
	sp = " "
)

const defaultPenalty = 1e5

// WrapString wraps s into a paragraph of lines of length lim, with minimal
// raggedness.
func WrapString(s string, lim int) ([]string, int) {
	if s == sp {
		return []string{sp}, lim
	}
	words := SplitWords(s)
	if len(words) == 0 {
		return []string{""}, lim
	}
	var lines []string
	max := 0
	for _, v := range words {
		max = runewidth.StringWidth(v)
		if max > lim {
			lim = max
		}
	}
	for _, line := range WrapWords(words, 1, lim, defaultPenalty) {
		lines = append(lines, strings.Join(line, sp))
	}
	return lines, lim
}

func SplitWords(s string) []string {
	words := make([]string, 0, len(s)/5)
	var wordBegin int
	wordPending := false
	for i, c := range s {
		if unicode.IsSpace(c) {
			if wordPending {
				words = append(words, s[wordBegin:i])
				wordPending = false
			}
			continue
		}
		if !wordPending {
			wordBegin = i
			wordPending = true
		}
	}
	if wordPending {
		words = append(words, s[wordBegin:])
	}
	return words
}

// WrapWords is the low-level line-breaking algorithm, useful if you need more
// control over the details of the text wrapping process. For most uses,
// WrapString will be sufficient and more convenient.
//
// WrapWords splits a list of words into lines with minimal "raggedness",
// treating each rune as one unit, accounting for spc units between adjacent
// words on each line, and attempting to limit lines to lim units. Raggedness
// is the total error over all lines, where error is the square of the
// difference of the length of the line and lim. Too-long lines (which only
// happen when a single word is longer than lim units) have pen penalty units
// added to the error.
func WrapWords(words []string, spc, lim, pen int) [][]string {
	n := len(words)
	if n == 0 {
		return nil
	}
	lengths := make([]int, n)
	for i := 0; i < n; i++ {
		lengths[i] = runewidth.StringWidth(words[i])
	}
	nbrk := make([]int, n)
	cost := make([]int, n)
	for i := range cost {
		cost[i] = math.MaxInt32
	}
	remainderLen := lengths[n-1]
	for i := n - 1; i >= 0; i-- {
		if i < n-1 {
			remainderLen += spc + lengths[i]
		}
		if remainderLen <= lim {
			cost[i] = 0
			nbrk[i] = n
			continue
		}
		phraseLen := lengths[i]
		for j := i + 1; j < n; j++ {
			if j > i+1 {
				phraseLen += spc + lengths[j-1]
			}
			d := lim - phraseLen
			c := d*d + cost[j]
			if phraseLen > lim {
				c += pen // too-long lines get a worse penalty
			}
			if c < cost[i] {
				cost[i] = c
				nbrk[i] = j
			}
		}
	}
	var lines [][]string
	i := 0
	for i < n {
		lines = append(lines, words[i:nbrk[i]])
		i = nbrk[i]
	}
	return lines
}

// getLines decomposes a multiline string into a slice of strings.
func getLines(s string) []string {
	return strings.Split(s, nl)
}
