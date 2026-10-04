# Current contract

Baseline: `main` after PR #35. For a new task, read the diff and the latest owner comments on the relevant issues; this table does not replace the owner's decision authority.

| Decision by `nguyenductien-qnm` | Applicable contract | Source |
| --- | --- | --- |
| Two projects share a module; existing network/ECR/certificate/secret APIs are excessive for the PoC | `infra/workload` composes `infra/internal/{network,platform,data,app}`; bootstrap has its own `bootstrap/internal` | [#15](https://github.com/nguyenductien-qnm/poc-infra/issues/15#issuecomment-5970465379) |
| Pulumi manages workload state/KMS; a script creates the bootstrap state bucket; no key per environment yet | Keep `scripts/setup-bootstrap-backend.sh` as the accepted exception. Do not move OIDC/CI roles back to shell scripts | [#16](https://github.com/nguyenductien-qnm/poc-infra/issues/16#issuecomment-5970478096) |
| Pulumi manages OIDC, six roles and the preview policy | Edit `infra/bootstrap/internal/ciiam`; an admin operates bootstrap using the existing guide | [#17](https://github.com/nguyenductien-qnm/poc-infra/issues/17#issuecomment-5970483982) |
| The PoC only requires imageTag and ECR name from config; runtime hardening is deferred to production | Preserve the current ECS public subnet/IP, ALB HTTP and DB/Bedrock/rollback behavior | [#18](https://github.com/nguyenductien-qnm/poc-infra/issues/18#issuecomment-5970563054) |
| Keep PR preview with read-only OIDC and Actions pinned by SHA at their current major versions | Keep `dev → dev`, `staging → staging`, `main → prod` and seven-character image SHA tags; digest/build-once, least privilege, CLI pinning and protections are separate production work | [#19](https://github.com/nguyenductien-qnm/poc-infra/issues/19#issuecomment-5970634952) |
| Build the AI tooling first; the owner writes the docs afterward | Pack #22; proposal #14 and adoption/runbook/pilot #20 remain open. #21 is closed; do not add comprehensive mocks/evidence | [#14](https://github.com/nguyenductien-qnm/poc-infra/issues/14#issuecomment-5970694983), [#20](https://github.com/nguyenductien-qnm/poc-infra/issues/20#issuecomment-5970703112), [#22](https://github.com/nguyenductien-qnm/poc-infra/issues/22#issuecomment-5970653092) |

## Limits to report accurately

- Project separation is a code/state boundary, **not yet an IAM boundary**: deploy roles still have `AdministratorAccess`. Preview has `ReadOnlyAccess` and state/KMS permissions, so PR code can still pose a data-access risk. Do not call it credential-free or safe merely because it is named read-only; do not read secrets to test it.
- `infra/workload/Pulumi.prod.yaml` currently sets `protectStateful`/`deletionProtection` to false, although the README table says true. Evaluate the actual config/code and report the discrepancy; do not enable protection as part of pack work.
- Automatic triggers are temporarily disabled in both infrastructure workflows. `pr-check.yml` only has dispatch but uses `github.base_ref`/PR payload, so dispatch is not a replacement for PR preview. Branch mapping covers only `dev`, `staging` and `main`; do not dispatch deployment from a feature branch to test the pack.
- The network root creates new VPCs/subnets. There is no existing-network composition API yet; supplied IDs are proposed inputs, not grounds for presenting an invented snippet as working implementation. ECR config changes the workload URL, while the deploy workflow pushes to `poc-app`; a rename requires checking both sides in an authorized task.
- The CLI is not pinned; preview/build does not prove AWS configuration acceptance, actual IAM denials, DB connectivity, rollback or production readiness.

Keep production proposals separate and identify decisions needed from the owner: private workload/egress, TLS, DB encryption/backup/Multi-AZ/protection, runtime IAM, state isolation and delivery using the same digest. Explain tradeoffs without implementing them in advance.
