# go.thesmos.sh/kanon

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
