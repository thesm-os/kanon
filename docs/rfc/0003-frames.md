---
rfc: 0003
title: Frames for messages on a stream
author: Roy Klopper <roy.klopper@stealthscale.io>
status: Accepted
created: 2026-09-27
updated: 2026-09-28
discussion: none
supersedes: none
superseded-by: none
produces-adr: ADR-0014, ADR-0015
---

# RFC-0003: Frames for messages on a stream

## Summary

A frame is one kanon message on a byte stream: a length, a format version, flags, a type ID,
the encoding, and an optional CRC-32C. The package `kanon/frame` writes frames to an
`io.Writer` and reads them from an `io.Reader` with one reused buffer, decodes a frame into a
`kanon.Message` without copying, and maps type IDs to constructors through a registry. Servers
of one fleet exchange frames over TCP, TLS or a message bus.

## Motivation

A kanon encoding has no length and no type of its own. Two servers on a stream need both: the
length to know where a message ends, and the type to know which struct decodes it. Without a
shared package, every service writes its own framing, and two services that frame differently
cannot talk. A frame also needs a format version, so that a fleet can change the frame layout
in a rolling upgrade, and an optional checksum for transports that do not verify integrity.

## Detailed design

### The frame

```text
frame    = length version flags type payload [checksum]
length   : uvarint, the number of bytes after itself
version  : 1 byte, value 1
flags    : 1 byte; bit 0 set when checksum is present; other bits must be 0
type     : uvarint, the type ID
payload  : the kanon encoding of the message
checksum : 4 bytes, little-endian, CRC-32C (Castagnoli) over version through payload
```

- A reader must reject a version other than 1, a flags byte with a bit above 0 set, a length
  above its limit, and a checksum mismatch.
- A payload may be empty: a message whose every field is absent is a valid frame.
- Type IDs are assigned by the application as constants and are part of its wire contract. An
  ID is never reused for another type.
- A length uvarint that runs past 10 bytes, or whose tenth byte is above 1, is malformed. A
  valid length above the reader's limit, or above what an `int` of the platform can address,
  is too large, and the reader reports it before it allocates for the frame.
- For a valid length within the limit, the reader consumes the whole frame before it reports a
  version, flag, type or checksum error, so that it stands at the next frame. A type uvarint
  that runs past the declared length, or a declared length too short for the checksum that the
  flags select, is malformed.
- A stream that ends before the first byte of a length ends between frames. A stream that ends
  inside a length or before the declared length is truncated.
- After a malformed or too large length, the reader cannot find the next frame.

### The package

```go
package frame

// Errors of Reader and Writer. A stream that ends before the first byte of
// a length returns io.EOF, and one that ends inside a length or before the
// declared length returns an error that wraps io.ErrUnexpectedEOF. Every
// error wraps one of these sentinels or the error of the stream, so that a
// caller compares with errors.Is and never parses a message.
var (
	ErrVersion    = errors.New("frame: unsupported version")
	ErrFlags      = errors.New("frame: unknown flag")
	ErrTooLarge   = errors.New("frame: longer than the limit")
	ErrChecksum   = errors.New("frame: checksum mismatch")
	ErrUnknown    = errors.New("frame: type not registered")
	ErrMalformed  = errors.New("frame: malformed length or frame")
	ErrNoFrame    = errors.New("frame: no current frame")
	ErrNilMessage = errors.New("frame: nil message")
)

// DefaultMaxSize is the frame length a Reader accepts when MaxSize is 0 or
// less.
const DefaultMaxSize = 16 << 20

// Registry maps type IDs to constructors. Registration completes before
// the first concurrent read, and the reads are then safe for concurrent
// use.
type Registry struct { /* ... */ }

// Register maps id to new, a constructor of the type. It fails when id is
// registered and when new is nil.
func (r *Registry) Register(id uint64, new func() kanon.Message) error

// New returns a new message of the type registered under id, and false
// when none is.
func (r *Registry) New(id uint64) (kanon.Message, bool)

// Writer writes frames to a stream with one buffer, which grows to the
// largest frame written. A Write into a buffer with capacity allocates
// nothing. A Writer is not safe for concurrent use.
type Writer struct {
	// Checksum adds a CRC-32C to every frame.
	Checksum bool
	/* ... */
}

// NewWriter returns a Writer that writes to w.
func NewWriter(w io.Writer) *Writer

// Write writes m as one frame of type id: it sizes m, encodes it into the
// buffer after the header, appends the checksum when Checksum is set, and
// writes the complete frame with one call of the underlying writer. It
// writes nothing when m fails to encode, and returns ErrNilMessage for a
// nil m, a typed nil pointer included, and ErrTooLarge when the length of
// the header, the payload and the checksum overflows an int. When the
// underlying writer returns an error, Write returns it. When it writes
// fewer bytes than the frame without an error, Write returns
// io.ErrShortWrite. Write does not retry, since a short write can already
// have put a prefix of the frame on the stream.
func (w *Writer) Write(id uint64, m kanon.Message) error

// Reader reads frames from a stream with one buffer. A Reader is not safe
// for concurrent use.
type Reader struct {
	// MaxSize is the longest frame the reader accepts. 0 or less means
	// DefaultMaxSize.
	MaxSize int
	/* ... */
}

// NewReader returns a Reader that reads from r.
func NewReader(r io.Reader) *Reader

// Next reads the next frame and returns its type ID and payload. The
// payload aliases the reader's buffer: a caller must not change it while
// Decode or a string that Decode returned is in use, and the next call of
// Next can overwrite it. A caller that keeps the payload longer copies it.
func (r *Reader) Next() (id uint64, payload []byte, err error)

// Decode decodes the payload of the frame that the last successful call
// of Next read, an empty payload included, into m, with the reader's
// buffer as the slab of its kanon.Options and kanon.DefaultDepth as the
// depth. It can decode that payload more than once. Strings of m alias the
// buffer until the next call of Next, and a value that is used after the
// next frame is copied with CloneKanon. Decode returns ErrNoFrame before
// the first successful Next and after a failed one, and ErrNilMessage for
// a nil m, a typed nil pointer included.
func (r *Reader) Decode(m kanon.Message) error
```

