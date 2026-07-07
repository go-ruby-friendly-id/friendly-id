// Copyright (c) 2026, the go-ruby-friendly-id/friendly-id authors
//
// SPDX-License-Identifier: BSD-3-Clause

package friendlyid

// Finder implements friendly_id's Model.friendly.find(input): resolve a current
// slug to a record id, then (when history is enabled) an old historical slug,
// and finally fall back to treating the input as a raw primary key. Each lookup
// is an injectable seam so the finder runs with no database; an rbgo binding
// wires them to ActiveRecord.
type Finder struct {
	// BySlug resolves the model's current slug column. Required.
	BySlug func(slug string) (id string, found bool)
	// ByHistory resolves a slug recorded in the slug history. Nil disables history
	// (friendly_id without the :history module).
	ByHistory func(slug string) (id string, found bool)
	// ByID reports whether a record exists for a raw primary key. Nil disables the
	// numeric-id fallback.
	ByID func(id string) (found bool)
}

// Find resolves input to a record id, trying the current slug, then history, then
// the raw id, and returns ErrNotFound when none match. The BySlug seam is
// required; a nil Finder.BySlug makes Find always fall through to history/id.
func (f Finder) Find(input string) (id string, err error) {
	if f.BySlug != nil {
		if got, ok := f.BySlug(input); ok {
			return got, nil
		}
	}
	if f.ByHistory != nil {
		if got, ok := f.ByHistory(input); ok {
			return got, nil
		}
	}
	if f.ByID != nil && f.ByID(input) {
		return input, nil
	}
	return "", ErrNotFound
}
