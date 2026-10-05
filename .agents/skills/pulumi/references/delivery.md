# State, identity and GitOps delivery

Identify the target projects/stacks, backend, deployment identity and actual pipeline or reconciliation agent. Trace controls in code/configuration; a workflow name or documented intention is not evidence of enforcement.

## State and deployment identity

- Establish state ownership, access, encryption, concurrency/locking, versioning and a recovery procedure appropriate to the target. State, saved plans and decrypted outputs can contain secrets; do not expose them in logs or shared artifacts.
- Separate infrastructure ownership from deployment permissions. A project boundary alone does not stop an identity from changing another project's resources or increasing its own permissions.
- Manage OIDC providers and CI roles/policies through IaC. Scope workload identity trust to the intended source, audience, environment and execution context. Distinguish trusted deployment from untrusted change evaluation; preview executes Go code and may read state or cloud APIs.
- Grant permissions for the actual backend and deployment scope. Review wildcard trust, administrator access, secret decryption and privilege escalation as explicit risks; inspect policies rather than reading secret values or probing credentials by default.
- For initial state/identity setup, identify the authorized operator and any unavoidable temporary bootstrap step. Document the handoff and cleanup instead of treating manual setup as the permanent ownership model.

## GitOps operating contract

- Identify the authoritative Git revision, stack configuration, provider/tool versions and external inputs. Changes made outside Git need an owner and a path back into the desired state; do not silently make emergency edits the new source of truth.
- Distinguish Git-triggered CI from a system that pulls desired state and continuously reconciles it. Record the actual delivery mode; do not assume a controller or add one merely to satisfy terminology. See the [OpenGitOps principles](https://opengitops.dev/).
- Bind preview/approval/apply to the same reviewed revision, target stack and effective configuration. Consume immutable artifact references supplied by upstream systems when resources need them; application builds are outside this skill. Reject or re-evaluate stale evidence when relevant code, config, inputs, provider versions or target state changes.
- Trace authorization gates and serialize writes per target stack. Inspect queued/rerun/cancel behavior so a newer change cannot be overwritten by an older execution. Review remote workflow dependency pins and runner permissions in the selected delivery system.
- Define promotion between environments without silently changing reviewed inputs. Identify who owns drift detection, its schedule/event source, the remediation decision and whether correction is manual or automatic.
- Define failure/abort signals, recovery ownership and rollback or forward-fix steps. Reverting Git is not sufficient evidence of a safe rollback for migrations, replaced resources or deleted data; verify state recovery and data protection separately.

Assess these controls against the requested environment and risks. Report explicit target exceptions with their consequences and missing evidence. Do not turn one project's temporary exception into a rule for other production systems or enable pipelines/reconciliation automatically.
