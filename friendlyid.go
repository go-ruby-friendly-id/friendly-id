// Copyright (c) 2026, the go-ruby-friendly-id/friendly-id authors
//
// SPDX-License-Identifier: BSD-3-Clause

// Package friendlyid is a pure-Go (CGO=0), MRI-faithful reimplementation of the
// Ruby friendly_id gem's slug engine: slug normalization (the ActiveSupport
// parameterize semantics), candidate generation with collision suffixing, an
// optional slug history, and the friendly finder that resolves a slug — or an
// old historical slug — back to a record id, falling back to a numeric id.
//
// Everything host-specific (the base attribute to slugify, uniqueness lookups,
// and history persistence) is an injectable seam, so the engine runs with no
// database and a go-embedded-ruby (rbgo) binding can wire the seams to
// ActiveRecord.
package friendlyid

import (
	"fmt"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// Config controls slug generation. The zero Config is usable: Separator defaults
// to "-" and SequenceSeparator to "-" when empty (see normalized()).
type Config struct {
	// Separator replaces runs of non-alphanumeric characters in a slug.
	Separator string
	// SequenceSeparator joins a base slug and a numeric sequence when resolving a
	// collision (e.g. "post" -> "post-2"). Defaults to Separator when empty.
	SequenceSeparator string
	// Reserved is a case-insensitive blocklist of words that may never be used as
	// a slug (friendly_id's reserved_words), e.g. "new", "edit".
	Reserved []string
	// MaxLength truncates a normalized slug to at most this many bytes when > 0.
	MaxLength int
}

func (c Config) separator() string {
	if c.Separator == "" {
		return "-"
	}
	return c.Separator
}

func (c Config) sequenceSeparator() string {
	if c.SequenceSeparator != "" {
		return c.SequenceSeparator
	}
	return c.separator()
}

// Parameterize turns an arbitrary string into a slug using the ActiveSupport
// parameterize semantics with sep as the separator: transliterate accented Latin
// characters to ASCII, lower-case, replace every run of characters outside
// [a-z0-9] with a single separator, and trim leading/trailing separators.
func Parameterize(s, sep string) string {
	if sep == "" {
		sep = "-"
	}
	ascii := transliterate(s)
	var b strings.Builder
	b.Grow(len(ascii))
	prevSep := false
	for _, r := range ascii {
		switch {
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r + ('a' - 'A'))
			prevSep = false
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			b.WriteRune(r)
			prevSep = false
		default:
			if !prevSep {
				b.WriteString(sep)
				prevSep = true
			}
		}
	}
	return strings.Trim(b.String(), sep)
}

// transliterate maps accented Latin characters to their ASCII base by NFD
// decomposition (dropping the combining marks) and folds a few common ligatures
// and symbols the way I18n's default transliteration table does.
func transliterate(s string) string {
	// A handful of characters do not decompose under NFD but have a conventional
	// ASCII transliteration (matching ActiveSupport's default rules).
	replacer := strings.NewReplacer(
		"ß", "ss", "æ", "ae", "Æ", "AE", "œ", "oe", "Œ", "OE",
		"ø", "o", "Ø", "O", "đ", "d", "Đ", "D", "ł", "l", "Ł", "L",
		"þ", "th", "Þ", "TH", "ð", "d", "Ð", "D",
	)
	s = replacer.Replace(s)
	decomposed := norm.NFD.String(s)
	var b strings.Builder
	b.Grow(len(decomposed))
	for _, r := range decomposed {
		if unicode.Is(unicode.Mn, r) { // drop combining marks
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// Normalize produces the canonical friendly id for base: parameterized, reserved
// words rejected (returning ErrReserved), truncated to MaxLength, and rejected
// when empty (ErrBlank). It is the value friendly_id stores as the slug.
func (c Config) Normalize(base string) (string, error) {
	slug := Parameterize(base, c.separator())
	if c.MaxLength > 0 && len(slug) > c.MaxLength {
		slug = strings.Trim(slug[:c.MaxLength], c.separator())
	}
	if slug == "" {
		return "", ErrBlank
	}
	if c.isReserved(slug) {
		return "", fmt.Errorf("%w: %q", ErrReserved, slug)
	}
	return slug, nil
}

func (c Config) isReserved(slug string) bool {
	for _, w := range c.Reserved {
		if strings.EqualFold(w, slug) {
			return true
		}
	}
	return false
}

// Candidates returns the ordered slug candidates friendly_id tries for base plus
// any extra discriminators: the base slug first, then the base joined with each
// discriminator by the sequence separator. Empty/reserved candidates are skipped.
// The first returned candidate is always the normalized base (when valid).
func (c Config) Candidates(base string, extra ...string) []string {
	out := make([]string, 0, len(extra)+1)
	seen := map[string]bool{}
	add := func(raw string) {
		slug := Parameterize(raw, c.separator())
		if c.MaxLength > 0 && len(slug) > c.MaxLength {
			slug = strings.Trim(slug[:c.MaxLength], c.separator())
		}
		if slug == "" || c.isReserved(slug) || seen[slug] {
			return
		}
		seen[slug] = true
		out = append(out, slug)
	}
	add(base)
	for _, e := range extra {
		add(base + " " + e)
	}
	return out
}

// Resolve returns the first candidate for which exists reports false — the slug
// that can be assigned without colliding. When every candidate collides it falls
// back to sequential suffixing of the base ("base-2", "base-3", …) until a free
// slug is found. base must normalize to a non-empty, non-reserved slug.
func (c Config) Resolve(base string, exists func(slug string) bool, extra ...string) (string, error) {
	if exists == nil {
		return "", ErrNilExists
	}
	cands := c.Candidates(base, extra...)
	if len(cands) == 0 {
		// No candidate means base normalizes to a blank or reserved slug, so
		// Normalize necessarily reports why (its error is always non-nil here).
		_, err := c.Normalize(base)
		return "", err
	}
	for _, cand := range cands {
		if !exists(cand) {
			return cand, nil
		}
	}
	root := cands[0]
	for n := 2; ; n++ {
		cand := root + c.sequenceSeparator() + fmt.Sprintf("%d", n)
		if !exists(cand) {
			return cand, nil
		}
	}
}
