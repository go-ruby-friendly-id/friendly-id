// Copyright (c) 2026, the go-ruby-friendly-id/friendly-id authors
//
// SPDX-License-Identifier: BSD-3-Clause

package friendlyid

import "errors"

var (
	// ErrBlank is returned when a base string normalizes to an empty slug.
	ErrBlank = errors.New("friendlyid: blank slug")
	// ErrReserved is returned when a normalized slug is on the reserved-words list.
	ErrReserved = errors.New("friendlyid: reserved slug")
	// ErrNilExists is returned when Resolve is called without an existence check.
	ErrNilExists = errors.New("friendlyid: nil exists function")
	// ErrNotFound is the friendly finder's RecordNotFound analogue.
	ErrNotFound = errors.New("friendlyid: record not found")
	// ErrTaken is returned when assigning a slug already current for another record.
	ErrTaken = errors.New("friendlyid: slug already taken")
)
