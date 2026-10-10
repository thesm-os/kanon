# Copyright ThesmOS B.V. 2026
# SPDX-License-Identifier: Apache-2.0

# The targets of kanon, which ergon init appends to the Makefile.

##@ Repository

.PHONY: test-386 test-e2e check-numbers

# BASE is the git revision whose recorded field numbers check-numbers compares with the working
# tree: the branch that a change merges into.
BASE ?= origin/main

test-386: ## Run the tests as 386 binaries, where int, uint and uintptr are 32 bits wide
	GOARCH=386 $(GO) test ./...
test-e2e: ## Run the end-to-end tests behind the build tag e2e
	$(GO) test -tags=e2e ./...
check-numbers: ## Fail when a change renumbers a field that BASE, origin/main by default, records
	KANON_CHECK=$(BASE) $(GO) generate -run 'go tool kanon' ./...
