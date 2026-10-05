# Verify

Scope the diff to infrastructure Go modules and Pulumi/CI configuration. The primary agent runs checks; reviewers only read.

## Checks

- Infrastructure Go modules (those requiring `github.com/pulumi/pulumi/sdk`): `gofmt -l .`, `go mod tidy -diff` when the Go version supports it, `go vet ./...`, `go build ./...`, `go test ./...`. Look for cloud side effects in tests before running them. Skip application modules.
- Pack edits: from the skill folder, `python -B scripts/verify.py` and `python -B -m unittest discover -s scripts -p test_verify.py`.
- `pulumi preview --diff` only when authorized, with the intended backend, stack and identity selected. Preview executes program code and reads state.

## Review

Use the router's reviewer only for consequential changes, both only for independent risks. Pass the scope, diff and check results. Otherwise self-review and say so. Validate findings against source before editing.

## Report

`pass | changes-requested | blocked`, checks run with results, findings (file:line, failure scenario, fix) and unknowns. A pass covers only the checked scope; a hook reminder is not a check.
