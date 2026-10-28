---
"go.thesmos.sh/kanon": patch
---

Fix the canonical decode of a validated field to return `kanon.ErrNotCanonical` for an explicit zero before it calls `ValidateKanon`.
