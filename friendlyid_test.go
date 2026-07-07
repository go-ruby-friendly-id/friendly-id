// Copyright (c) 2026, the go-ruby-friendly-id/friendly-id authors
//
// SPDX-License-Identifier: BSD-3-Clause

package friendlyid

import (
	"errors"
	"testing"
)

func TestParameterize(t *testing.T) {
	cases := []struct{ in, sep, want string }{
		{"Hello World", "-", "hello-world"},
		{"  Leading  and Trailing  ", "-", "leading-and-trailing"},
		{"Åccénts & Ñoisé", "-", "accents-noise"},
		{"straße", "-", "strasse"},
		{"Œuvre æther", "-", "oeuvre-aether"},
		{"Øre Łódź", "-", "ore-lodz"},
		{"Þor ðag", "-", "thor-dag"},
		{"under_score.dots/slash", "-", "under-score-dots-slash"},
		{"Multiple   Spaces", "_", "multiple_spaces"},
		{"CamelCase123", "-", "camelcase123"},
		{"!!!", "-", ""},
		{"", "-", ""},
		{"already-slugged", "", "already-slugged"}, // empty sep defaults to "-"
	}
	for _, c := range cases {
		if got := Parameterize(c.in, c.sep); got != c.want {
			t.Errorf("Parameterize(%q,%q)=%q want %q", c.in, c.sep, got, c.want)
		}
	}
}

func TestCustomSeparator(t *testing.T) {
	c := Config{Separator: "_"}
	if got, err := c.Normalize("Hello World"); err != nil || got != "hello_world" {
		t.Fatalf("custom sep normalize = %q,%v", got, err)
	}
	// SequenceSeparator defaults to Separator when empty.
	if got, _ := c.Resolve("Hello World", func(s string) bool { return s == "hello_world" }); got != "hello_world_2" {
		t.Fatalf("custom sep resolve = %q", got)
	}
}

func TestNormalize(t *testing.T) {
	c := Config{Reserved: []string{"new", "edit"}, MaxLength: 10}
	if got, err := c.Normalize("Hello World"); err != nil || got != "hello-worl" {
		t.Fatalf("normalize truncate = %q,%v", got, err)
	}
	// Truncation that lands on a separator trims it.
	c2 := Config{MaxLength: 6}
	if got, _ := c2.Normalize("ab cdef ghi"); got != "ab-cde" {
		t.Fatalf("normalize sep-trim = %q", got)
	}
	if _, err := c.Normalize("!!!"); !errors.Is(err, ErrBlank) {
		t.Fatalf("blank err = %v", err)
	}
	if _, err := c.Normalize("New"); !errors.Is(err, ErrReserved) {
		t.Fatalf("reserved err = %v", err)
	}
	if got, err := c.Normalize("Fine"); err != nil || got != "fine" {
		t.Fatalf("ok = %q,%v", got, err)
	}
}

func TestCandidates(t *testing.T) {
	c := Config{Reserved: []string{"new"}}
	got := c.Candidates("Post", "2026", "2026") // duplicate discriminator dedup
	want := []string{"post", "post-2026"}
	if len(got) != len(want) {
		t.Fatalf("candidates = %v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("candidates = %v want %v", got, want)
		}
	}
	// Base reserved/blank is skipped; only the valid discriminated candidate remains.
	if g := c.Candidates("new", "admin"); len(g) != 1 || g[0] != "new-admin" {
		t.Fatalf("reserved base candidates = %v", g)
	}
	if g := c.Candidates("!!!"); len(g) != 0 {
		t.Fatalf("blank base candidates = %v", g)
	}
	// MaxLength truncation applies to candidates too.
	cl := Config{MaxLength: 4}
	if g := cl.Candidates("abcdef"); g[0] != "abcd" {
		t.Fatalf("maxlen candidate = %v", g)
	}
}

