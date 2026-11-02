---
"go.thesmos.sh/kanon": minor
---

Add `kanon.Strict`, whose method `StrictKanon` marks a type whose decode method accepts only the bytes that its encode method writes, so that a canonical decode does not encode its values again.
