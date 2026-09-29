// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package inspect

import (
	"bytes"
	"cmp"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"go.thesmos.sh/kanon"
	"go.thesmos.sh/kanon/batch"
	"go.thesmos.sh/kanon/frame"
)

// noteStart begins a line of the text that describes a frame, a batch or a
// record, or states an error, as a comment of protoscope.
const noteStart = "# "

// Words of the labels of frames and records.
const (
	frameWord  = "frame "
	recordWord = "record "
)

// framesDoc is the JSON document of a stream of frames.
type framesDoc struct {
	Frames []frameJSON `json:"frames"`
}

// frameJSON is the JSON of one frame: its index, its offset and its size in
// the input with its length; whether it has a checksum, the type ID and
// the fields of the payload of a frame that reads; and the error of a frame
// that fails its version, its flags or its checksum, or whose payload does
// not parse.
type frameJSON struct {
	Frame    int     `json:"frame"`
	Offset   int     `json:"offset"`
	Size     int     `json:"size"`
	Checksum *bool   `json:"checksum,omitempty"`
	Type     *uint64 `json:"type,omitempty"`
	Fields   []any   `json:"fields,omitzero"`
	Error    string  `json:"error,omitempty"`
}

// batchDoc is the JSON document of a batch: the length in bytes of the
// offsets of its index, and its records.
type batchDoc struct {
	Width   int          `json:"width"`
	Records []recordJSON `json:"records"`
}

// recordJSON is the JSON of one record of a batch: its index, its offset in
// the batch and its size, the fields of its encoding, and the error of an
// encoding that does not parse.
type recordJSON struct {
	Record int    `json:"record"`
	Offset int    `json:"offset"`
	Size   int    `json:"size"`
	Fields []any  `json:"fields"`
	Error  string `json:"error,omitempty"`
}

// frameEntry is one frame of a stream: its index, and its offset and its
// size in the input with its length. A frame that reads has its checksum
// state, its type ID and the fields of its payload, which alias the buffer
// of the reader until its next frame, and err is the error of the payload.
// For a frame that does not read, err is the error of the reader.
type frameEntry struct {
	index, offset, size int
	read, checksummed   bool
	id                  uint64
	fields              []Field
	err                 error
}

// inspectFrames writes the frames of the stream in data to stdout in the
// format out, and the error that ends the stream to stderr, and returns the
// exit status of [Command]. It reads the frames with a frame.Reader whose
// limit is the length of data. A frame that fails its version, its flags or
// its checksum prints with its error, and the reader continues after it, as
// an application's reader does. A payload that does not parse prints its
// fields before the error, and its error. The reader stops at the end of
// data, and at a length that is malformed, above the limit or cut short.
func inspectFrames(data []byte, out format, stdout, stderr io.Writer) int {
	in := bytes.NewReader(data)
	r := frame.NewReader(in)
	r.MaxSize = len(data)
	var text strings.Builder
	frames := []frameJSON{}
	status := exitOK
	var end error
	for k := 0; ; k++ {
		start := len(data) - in.Len()
		id, payload, err := r.Next()
		e := frameEntry{index: k, offset: start, size: len(data) - in.Len() - start, id: id, err: err}
		if err == nil {
			e.read, e.checksummed = true, r.Checksummed()
			e.fields, e.err = parse(payload, kanon.DefaultDepth, frameWord+strconv.Itoa(k))
		} else if errors.Is(err, io.EOF) {
			break
		} else if !skipped(err) {
			end = fmt.Errorf("%s%d at %d: %w", frameWord, k, start, err)
			break
		}
		if e.err != nil {
			status = exitFail
		}
		if out == formatJSON {
			frames = append(frames, frameJSONOf(e))
		} else {
			appendFrameText(&text, e)
		}
	}
	var werr error
	if out == formatJSON {
		werr = writeJSON(stdout, framesDoc{Frames: frames})
	} else {
		_, werr = io.WriteString(stdout, text.String())
	}
	if err := cmp.Or(werr, end); err != nil {
		return report(stderr, err)
	}
	return status
}

