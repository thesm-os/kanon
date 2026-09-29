// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package validate

import (
	"errors"
	"slices"
	"strings"
)

// Limits of the containers of the package.
const (
	// blobLimit is the length of the longest valid Blob.
	blobLimit = 200
	// hashTaken is the first byte that no valid Hash starts with.
	hashTaken = 0xff
)

// Errors of the valid methods of the containers.
var (
	// ErrTagsNUL is the error of Tags.valid for a tag with a NUL byte.
	ErrTagsNUL = errors.New("validate: tag contains a NUL byte")
	// ErrScoresNegative is the error of Scores.valid for a negative score.
	ErrScoresNegative = errors.New("validate: score is negative")
	// ErrHashTaken is the error of Hash.valid for a hash that starts with
	// the byte 0xff.
	ErrHashTaken = errors.New("validate: hash starts with 0xff")
	// ErrBlobLength is the error of Blob.valid for a blob longer than 200
	// bytes.
	ErrBlobLength = errors.New("validate: blob longer than 200 bytes")
	// ErrSpanOrder is the error of Span.valid for a span whose low bound is
	// above its high bound.
	ErrSpanOrder = errors.New("validate: span low bound above its high bound")
)

// Tags is a slice of tags without a NUL byte.
type Tags []string

// valid returns ErrTagsNUL for a tag with a NUL byte.
func (t Tags) valid() error {
	if slices.ContainsFunc(t, func(s string) bool { return strings.ContainsRune(s, 0) }) {
		return ErrTagsNUL
	}
	return nil
}

// Scores maps names to scores that are not negative.
type Scores map[string]int32

// valid returns ErrScoresNegative for a negative score.
func (s Scores) valid() error {
	for _, v := range s {
		if v < 0 {
			return ErrScoresNegative
		}
	}
	return nil
}

// Hash is four bytes that do not start with the byte 0xff.
type Hash [4]byte

// valid returns ErrHashTaken for a hash that starts with the byte 0xff.
func (h Hash) valid() error {
	if h[0] == hashTaken {
		return ErrHashTaken
	}
	return nil
}

// Blob is at most 200 bytes.
type Blob []byte

// valid returns ErrBlobLength for a blob longer than 200 bytes.
func (b Blob) valid() error {
	if len(b) > blobLimit {
		return ErrBlobLength
	}
	return nil
}

// Span is a low and a high bound, in that order, whose low bound is at most
// its high bound.
type Span [2]int32

// valid returns ErrSpanOrder for a span whose low bound is above its high
// bound.
func (s Span) valid() error {
	if s[0] > s[1] {
		return ErrSpanOrder
	}
	return nil
}
