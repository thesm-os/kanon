---
rfc: 0005
title: Inspecting encodings, frames and batches without their types
author: Roy Klopper <roy.klopper@stealthscale.io>
status: Draft
created: 2026-09-29
updated: 2026-09-29
discussion: none
supersedes: none
superseded-by: none
produces-adr: tbd
---

# RFC-0005: Inspecting encodings, frames and batches without their types

## Summary

`go tool kanon inspect` prints the fields of a kanon encoding without the Go types that wrote
it: each field's number, wire format and value, with every reading its bytes allow. It also
reads a stream of frames and a batch, and prints each frame or record with its fields. The
output uses the text notation of protoscope, since the tags and wire formats are protobuf's,
and `-json` writes every reading for other tools. The command is a subcommand of the
generator, so every module that pins the generator with a `tool` directive has it.

## Motivation

Debugging a production fault often starts with bytes, such as a frame that a reader rejected or
a record with an unexpected value. Reading them takes the Go types of the writer at the
writer's version, and a program that decodes with them. Without those types, or when the bytes
do not decode, an engineer reads the hex by hand. Every length prefix and zigzag varint is then
a chance to misread.

Each field starts with a tag that gives its number and its wire format. The wire format gives
the length of the value, so a reader finds every field and its bytes without the types. It
cannot know what the bytes mean, but it can show every reading that they allow. Protobuf's tags
work the same way. `protoc --decode_raw` prints the tag and value pairs of a raw protobuf
message, and protoscope prints them in a text notation. Neither tool knows the frame header,
its checksum or the batch index. Both are separate installs.

The inspector belongs to the generator's command because that command is already pinned in
every module that uses kanon: `go tool kanon inspect` runs the version of the format that the
module encodes with.

## Detailed design

### Command

```text
kanon inspect [-frames | -batch] [-hex] [-json] [file]
```

- The input is the file, or standard input when no file is named.
- Without `-frames` or `-batch`, the input is one struct encoding.
- `-frames` reads the input as a stream of frames. It prints each frame's index, its offset and
  its size in the input with its length prefix, whether it has a checksum, and its type
  ID, followed by its payload as a struct encoding.
- `-batch` reads the input as one batch. It prints the number of records and the width of the
  offsets, followed by each record's index, offset and length, and the record as a struct
  encoding.
- `-hex` reads the input as hexadecimal digits and ignores whitespace, so that a vector such
  as `0a 01 78 10 0e` pastes as it is written.
- `-json` writes JSON instead of the text notation.

The exit status is 0 when the whole input parses, 1 when a part of it does not, and 2 for a
usage error. An error that the input continues after prints in the output at its place, such
as a frame that fails its checksum or a record that does not parse. A malformed struct encoding
and a frame that the input cuts short end the output, and the command writes their error to
its standard error. Each error gives the byte offset and the cause: truncation, a malformed tag
or length, or the error of the frame or batch reader.

When the first argument is `inspect`, the generator runs the inspector with the other
arguments. Any other first argument keeps its meaning. A source file ends in `.go`, so no
existing invocation changes.

### Readings of a field

The tag gives each field its number and wire format. The value prints as every reading its
wire format allows:

| Wire | Text | Readings |
|---|---|---|
| varint | `2: 14` | the unsigned value, and in a comment the zigzag value for a value other than 0, since every signed integer type encodes as zigzag |
| fixed64 | `4: 2i64` | the unsigned value, and in a comment the signed value when it is negative, and the float64 |
| fixed32 | `6: 7i32` | the unsigned value, and in a comment the signed value when it is negative, and the float32 |
| bytes | `1: {...}` | the first reading of: a struct encoding, `{}` for no bytes, text, or hexadecimal digits |

A bytes value reads as a struct encoding when it is not empty and parses to its end under the
rules of a decode that skips unknown fields: every tag has a field number above 0 and a valid
wire format, every value ends within the bytes value, and no byte is left over. It then prints
its fields in braces, one level deeper. A bytes value that does not parse reads as text when it
is valid UTF-8 without control characters other than tab and newline, as `{"text"}`, and
otherwise as hexadecimal digits, as ``{`010002`}``. A comment on a struct reading gives the
text when the value is also text. A comment on hexadecimal digits gives the value as a run of
varints and their zigzag values when it is one.

