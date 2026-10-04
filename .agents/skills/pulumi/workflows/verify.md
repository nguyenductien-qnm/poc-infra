# Verify

Read the diff, current contract and relevant implementation. The primary runs checks; the reviewer receives the scope, diff/base, relevant files, check results and missing evidence.

## Checks by change

- Pack/adapters/hooks: at the repository root, run `python -B scripts/verify-pulumi-pack.py`, then `python -B -m unittest discover -s scripts -p test_pulumi_pack.py`. No AWS calls or Go build.
- Go infrastructure: in `infra/`, run `go mod tidy -diff`, `go vet ./...`, `go test ./...` and `go build ./...`.
- App or app-to-infrastructure changes: in `app/`, run `go mod tidy -diff`, `go vet ./...`, `go test ./...` and `go build -o <temporary-path-outside-repo> .`; check environment variables/ports/secrets/images in the task definition. Build the Docker image when the container changes and a runtime is available.
- Authorized workload preview: the operator selects backend/stack/imageTag first; the primary runs `pulumi preview --diff` when the task permits it. Do not read live state, change config or dispatch workflows by default to verify. Do not run up/destroy/import/refresh or bypass gates. Report missing live checks as unknown; do not replace them with mocks and claim a pass.

## Review

Args/Outputs/identity/ownership → `pulumi-component-reviewer`; backend/IAM/workflow → `pulumi-delivery-reviewer`. Invoke the agent by its runtime name and provide local results; default to one reviewer. Use both only for two independent risks. If subagents are unavailable, perform and disclose self-review; the primary validates findings before editing.

Output: `pass | changes-requested | blocked`, scope/commit, actual checks (commands + results), findings with file:line/scenario, and unknowns. A pass applies only to the verified scope and is not production approval. A stale-check reminder does not mean checks ran. Do not create large reports/evidence in the repository.
