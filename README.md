<p align="center"><img src="https://go-ruby-friendly-id.github.io/logo.png" alt="go-ruby-friendly-id/friendly-id" width="720"></p>

# friendly-id — go-ruby-friendly-id

[![Docs](https://img.shields.io/badge/docs-mkdocs--material-DC2626)](https://go-ruby-friendly-id.github.io/docs/)
[![License](https://img.shields.io/badge/license-BSD--3--Clause-blue)](LICENSE)
[![Go](https://img.shields.io/badge/go-1.26.4-00ADD8)](go.mod)

Pure-Go (CGO=0), MRI-faithful reimplementation of the Ruby
[**`friendly_id`**](https://github.com/norman/friendly_id) gem's slug engine — the
computation behind pretty, human-readable URL slugs — to be bound into
[**rbgo**](https://github.com/go-embedded-ruby/ruby), the pure-Go embeddable Ruby
interpreter, as a native `require "friendly_id"` module.

It reproduces the pieces of FriendlyId that are pure logic: slug **normalization**
(the ActiveSupport `parameterize` semantics — accent transliteration, downcasing,
non-alphanumeric collapsing), **candidate generation** with sequential
collision-suffixing, an optional **slug history**, and the **friendly finder**
that resolves a current slug — or an old historical slug — back to a record id,
falling back to the numeric primary key.

> **What it is — and isn't.** Everything host-specific — the base attribute to
> slugify, uniqueness lookups, and history persistence — is left as an injectable
> **seam**, so the engine runs with no database. A rbgo binding wires the seams to
> ActiveRecord; the reference `MemStore` backs tests and standalone use.

## Features

Faithful port of the `friendly_id` slug core:

- **`Parameterize(s, sep)`** — ActiveSupport-compatible slug generation: NFD accent
  transliteration (`Åccénts` → `accents`, `straße` → `strasse`, `œuvre` → `oeuvre`),
  downcase, collapse every run of non-`[a-z0-9]` to one separator, trim ends.
- **`Config.Normalize` / `Candidates` / `Resolve`** — reserved-words rejection,
  `MaxLength` truncation, the `slug_candidates` sequence, and collision resolution
  by sequential suffixing (`post`, `post-2`, `post-3`, …) with a configurable
  sequence separator.
- **History (`MemStore`)** — records each record's current slug plus every slug it
  ever held, so an old slug keeps resolving to the record (the `:history` module).
- **`Finder`** — `Model.friendly.find(input)`: current slug → historical slug →
  numeric id, returning `ErrNotFound` when none match. Each lookup is a seam.

## Usage

```go
import "github.com/go-ruby-friendly-id/friendly-id"

cfg := friendlyid.Config{Reserved: []string{"new", "edit"}}
store := friendlyid.NewMemStore("Post")

// Generate a unique slug, resolving collisions sequentially.
slug, _ := cfg.Resolve("My First Post!", store.Exists) // "my-first-post"
_ = store.Assign("1", slug)

// A second post with the same title gets a suffixed slug.
slug2, _ := cfg.Resolve("My First Post!", store.Exists) // "my-first-post-2"
_ = store.Assign("2", slug2)

// The friendly finder resolves current + historical slugs, then the id.
id, _ := store.Finder(func(id string) bool { return id == "99" }).Find("my-first-post")
// id == "1"
```

## Intended Ruby surface (via rbgo)

```ruby
class Post < ApplicationRecord
  extend FriendlyId
  friendly_id :title, use: [:slugged, :history]
end

Post.friendly.find("my-first-post")   # resolves current or historical slug
post.slug                             # "my-first-post"
post.to_param                         # the slug, for URLs
```

## Quality bar

100% test coverage (including every error branch); builds and tests green on all
six 64-bit Go architectures (amd64, arm64, riscv64, loong64, ppc64le, s390x)
plus `js/wasm` and `wasip1/wasm`; CGO=0; Go 1.26.4 floor.

## License

BSD-3-Clause © the go-ruby-friendly-id/friendly-id authors.