Text uses the escapes of protoscope: `\\`, `\"`, `\n`, and `\x09` for a tab. A hex literal of
protoscope takes no spaces. protoscope assembles the text into the input when every varint of
the input has its shortest form, as an encoder writes it. A varint in a longer form prints as
its value, which protoscope assembles in the shortest form.

The readings do not know the schema:

- A slice of integers prints as hexadecimal digits, with its varints in the comment.
- A map prints its keys and values as the fields or bytes they parse as.
- A time prints as a struct of up to three fields.
- A bytes value that happens to parse as a struct prints as one. A short string such as `hi`
  does, since `68 69` is field 13 with the varint 105. The comment keeps the text, and `-json`
  gives every reading.

Struct readings nest to `kanon.DefaultDepth` levels below the input at most. A deeper bytes
value prints as text or as hexadecimal digits.

### Examples

A struct encoding of the wire format's test vectors, `{Label: "x", Count: 7}`:

```text
$ echo '0a 01 78 10 0e' | go tool kanon inspect -hex
1: {"x"}
2: 14  # zigzag 7
```

A frame of the frame vectors, type 1 with the payload `0a 01 61`:

```text
$ echo '06 01 00 01 0a 01 61' | go tool kanon inspect -frames -hex
# frame 0 at 0: 7 bytes, no checksum, type 1
1: {"a"}
```

A batch of the batch vectors, one record `08 02`:

```text
$ echo '01 00 08 02 00 00 00 00 01 00 00 00' | go tool kanon inspect -batch -hex
# batch: 1 record, 4-byte offsets
# record 0 at 2: 2 bytes
1: 2  # zigzag 1
```

The JSON of the first example:

```json
{
  "fields": [
    {
      "field": 1,
      "wire": "bytes",
      "offset": 0,
      "length": 1,
      "hex": "78",
      "text": "x",
      "varints": [
        {
          "unsigned": 120,
          "zigzag": 60
        }
      ]
    },
    {
      "field": 2,
      "wire": "varint",
      "offset": 3,
      "unsigned": 14,
      "zigzag": 7
    }
  ]
}
```

In JSON, a fixed64 and a fixed32 have `unsigned`, `signed` and `float`, with the float as a
string, since JSON has no NaN or infinity. The JSON of a stream is `{"frames": [...]}`: each
frame has `frame`, `offset` and `size`, and `checksum` and `type` with `fields`, or an
`error`, or both for a payload that does not parse. The JSON of a batch is
`{"width": 4, "records": [...]}`: each record has `record`, `offset`, `size` and `fields`, and
an `error` when it does not parse.

### Frames and batches

The inspector reads frames with `frame.Reader` and batches with `batch.ParseAlias`, so the
header, the checksum and the index follow the same rules as they do in an application. The
limit of the frame reader is the length of the input, which no frame can exceed. The offset and
the size of a frame are the positions of the reader in the input before and after the frame.
A frame that fails its version, flags or checksum prints its offset, its size and the error,
and the inspector continues with the next frame, as the reader positions itself after such a
frame. A malformed or too long length, and a frame that the input cuts short, end the output,
as they end a reader. A batch whose layout fails prints the error of `batch.ParseAlias` and
nothing else.

`Reader.Checksummed` reports whether a frame that reads has a checksum. `Batch.Width`
returns the width of the offsets of a batch, and `Batch.Offset` the offset of a record in the
batch. The version of a frame that reads is 1, since the reader rejects any other, so the
output leaves it out.

### Package

The command calls one internal package. It has no public API, since the command is the
interface:

