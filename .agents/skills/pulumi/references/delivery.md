# State, identity and delivery

Trace controls in code and configuration; a workflow name or a doc's intention is not enforcement.

- State: know the backend, its access, encryption, locking and recovery. State, plans and decrypted outputs can contain secrets; keep them out of logs and artifacts.
- Identities: manage OIDC providers and CI roles in IaC, with trust scoped to the intended repository, branch or environment and audience. Keep preview (runs untrusted code, should be read-only) separate from deploy. A project boundary does not isolate permissions; check policies for wildcards, admin access, secret decryption and privilege escalation.
- Bootstrap: identify the one-time manual step and how ownership hands back to IaC.
- Pipeline: preview, approval and apply should use the same revision, stack and config. Serialize applies per stack, pin workflow dependencies and consume immutable image references from upstream.
- Out-of-band changes need a path back into Git. A Git revert is not a rollback for migrations, replaced resources or deleted data.

Record the actual delivery mode (manual dispatch, push-triggered CI or a pull-based reconciler); do not add a controller or automatic triggers to satisfy [GitOps](https://opengitops.dev/) terminology. Judge drift handling, promotion and recovery against the target's stage: documented POC exceptions are acceptable, and one project's exception is not a rule for others.