`Decode` builds the slab with `unsafe.String` over the reader's buffer, so a decode into a
reused message allocates nothing. It takes no payload argument: after the reader reuses its
buffer, an old payload slice can have the address and the length of the new one, so no check
could tell a stale payload from the current one. `Next` grows the buffer to the frame length
when it is shorter, and never above `MaxSize`. Type IDs are constants that the application
assigns. A hash of a type name changes on a rename and collides silently, so the registry
takes the ID from the application.

```mermaid
sequenceDiagram
    participant S as Sender
    participant W as frame.Writer
    participant R as frame.Reader
    participant D as Receiver
    S->>W: Write(id, m)
    W->>W: SizeKanon, EncodeKanon into the buffer, CRC-32C
    W-->>R: length version flags type payload checksum
    R->>R: Next: read length, read the frame, verify the checksum
    R-->>D: id, payload
    D->>R: Decode(m)
    R-->>D: m, strings alias the buffer
```

### Use with a registry

```go
reg := new(frame.Registry)
reg.Register(OrderID, func() kanon.Message { return new(Order) })

r := frame.NewReader(conn)
for {
	id, _, err := r.Next()
	if err != nil {
		return err
	}
	m, ok := reg.New(id)
	if !ok {
		return fmt.Errorf("%w: %d", frame.ErrUnknown, id)
	}
	if err := r.Decode(m); err != nil {
		return err
	}
	handle(m)
}
```

### Failure behaviour

- A checksum mismatch, an unknown flag, a version above 1 or an unregistered type leaves the
  reader positioned after the frame, so a caller can skip the frame and continue.
- `ErrTooLarge` for a length leaves the stream inside the frame, since the reader does not
  buffer it, and `ErrMalformed` for a length leaves it at an unknown position. The caller
  closes the stream.
- A decode error leaves the reader positioned after the frame.

### Test vectors

For type ID 1, where the checksum of the checksummed frame is the CRC-32C of `01 01 01`,
`0x24ec2a70`, little-endian:

| Frame | Bytes |
|---|---|
| Empty payload, no checksum | `03 01 00 01` |
| Payload `0a 01 61`, no checksum | `06 01 00 01 0a 01 61` |
| Empty payload, checksum | `07 01 01 01 70 2a ec 24` |

A reader must return these errors for these inputs, which tell truncation, overflow and a
valid but disallowed length apart:

| Input | Error |
|---|---|
| The length byte `80`, then the end of the stream | wraps `io.ErrUnexpectedEOF` |
| Ten length bytes `80` | `ErrMalformed` |
| `03 01 00 01` with `MaxSize` 2 | `ErrTooLarge`, before the reader buffers the frame |

The tests of the package cover an empty payload, frames with and without a checksum, a
malformed length, short reads, short writes, a truncated checksum, an unknown flag and
version, a checksum mismatch, registered and unregistered type IDs, `ErrNoFrame`,
`ErrNilMessage`, and a payload that the next frame overwrites.

## Alternatives considered

### gRPC, or another RPC framework, as the transport

**Why not:** gRPC frames, multiplexes and flow-controls, and it needs HTTP/2 and its
dependency tree. A fleet that already runs gRPC plugs kanon in through a codec adapter of six
lines, and a fleet that streams over TCP or a message bus needs only a frame.

### A fixed-width length and type

A 4-byte length and a 4-byte type ID, as many protocols use.

**Why not:** 8 bytes per frame against 2 to 3 for small frames, which are most of a fleet's
traffic. The uvarint costs one branch per frame.

### No version byte

**Why not:** a frame layout that cannot change without a flag day is a frame layout that
never changes.

### Checksum always on

**Why not:** TLS and TCP verify integrity already, and a checksum costs time per byte and 4
bytes per frame. A transport over a plain pipe or a file turns it on.

## Drawbacks

- 4 to 6 bytes per frame: a length of 1 or 2 bytes, the version, the flags and a type of 1 or
  2 bytes, plus 4 bytes for the checksum.
- The reader's buffer aliasing makes a decoded message invalid after the next `Next`, which is
  a use-after-read bug that the race detector does not catch. The documentation and a
  conformance check for `CloneKanon` are the defence.
- One package of about 250 lines with about 300 lines of tests, at 100% coverage and
  mutation.

## Unresolved and future work

- A gRPC `encoding.Codec` adapter as a separate module.
- Compression of a stream of frames, which the transport layer provides.

## References

| What | Where |
|---|---|
| gRPC codec interface, for the adapter | google.golang.org/grpc v1.84.0, encoding/encoding.go |
| CRC-32C, the Castagnoli polynomial | `hash/crc32.Castagnoli`, Go 1.27.1 |
