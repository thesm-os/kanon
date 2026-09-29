# Contributing to kanon

kanon generates binary codecs for Go struct types. Every change keeps these rules:

1. **The wire format is a specification.** [The kanon wire format](docs/rfc/0001-wire-format.md)
   defines every byte that an encoder writes and a decoder accepts. A change to it needs an
   RFC, and data that the current code encodes must still decode.
2. **The generated code imports the standard library and the kanon runtime alone.** The
   runtime is the packages `go.thesmos.sh/kanon` and `go.thesmos.sh/kanon/wire`. The production
   code of the module imports the standard library and the module itself, and `depguard`
   rejects any other import. The tests and `kanontest` may also import `go.dokimi.dev/assert`.
3. **Warm paths allocate nothing.** An encode into a buffer with room for the encoding, and a
   decode into a receiver that decoded before, allocate nothing. The allocation checks of
   `kanontest` fail on every allocation.
4. **The code keeps 100% statement coverage and a 100% mutation score**, per layer as
   `.ergon.yaml` sets them. `make check` enforces the coverage, and `make check-mutation` runs
   gremlins.
5. **Apache 2.0 by submission.** A pull request asserts that you may license its contribution
   under the Apache License 2.0.

## Development setup

The Makefile runs [ergon](https://github.com/thesm-os/ergon), which you install first:

```bash
# Homebrew (macOS, Linux)
brew install thesm-os/tap/ergon

# Go toolchain
go install go.thesmos.sh/ergon/cmd/ergon@latest
```

Then set up the repository:

```bash
git clone git@github.com:thesm-os/kanon.git && cd kanon

# Go 1.27.0 or later, as go.mod requires
go version

# gofumpt, gci, golangci-lint, govulncheck, go-license, benchstat, gremlins, and
# markdownlint-cli2 through npm when it is missing
make bootstrap

# make check before each commit, and ergon check commit-msg on each message
pre-commit install --hook-type pre-commit --hook-type commit-msg

make check
```

## Making a change

For a fix, documentation or a small improvement:

1. Branch from `main`.
2. Make the change.
3. After a change to the generator or to a fixture, run `make generate` to regenerate the
   fixtures. The tests of `internal/kanon` compare the output of the generator with the committed
   files, declaration by declaration.
4. When a change alters samples or probes on purpose, update the golden files with
   `go test ./kanontest/ ./internal/fixture/... -update`, and read their diff before you commit.
5. Run `make check` and `make check-generated`.
6. Open a pull request against `main`.

For a change to the wire format, to the generated API or to the options of the generator:

1. Open an issue first, so that the change is agreed before it is built.
2. When more than one design competes, write an RFC under [docs/rfc](docs/rfc/README.md).
3. Record each decision in an ADR under [docs/adr](docs/adr/README.md), from
   [its template](docs/templates/ADR.md).
4. Cite the RFC or the ADR in the pull request that implements it.

## Code standards

- Return an error where library code could panic. Only the kanon command writes output,
  through an `io.Writer`. `forbidigo` rejects `panic` and `fmt.Print*`.
- Give every declaration a doc comment that states its contract, exported or not.
- Start an error message with the name of its package.
- Write the generated files, which end in `.kanon.go` and `.kanon_test.go`, with
  `make generate` alone.

## Testing

```bash
make check            # the gate that CI runs: mod, lint, test and coverage
make test-race        # the tests under the race detector
make test-386         # the tests where int, uint and uintptr are 32 bits wide
make check-generated  # fail when go generate changes a committed generated file
make check-numbers    # fail when a change renumbers a field that main records
make check-mutation   # gremlins on every layer, with go test ./... for every mutant
```

## Commit messages

Write [Conventional Commits](https://www.conventionalcommits.org/):

```text
feat: index views in one scan, and size opaque values with SizeKanon
fix: return the index of a view without fields without a scan
docs: record the zero rule of opaque values and their views
```

The types are `feat`, `fix`, `docs`, `refactor`, `test`, `ci`, `chore`, `perf`, `build` and
`revert`. A subject has 72 bytes at most, and a line of the body 100 bytes. The `commit-msg`
hook and the CI check every commit with `ergon check commit-msg`.

Sign your commits, so that GitHub shows them as **Verified**. For SSH signing:

```bash
git config --global gpg.format ssh
git config --global user.signingkey ~/.ssh/id_ed25519.pub
git config --global commit.gpgsign true
```

GitHub's [guide to signing commits][sign] covers GPG and sigstore.

## Review

- A maintainer, as [CODEOWNERS](.github/CODEOWNERS) lists them, reviews every pull request.
- These jobs of the CI must pass:
  - Vet and build on Linux, macOS and Windows.
  - The gate, which runs mod, lint, test and coverage.
  - The race tests on Linux, macOS and Windows.
  - The tests on 386.
  - The check that the generated code is current.
  - The check of the field numbers, on a pull request.
  - The check of the commit messages.
- A pull request that changes the wire format or the generated API also needs its RFC or ADR.

[sign]: https://docs.github.com/en/authentication/managing-commit-signature-verification/about-commit-signature-verification
