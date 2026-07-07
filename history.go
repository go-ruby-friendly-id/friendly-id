// Copyright (c) 2026, the go-ruby-friendly-id/friendly-id authors
//
// SPDX-License-Identifier: BSD-3-Clause

package friendlyid

// Slug is one recorded slug row — the friendly_id_slugs table analogue used by
// the :history module.
type Slug struct {
	Slug          string
	SluggableType string
	SluggableID   string
}

// MemStore is an in-memory reference slug store for a single sluggable type. It
// tracks each record's current slug plus the full history of slugs it ever held,
// and exposes the seams a Config/Finder needs: Exists (uniqueness among current
// slugs), CurrentID (the model's slug column), and HistoryID (an old slug). An
// rbgo binding replaces it with an ActiveRecord-backed store; tests use it
// directly.
type MemStore struct {
	sluggableType string
	current       map[string]string // slug -> id (currently assigned)
	idToSlug      map[string]string // id   -> current slug
	history       []Slug            // every slug ever assigned, in order
}

// NewMemStore returns an empty store for records of sluggableType.
func NewMemStore(sluggableType string) *MemStore {
	return &MemStore{
		sluggableType: sluggableType,
		current:       map[string]string{},
		idToSlug:      map[string]string{},
	}
}

// Assign records slug as id's current slug. The record's previous current slug
// (if any and different) is retained only in history, so an old slug keeps
// resolving to the record. A slug already current for a different record is
// rejected with ErrTaken.
func (m *MemStore) Assign(id, slug string) error {
	if owner, ok := m.current[slug]; ok && owner != id {
		return ErrTaken
	}
	if prev, ok := m.idToSlug[id]; ok && prev != slug {
		delete(m.current, prev) // prev remains in history
	}
	m.current[slug] = id
	m.idToSlug[id] = slug
	m.history = append(m.history, Slug{Slug: slug, SluggableType: m.sluggableType, SluggableID: id})
	return nil
}

// Exists reports whether slug is currently assigned to some record — the
// uniqueness check Config.Resolve needs.
func (m *MemStore) Exists(slug string) bool {
	_, ok := m.current[slug]
	return ok
}

// CurrentID returns the record id whose current slug is slug (the BySlug seam).
func (m *MemStore) CurrentID(slug string) (string, bool) {
	id, ok := m.current[slug]
	return id, ok
}

// HistoryID returns the id of the most recent record to have held slug at any
// time (the ByHistory seam), even if the slug is no longer current.
func (m *MemStore) HistoryID(slug string) (string, bool) {
	for i := len(m.history) - 1; i >= 0; i-- {
		if m.history[i].Slug == slug {
			return m.history[i].SluggableID, true
		}
	}
	return "", false
}

// Slugs returns a copy of the full recorded history in assignment order.
func (m *MemStore) Slugs() []Slug {
	out := make([]Slug, len(m.history))
	copy(out, m.history)
	return out
}

// Finder builds a Finder wired to this store, with the numeric-id fallback
// checking byID.
func (m *MemStore) Finder(byID func(id string) bool) Finder {
	return Finder{BySlug: m.CurrentID, ByHistory: m.HistoryID, ByID: byID}
}
