# Bootstrap, state and delivery

Read `infra/bootstrap/main.go`, `bootstrap/internal/{statebackend,registry,ciiam}` and the actual workflows. The [current contract](current-contract.md) distinguishes the PoC from the production backlog.

- Bootstrap manages workload state S3, secrets KMS, ECR and CI OIDC/IAM; admins follow [BOOTSTRAP.md](../../../../docs/BOOTSTRAP.md). The script-created bootstrap S3 backend is an accepted exception. One KMS key is shared; object SSE-S3 is distinct from KMS secret encryption.
- Import mode is for existing resources; review disabling `importExisting` after adoption. Do not run import/up to test the pack or write checkpoints, secrets or decrypted output to logs/artifacts.
- State bucket/key/ECR protection and retention follow the code. Project separation does not constrain workload CI with Admin permissions. Trace OIDC `aud`/`sub`, repo identity and branch/environment from config to roles; do not introduce wildcard trust or move roles back to scripts.
- Preview receives OIDC for PRs and branches, state permissions needed by the CLI, and KMS decrypt. This remains a PoC risk; do not infer IAM enforcement from fork rules. Check secret schemas through metadata/code or nonsensitive fixtures; do not call GetSecretValue merely to inspect keys.
- Deployment currently follows new-tag preview → environment approval (when configured) → build/push short SHA tag → up. The image is not built at preview; digest/artifact binding and state freshness are not enforced. Do not claim that the same SHA proves artifact approval or that protections are enabled.
- Remote Actions use full commit SHA pins at their current major versions. Do not change workflow triggers, permissions or GitHub settings to test the pack.

Production delivery is separate work: build once before preview, use the same digest/config/SHA for apply, reject stale artifacts/state, and establish least privilege/boundaries and actual protections. Do not make these mandatory PoC pack gates.
