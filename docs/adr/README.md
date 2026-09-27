# ADRs

An ADR records one decision. The argument for it is in the RFC that its frontmatter names,
under `../rfc`.

| ADR | Title | Status |
|---|---|---|
| [0001](0001-tagged-fields-over-positional.md) | Encode structs as tagged fields with locked numbers | Accepted |
| [0002](0002-varints-by-default-fixed-opt-in.md) | Encode integers as varints by default, fixed widths per field on request | Accepted |
| [0003](0003-accept-non-minimal-varints.md) | Decoders accept non-minimal varints | Accepted |
| [0004](0004-no-version-in-the-encoding.md) | The encoding carries no version | Accepted |
| [0005](0005-map-keys-in-a-total-order.md) | Write map entries in a total key order, struct keys by field number | Accepted |
| [0006](0006-time-as-seconds-nanos-and-zone.md) | Encode a time as seconds, nanoseconds and zone offset | Accepted |
| [0007](0007-go-structs-as-the-schema.md) | Go struct types are the schema, generated in stringer's model | Accepted |
| [0008](0008-lock-field-numbers-and-check-them.md) | Lock field numbers in the generated file and check them against a revision | Accepted |
| [0009](0009-generated-code-imports-a-runtime.md) | Generated code imports a small runtime | Accepted |
| [0010](0010-nesting-limit-on-decode.md) | Bound the nesting depth of a decode, 100 levels by default | Accepted |
| [0011](0011-decode-options-in-one-struct.md) | Pass decode options in one struct | Accepted |
| [0012](0012-unknown-fields-retention-and-rule.md) | Keep unknown fields through an opt-in field, and rewrite records as opaque bytes otherwise | Accepted |
| [0013](0013-generated-views-for-single-fields.md) | Generate view types that read one field from an encoding | Accepted |
| [0014](0014-frame-layout.md) | Frame messages on a stream with length, version, flags, type ID and optional CRC-32C | Accepted |
| [0015](0015-type-ids-are-assigned-constants.md) | Frame type IDs are constants the application assigns | Accepted |
| [0016](0016-batch-layout.md) | Lay out a batch as header, payloads, then offsets and count | Accepted |
| [0017](0017-batch-offsets-32-bit-with-opt-in-64.md) | Batch offsets are 32 bits wide, with 64 bits as an opt-in | Accepted |
| [0018](0018-compression-stays-with-the-engine.md) | Compression stays with the storage engine | Accepted |