func TestResolve(t *testing.T) {
	c := Config{}
	if _, err := c.Resolve("Post", nil); !errors.Is(err, ErrNilExists) {
		t.Fatalf("nil exists = %v", err)
	}
	// No candidates (blank base) -> ErrBlank.
	if _, err := c.Resolve("!!!", func(string) bool { return false }); !errors.Is(err, ErrBlank) {
		t.Fatalf("blank resolve = %v", err)
	}
	// No candidates because reserved -> the Normalize error surfaces (ErrReserved).
	cr := Config{Reserved: []string{"post"}}
	if _, err := cr.Resolve("Post", func(string) bool { return false }); !errors.Is(err, ErrReserved) {
		t.Fatalf("reserved resolve = %v", err)
	}
	// First candidate free.
	if got, _ := c.Resolve("Post", func(string) bool { return false }); got != "post" {
		t.Fatalf("free = %q", got)
	}
	// Base taken, discriminator free.
	taken := map[string]bool{"post": true}
	if got, _ := c.Resolve("Post", func(s string) bool { return taken[s] }, "b"); got != "post-b" {
		t.Fatalf("discriminator = %q", got)
	}
	// All candidates taken -> sequential suffix, skipping taken sequence numbers.
	taken2 := map[string]bool{"post": true, "post-2": true}
	if got, _ := c.Resolve("Post", func(s string) bool { return taken2[s] }); got != "post-3" {
		t.Fatalf("sequence = %q", got)
	}
	// Custom sequence separator.
	cs := Config{SequenceSeparator: "_"}
	if got, _ := cs.Resolve("Post", func(s string) bool { return s == "post" }); got != "post_2" {
		t.Fatalf("seq sep = %q", got)
	}
}

func TestFinder(t *testing.T) {
	f := Finder{
		BySlug:    func(s string) (string, bool) { return "id-slug", s == "current" },
		ByHistory: func(s string) (string, bool) { return "id-hist", s == "old" },
		ByID:      func(id string) bool { return id == "42" },
	}
	if got, _ := f.Find("current"); got != "id-slug" {
		t.Fatalf("by slug = %q", got)
	}
	if got, _ := f.Find("old"); got != "id-hist" {
		t.Fatalf("by history = %q", got)
	}
	if got, _ := f.Find("42"); got != "42" {
		t.Fatalf("by id = %q", got)
	}
	if _, err := f.Find("nope"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("not found = %v", err)
	}
	// Nil seams fall through cleanly.
	empty := Finder{}
	if _, err := empty.Find("x"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("empty finder = %v", err)
	}
}

func TestMemStore(t *testing.T) {
	m := NewMemStore("Post")
	if err := m.Assign("1", "hello"); err != nil {
		t.Fatal(err)
	}
	if !m.Exists("hello") {
		t.Fatal("exists")
	}
	if id, ok := m.CurrentID("hello"); !ok || id != "1" {
		t.Fatalf("current = %q %v", id, ok)
	}
	// Re-slug record 1: old slug leaves current but stays resolvable via history.
	if err := m.Assign("1", "world"); err != nil {
		t.Fatal(err)
	}
	if m.Exists("hello") {
		t.Fatal("old slug should no longer be current")
	}
	if id, ok := m.HistoryID("hello"); !ok || id != "1" {
		t.Fatalf("history = %q %v", id, ok)
	}
	if _, ok := m.HistoryID("never"); ok {
		t.Fatal("unknown history")
	}
	if id, ok := m.CurrentID("world"); !ok || id != "1" {
		t.Fatalf("new current = %q %v", id, ok)
	}
	// Re-assigning the same slug to the same record is idempotent (no delete path).
	if err := m.Assign("1", "world"); err != nil {
		t.Fatal(err)
	}
	// A slug current for another record is rejected.
	if err := m.Assign("2", "world"); !errors.Is(err, ErrTaken) {
		t.Fatalf("taken = %v", err)
	}
	// History accumulates in order.
	if s := m.Slugs(); len(s) != 3 || s[0].Slug != "hello" || s[2].SluggableID != "1" {
		t.Fatalf("slugs = %+v", s)
	}
	// Store-built finder resolves current, history, and a numeric id fallback.
	f := m.Finder(func(id string) bool { return id == "99" })
	if got, _ := f.Find("world"); got != "1" {
		t.Fatalf("finder current = %q", got)
	}
	if got, _ := f.Find("hello"); got != "1" {
		t.Fatalf("finder history = %q", got)
	}
	if got, _ := f.Find("99"); got != "99" {
		t.Fatalf("finder id = %q", got)
	}
}

func TestResolveWithStore(t *testing.T) {
	m := NewMemStore("Post")
	c := Config{}
	// First post.
	s1, err := c.Resolve("My Post", m.Exists)
	if err != nil || s1 != "my-post" {
		t.Fatalf("s1 = %q %v", s1, err)
	}
	_ = m.Assign("1", s1)
	// Second post with same title collides -> sequential.
	s2, err := c.Resolve("My Post", m.Exists)
	if err != nil || s2 != "my-post-2" {
		t.Fatalf("s2 = %q %v", s2, err)
	}
	_ = m.Assign("2", s2)
	if !m.Exists("my-post-2") {
		t.Fatal("assigned")
	}
}