```go
package inspect

// Field is one field of a struct encoding, with the struct reading of a
// bytes value.
type Field struct {
	// Number is the field number of the tag, and Wire its wire format:
	// wire.Varint, wire.Fixed64, wire.Bytes or wire.Fixed32.
	Number uint64
	Wire   uint64
	// Offset is the offset of the tag in the input of Parse.
	Offset int
	// Value is the bytes of the value after the tag, without the length of
	// a bytes value. It aliases the input of Parse.
	Value []byte
	// Fields is the struct reading of a bytes value: the fields that Value
	// parses to, with offsets in Value. It is nil for any other value, and
	// for a bytes value that is empty, that does not parse to its end, or
	// that lies deeper than the depth of Parse.
	Fields []Field
}

// Parse returns the fields of the struct encoding in data, with the struct
// reading of every bytes value down to depth levels below data. On
// malformed input it returns the fields before the malformed one and a
// *kanon.DecodeError at the offset of its tag.
func Parse(data []byte, depth int) ([]Field, error)

// WriteText writes fields in the text notation of protoscope, and
// WriteJSON as JSON. Both return the error of the write.
func WriteText(w io.Writer, fields []Field) error
func WriteJSON(w io.Writer, fields []Field) error

// Command runs the inspect subcommand with the arguments after the word
// inspect, and returns its exit status.
func Command(args []string, stdin io.Reader, stdout, stderr io.Writer) int
```

`Parse` reads each tag with `wire.Uvarint` and skips each value with `wire.Skip`, as a decode
skips an unknown field. Its errors are the errors of that decode. They name `input` as the
location for the input of `Parse`, and `frame N` or `record N` for a payload or a record.

### Tests

- The test vectors of the wire format, of frames and of batches, with their text output in the
  tests and their JSON output in golden files.
- Truncated and malformed inputs, with the fields before the error and its offset.
- A fuzz target over `Parse` and the writers: no input panics, the JSON is valid, and `Parse`
  reads every encoding that the generated code of a fixture writes, in ascending field number.
- 100% line coverage and mutation, as for every package of the module. The loops of `Parse`
  and of the varint reading run a number of iterations fixed before they start, so that a
  mutant of the loop fails instead of hanging.

## Alternatives considered

### A separate command and a public package

`cmd/kanon-inspect`, with the reader in `go.thesmos.sh/kanon/inspect`, so that an application
can print an encoding from code, such as in a debug endpoint.

**Why not:** a second binary to install and to pin, and a public API to keep compatible. The
subcommand is part of the generator that each module pins already. A public package can
export the internal one when an application needs it.

### Protoscope or `protoc --decode_raw`

The tags and wire formats are protobuf's, so both tools already print a kanon struct encoding.

**Why not:** neither reads the frame header, the checksum or the batch index. Neither shows the
zigzag reading that every signed integer needs. Both are separate installs, and the protoscope
repository is archived. The inspector adopts protoscope's notation, so that its output reads
the same and protoscope can assemble test input from it.

### Inspection with the types

The inspector loads the package of a type, as the generator does, and decodes with the
generated code, to print field names and typed values.

**Why not:** it needs the source of the writer's types at the writer's version, which is what
an engineer lacks when the inspector helps most. A program with the types decodes with them
already.

## Drawbacks

- The generator's command gains a subcommand, and its binary the inspector's code: 860 lines
  in the internal package, 230 of them comments, and 30 in the command, with 900 lines of
  tests and 180 lines of golden files.
- A reading without the schema can mislead: bytes that parse as a struct print as one. The
  comment and the JSON keep the other readings, but the text shows one first.
- The payload of a frame that fails its checksum does not print, because the frame reader
  returns no payload for it.
- The frame and batch packages export three methods for the inspector: `Reader.Checksummed`,
  `Batch.Width` and `Batch.Offset`.

## Unresolved and future work

- Printing the payload of a frame that fails its checksum needs the frame reader to return the
  frame with the error. That is a change to the frame package, not proposed here.
- A flag that names type IDs, such as `-types 1=Order`, to label frames.
- A public package, if an application needs to print encodings from code.

## References

| What | Where |
|---|---|
| `protoc --decode_raw`, raw tag and value pairs | protoc help text, protobuf compiler |
| Protoscope, its text notation and its heuristic disassembler | https://github.com/protocolbuffers/protoscope, language.txt |
| Protoscope's escapes and hex literals | scanner.go at commit 8e7a6aafa2c9 |
