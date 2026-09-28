// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package frame

import (
	"fmt"

	"go.thesmos.sh/kanon"
)

// Registry maps type IDs to constructors. The application assigns the IDs
// as constants, which are part of its wire contract, and never reuses an ID
// for another type. The zero Registry is empty and ready to use. It is safe
// for concurrent calls of New once every call of Register has returned.
type Registry struct {
	constructors map[uint64]func() kanon.Message
}

// Register maps id to newMessage, a constructor of the type of the frames
// with that ID. It fails when id is registered and when newMessage is nil.
func (r *Registry) Register(id uint64, newMessage func() kanon.Message) error {
	if newMessage == nil {
		return fmt.Errorf("frame: type %d has a nil constructor", id)
	}
	if _, ok := r.constructors[id]; ok {
		return fmt.Errorf("frame: type %d is registered", id)
	}
	if r.constructors == nil {
		r.constructors = make(map[uint64]func() kanon.Message)
	}
	r.constructors[id] = newMessage
	return nil
}

// New returns a new message of the type registered under id, and false
// when none is.
func (r *Registry) New(id uint64) (kanon.Message, bool) {
	newMessage, ok := r.constructors[id]
	if !ok {
		return nil, false
	}
	return newMessage(), true
}