// skipped reports whether err is an error of a frame.Reader after which the
// reader is positioned after the frame: frame.ErrVersion, frame.ErrFlags and
// frame.ErrChecksum.
func skipped(err error) bool {
	return errors.Is(err, frame.ErrVersion) || errors.Is(err, frame.ErrFlags) || errors.Is(err, frame.ErrChecksum)
}

// frameJSONOf returns the JSON of the frame e.
func frameJSONOf(e frameEntry) frameJSON {
	f := frameJSON{Frame: e.index, Offset: e.offset, Size: e.size}
	if e.read {
		f.Checksum, f.Type, f.Fields = &e.checksummed, &e.id, jsonFields(e.fields)
	}
	if e.err != nil {
		f.Error = e.err.Error()
	}
	return f
}

// appendFrameText appends the text of the frame e to b: a note of its index,
// its offset, its size, its checksum state and its type ID, the fields of
// its payload and a note of the error of the payload, or a note of the error
// of the frame.
func appendFrameText(b *strings.Builder, e frameEntry) {
	head := frameWord + strconv.Itoa(e.index) + " at " + strconv.Itoa(e.offset) + ": " + byteCount(e.size)
	if !e.read {
		appendNote(b, head+": "+e.err.Error())
		return
	}
	appendNote(b, head+", "+checksumState(e.checksummed)+", type "+strconv.FormatUint(e.id, 10))
	appendText(b, e.fields, "")
	if e.err != nil {
		appendNote(b, e.err.Error())
	}
}

// inspectBatch writes the batch in data to stdout in the format out: the
// width of its offsets, and each record with its offset, its size and its
// fields. It writes the error of a batch whose layout does not check to
// stderr, and returns the exit status of [Command]. It checks the batch with
// batch.ParseAlias. A record whose encoding does not parse prints its fields
// before the error, and its error.
func inspectBatch(data []byte, out format, stdout, stderr io.Writer) int {
	b, err := batch.ParseAlias(data)
	if err != nil {
		return report(stderr, err)
	}
	var text strings.Builder
	appendNote(&text, "batch: "+recordCount(b.Len())+", "+strconv.Itoa(b.Width())+"-byte offsets")
	records := make([]recordJSON, 0, b.Len())
	status := exitOK
	for i := range b.Len() {
		record := b.Record(i)
		fields, perr := parse(record, kanon.DefaultDepth, recordWord+strconv.Itoa(i))
		if perr != nil {
			status = exitFail
		}
		if out == formatJSON {
			rj := recordJSON{Record: i, Offset: b.Offset(i), Size: len(record), Fields: jsonFields(fields)}
			if perr != nil {
				rj.Error = perr.Error()
			}
			records = append(records, rj)
			continue
		}
		appendNote(&text, recordWord+strconv.Itoa(i)+" at "+strconv.Itoa(b.Offset(i))+": "+byteCount(len(record)))
		appendText(&text, fields, "")
		if perr != nil {
			appendNote(&text, perr.Error())
		}
	}
	var werr error
	if out == formatJSON {
		werr = writeJSON(stdout, batchDoc{Width: b.Width(), Records: records})
	} else {
		_, werr = io.WriteString(stdout, text.String())
	}
	if werr != nil {
		return report(stderr, werr)
	}
	return status
}

// appendNote appends a line of text to b as a comment of protoscope.
func appendNote(b *strings.Builder, text string) {
	b.WriteString(noteStart)
	b.WriteString(text)
	b.WriteByte('\n')
}

// byteCount returns n with the word bytes, or byte for 1.
func byteCount(n int) string {
	if n == 1 {
		return "1 byte"
	}
	return strconv.Itoa(n) + " bytes"
}

// checksumState returns the words of the checksum state of a frame that
// reads: checksum for a frame that has one, and no checksum otherwise.
func checksumState(checksummed bool) string {
	if checksummed {
		return "checksum"
	}
	return "no checksum"
}

// recordCount returns n with the word records, or record for 1.
func recordCount(n int) string {
	if n == 1 {
		return "1 record"
	}
	return strconv.Itoa(n) + " records"
}
