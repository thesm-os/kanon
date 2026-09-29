// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanon

import (
	"errors"
	"fmt"
	"maps"
	"slices"
)

// Nouns that name the entries of a numbers line in the errors of [Check]:
// the fields of a struct and the concrete types of a list.
const (
	fieldNoun = "field"
	typeNoun  = "type"
)

// Check checks the numbers that [Generate] records for the struct types that
// opts.Types names, from the kanon directive of the Go source file named
// file of the package in dir, against the numbers that the code files of a
// base revision record. base maps the name of each code file of the package
// at that revision to its content. Every struct and every list of concrete
// types that base records must follow these rules, so that data that the
// code of the base revision encoded decodes into the same fields and types:
//
//   - A field keeps its number.
//   - The number of a removed field is reserved, or a field whose tag names
//     the number takes it.
//   - A reserved number is still reserved, or a field whose tag names the
//     number takes it.
//   - A concrete type keeps its number. The number of a removed concrete
//     type and a reserved number are still reserved.
//
// The rules also apply to a struct of another package that base records as
// an inline struct, and that a field now encodes through the struct's own
// kanon codec: the numbers line that the code files of its package record
// for it must follow them. Such a struct whose package records no numbers
// for it fails, since Check cannot compare them.
//
// A struct, a list, a field or a concrete type that base does not record
// passes. Check returns one error per broken rule, joined. It fails as
// Generate fails, apart from the rendering of the files. It also fails when
// a code file of base or of the package of such a struct does not parse,
// and when two code files of base other than the code file of file record
// different numbers for a struct or a list.
func Check(dir, file string, opts Options, base map[string][]byte) error {
	u, err := newUnit(dir, file, opts)
	if err != nil {
		return err
	}
	own, others, err := baseRecords(codeName(file), base)
	if err != nil {
		return err
	}
	var errs []error
	for _, m := range u.numbered {
		rec, err := recordFor(m.recordKeys(), own, others)
		if err != nil {
			return err
		}
		errs = append(errs, fieldNumbers(m).against(rec)...)
	}
	for _, l := range u.lists {
		rec, err := recordFor(l.recordKeys(), own, others)
		if err != nil {
			return err
		}
		errs = append(errs, typeNumbers(l).against(rec)...)
	}
	for _, key := range slices.Sorted(maps.Keys(u.codecs)) {
		rec, err := recordFor([]string{key}, own, others)
		if err != nil {
			return err
		}
		if len(rec.order) == 0 && len(rec.reserved) == 0 {
			continue
		}
		obj := u.codecs[key].Obj()
		recs, err := u.pkg.depRecords(obj.Pkg().Path())
		if err != nil {
			return err
		}
		now := recs[obj.Name()]
		if now == nil {
			errs = append(errs, fmt.Errorf("kanon: %s encodes through a kanon codec whose package records no "+
				"numbers for it, and the base revision records its fields: kanon cannot check them", key))
			continue
		}
		errs = append(errs, recordNumbers(key, now).against(rec)...)
	}
	return errors.Join(errs...)
}

// numbers is the numbering of the fields of a struct, or of the concrete
// types of a list, in the working tree, which [Check] compares with the
// record of the base revision.
type numbers struct {
	// key names the struct or the list in its numbers line.
	key string
	// noun names the entries in errors: fieldNoun or typeNoun.
	noun string
	// nums maps the name of each field or concrete type to its number.
	nums map[string]int
	// names maps each number of nums to its field or concrete type.
	names map[int]string
	// tagged marks the numbers that the tags of fields name.
	tagged map[int]bool
	// reserved marks the reserved numbers.
	reserved map[int]bool
}

// fieldNumbers returns the numbering of the fields of m.
func fieldNumbers(m *target) *numbers {
	n := newNumbers(m.key, fieldNoun, m.reserved)
	for _, f := range m.fields {
		n.add(f.name, f.num, f.tag.num != 0)
	}
	return n
}

// recordNumbers returns the numbering of the fields that rec, a numbers
// line of the code file of another package, records, under key. The line
// does not state which numbers a tag names, so no number counts as tagged.
func recordNumbers(key string, rec *record) *numbers {
	n := newNumbers(key, fieldNoun, rec.reserved)
	for _, name := range rec.order {
		n.add(name, rec.nums[name], false)
	}
	return n
}

// typeNumbers returns the numbering of the concrete types of l. No tag
// names the number of a concrete type.
func typeNumbers(l *typeList) *numbers {
	n := newNumbers(l.key(), typeNoun, l.reserved)
	for _, c := range l.types {
		n.add(c.name, c.num, false)
	}
	return n
}

// newNumbers returns the numbering under key, whose entries noun names in
// errors, with the numbers reserved and no entry.
func newNumbers(key, noun string, reserved []int) *numbers {
	n := &numbers{
		key:      key,
		noun:     noun,
		nums:     make(map[string]int),
		names:    make(map[int]string),
		tagged:   make(map[int]bool),
		reserved: make(map[int]bool, len(reserved)),
	}
	for _, num := range reserved {
		n.reserved[num] = true
	}
	return n
}

// add gives the entry name the number num, which a tag names when tagged is
// set.
func (n *numbers) add(name string, num int, tagged bool) {
	n.nums[name] = num
	n.names[num] = name
	n.tagged[num] = tagged
}

// against returns one error for each rule of [Check] that n breaks against
// rec, the record of the base revision, in the order of rec: an entry with
// another number, and a number of a removed entry or a reserved number that
// n neither reserves nor gives a field by its tag.
func (n *numbers) against(rec *record) []error {
	var errs []error
	for _, name := range rec.order {
		was := rec.nums[name]
		now, ok := n.nums[name]
		if !ok {
			if err := n.kept(was, "gives the removed "+n.noun+" "+name); err != nil {
				errs = append(errs, err)
			}
			continue
		}
		if now != was {
			errs = append(errs, fmt.Errorf(
				"kanon: %s %s of %s has number %d, and the base revision gives it number %d",
				n.noun,
				name,
				n.key,
				now,
				was,
			))
		}
	}
	for _, num := range rec.reserved {
		if err := n.kept(num, "reserves"); err != nil {
			errs = append(errs, err)
		}
	}
	return errs
}

// kept returns nil when n reserves num or gives it to a field whose tag
// names it. Otherwise the error cites the entry that takes num, or states
// that n does not reserve num. why states what the base revision does with
// num.
func (n *numbers) kept(num int, why string) error {
	if n.reserved[num] || n.tagged[num] {
		return nil
	}
	if name, ok := n.names[num]; ok {
		return fmt.Errorf(
			"kanon: %s %s of %s takes number %d, which the base revision %s",
			n.noun,
			name,
			n.key,
			num,
			why,
		)
	}
	return fmt.Errorf("kanon: %s does not reserve number %d, which the base revision %s", n.key, num, why)
}
