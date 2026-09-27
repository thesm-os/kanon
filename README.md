# kanon

A Go project using [ergon](https://go.thesmos.sh/ergon) for the
build, test, lint, and release lifecycle.

## Decoding without a copy

The codec that kanon generates for a struct type has a `DecodeKanon(data, slab, slabOff)` method.
`data` starts at offset `slabOff` of `slab`.
Every string that `DecodeKanon` decodes is a substring of `slab`.
`UnmarshalBinary` passes `string(data)`, a copy of its input.
For a type with strings, that copy is one allocation per decode.
If you control the input buffer, pass the slab yourself.

### Alias the input buffer

`unsafe.String` returns a slab that shares the memory of `data`.
The decode then does not allocate a slab:

```go
slab := unsafe.String(unsafe.SliceData(data), len(data))
if err := m.DecodeKanon(data, slab, 0); err != nil {
	return err
}
```

Leave `data` unchanged while `m` or a string decoded from it is in use.

### Share one slab across a batch

If you decode many values from one buffer, copy the buffer into one slab and pass the offset of each value:

```go
slab := string(batch)
for _, r := range records { // r.off and r.n locate one encoding in batch
	if err := m.DecodeKanon(batch[r.off:r.off+r.n], slab, r.off); err != nil {
		return err
	}
}
```

The batch then allocates one slab.
The offset in an error is the offset in `batch`.

### Hazards

- A change to an aliased buffer changes every string decoded from it.
  If you reuse the buffer for the next read, the values of the previous read change without an error.
- A string map key decoded from an aliased buffer changes with the buffer.
  The map computed the hash of the key from its old bytes, so a lookup by the old bytes or by the new ones can miss it.
- The garbage collector frees a slab only after every string decoded from it is unreachable.
  A value that retains one short string from a batch prevents the collector from freeing the whole batch.
  If you retain a decoded string, retain the copy that `strings.Clone` returns.

Decoded byte slices do not share memory with `data`.
`DecodeKanon` copies each one into the memory of the receiver.

## Development

```
make bootstrap      # install dev tools
make install        # go mod download
make check          # full pre-merge gate (mod verify + lint + test + checks)
make test-386       # run the tests where int, uint and uintptr are 32 bits wide
make check-numbers  # fail when a change renumbers a field of BASE (default origin/main)
make build          # compile every module's source
```

`make help` lists every target.

## License

See [LICENSE](LICENSE).
