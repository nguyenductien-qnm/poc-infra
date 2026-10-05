# Verify

Identify the infrastructure diff, target Pulumi Go projects/modules, applicable decisions and expected impact. The primary runs checks; reviewers receive the scope, diff/base, relevant files, actual results and missing evidence. App-only work is outside this workflow.

## Checks by change

- Pack changes: from this skill's directory, run `python -B scripts/verify.py` and `python -B -m unittest discover -s scripts -p test_verify.py`. These verify harness mechanics, not infrastructure. Optional runtime installation checks are in the [setup guide](../README.md).
- Go infrastructure: find affected module roots and existing repository checks. Inspect tests for cloud/state side effects before running them. Use appropriate formatting checks, `go mod tidy -diff` when supported by the selected Go version, `go vet ./...`, `go test ./...` and `go build ./...` within those infrastructure modules. Run focused checks when the change warrants them; avoid application modules.
- IaC workload wiring: inspect ports, environment variables, secret references, health-check settings and supplied artifact references in Pulumi resources. A declared consumer contract is read-only input; do not run application tests/builds or container builds.
- Infrastructure GitOps: trace desired revision/configuration, target stack, deployment identity, reviewed inputs, authorization, concurrency, promotion and drift/recovery ownership. Distinguish implemented enforcement from recommendations; report absent controls and explicit exceptions against the target's requirements.
- Authorized preview: inspect the program and select the intended backend, stack, configuration and deployment identity before `pulumi preview --diff`. Preview can execute arbitrary Go and read APIs/state. Run it only in an authorized context; do not mutate config, dispatch pipelines, run up/destroy/import/refresh or bypass gates by default. Missing live checks stay unknown; mocks do not prove cloud acceptance or runtime behavior.

## Review

Composition/resource identity/ownership → `pulumi-component-reviewer`; state/deployment identities/GitOps → `pulumi-delivery-reviewer`. Provide the target context and local results; use both only for independent risks. When reviewers are unavailable, perform and disclose self-review. Validate findings against source or runtime evidence before editing.

Output: `pass | changes-requested | blocked`, scope/commit, actual checks (commands + results), findings with file:line/scenario, and unknowns. A pass applies only to the verified scope and is not production approval. A stale-check reminder does not mean checks ran. Do not create large reports/evidence in the repository.
