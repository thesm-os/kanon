---
name: Bug report
about: Report a defect of the generator, the generated code or the runtime
title: "bug: "
labels: ["bug"]
assignees: []
---

## Summary

<!-- One sentence: what goes wrong. -->

## Affected part

<!-- The generator (the kanon command), the generated code, the runtime
     (kanon, wire), frame, batch, kanontest, or build and CI. -->

## Reproduction

<!-- The smallest struct type and directive that show the defect, and the
     call or the input that fails. When the generated test of a type fails,
     paste its output. -->

```go
//go:generate go tool kanon -type=T

type T struct {
}
```

## Expected behaviour

<!-- What should happen. -->

## Actual behaviour

<!-- What happens: the exact error, panic or output. -->

## Environment

- kanon version (`go tool kanon -version`, or the commit):
- Go version (`go version`):
- OS and architecture:

## Additional context

<!-- Related issues, recent changes, or anything else that matters. -->
