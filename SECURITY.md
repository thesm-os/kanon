# Security policy

## Supported versions

kanon has no release yet. Security fixes go to the `main` branch.

| Version | Supported |
|---------|-----------|
| main    | Yes       |

## Reporting a vulnerability

Do not open a public issue. Report a vulnerability by email to **security@thesmos.sh**, with:

1. A description of the vulnerability and its impact.
2. The steps to reproduce it, or a minimal proof of concept, such as the struct type, the
   directive and the input that a generated decoder mishandles.
3. The part of kanon that it affects, such as the generator (`cmd/kanon`, `internal/kanon`),
   the code that the generator writes, the runtime (`kanon`, `wire`), `frame`, `batch` or
   `kanontest`.
4. The severity that you assess, from critical, high, medium and low.

Encrypt a sensitive report with the PGP key at <https://thesmos.sh/.well-known/security.txt>.

## Response targets

| Step | Target |
|------|--------|
| Acknowledgement of the report | 2 business days |
| Triage and severity assessment | 5 business days |
| Fix | The fix target of the severity |
| Disclosure | A date agreed with the reporter |
| Public advisory | The day the fix is released |

| Severity | Fix target | Disclosure window |
|----------|------------|-------------------|
| Critical | 7 calendar days | 14 days after the fix |
| High | 14 calendar days | 30 days after the fix |
| Medium | 30 calendar days | 60 days after the fix |
| Low | The next minor release | With the release notes |

## Scope

In scope:

- Every Go package of the module.
- The code that the generator writes. A decoder, a view or a frame reader that panics, reads
  past its input or allocates without bound on untrusted input is a vulnerability.

Out of scope: typos in the documentation, the CI configuration and the developer tooling.

## Safe harbour

We will not pursue legal action against a researcher who keeps these rules:

- Act in good faith and follow this policy.
- Leave the data of others unread and unchanged.
- Keep every service available.
- Give enough detail for us to reproduce and fix the issue.
