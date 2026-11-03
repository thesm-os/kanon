# go.thesmos.sh/kanon

## 0.5.0

### Minor Changes

- 2551cb7: Add the `kanontest` checks that a decode keeps the strings that its input repeats, copies its input once for the strings that change, and takes every string from a slab.
- 2551cb7: Add `wire.Slab`, the slab of the strings of one decode, which the generated decode functions share.
- 2551cb7: Keep each string of the receiver that the input repeats in a decode with the zero `kanon.Options`, and copy the input once for the first string that changes.

### Patch Changes

- 57b7737: Require `go.dokimi.dev/assert` v0.16.0.

## 0.4.0

### Minor Changes

- 77e2bf6: Add `kanon.Strict`, whose method `StrictKanon` marks a type whose decode method accepts only the bytes that its encode method writes, so that a canonical decode does not encode its values again.
- 77e2bf6: Add `kanontest.StrictChecks` and `kanontest.RunStrict`, which check the guarantee of `kanon.Strict`.

### Patch Changes

- 77e2bf6: Check the error of the decode method and the presence rule of a canonical field of a type that encodes itself in one statement.
- dac27e4: Fix `kanontest` to probe the field order of an inline struct whose table samples set one field each.

## 0.3.1

### Patch Changes

- 236ecc6: Fix the canonical decode of a validated field to return `kanon.ErrNotCanonical` for an explicit zero before it calls `ValidateKanon`.
- 685fdc4: Fix the reference decode of `kanontest` to apply the presence rule of a canonical field before `ValidateKanon`.
- 7275b48: Require `go.dokimi.dev/assert` v0.10.0.

## 0.3.0

### Minor Changes

- b0566b4: Bound the length of a string or a byte slice field with the tag option `max`.

### Patch Changes

- b0566b4: Reword `kanon.ErrMax` and the detail of `wire.MaxError` as "longer than the max" for every kind of bounded field.

## 0.2.0

### Minor Changes

- 5c99023: **Breaking:** Return the value alone from each index method whose read cannot fail, and check the range of each integer narrower than 64 bits in `IndexKanon`.

## 0.1.0

### Minor Changes

- a08ae56: Add the kanon generator of binary codecs for Go struct types, with the packages kanon, wire, frame, batch and kanontest.
