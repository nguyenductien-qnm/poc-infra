---
name: pulumi
description: Adopt, implement or verify Pulumi Go infrastructure and its GitOps delivery. Use for resource composition, ownership, state, secrets, identity, workload configuration or infrastructure pipelines; application implementation and builds are outside this skill.
---

# Pulumi Go infrastructure

Scope: Pulumi Go programs, their state, deployment identities and delivery pipelines. Application code, app tests and container builds are out of scope; treat app contracts and supplied image references as read-only inputs.

Start by reading the target's Pulumi projects, Go modules, affected packages and their consumers, and repository guidance. Documented project decisions override the generic advice here.

Use [adopt](workflows/adopt.md) to design or implement and [verify](workflows/verify.md) to check or review. Load only the reference you need:

| Change | Reference | Reviewer for consequential changes |
| --- | --- | --- |
| Composition, ownership, resource identity, workload wiring | [components](references/components.md) | `pulumi-component-reviewer` |
| State, secrets, deployment identity, CI/GitOps | [delivery](references/delivery.md) | `pulumi-delivery-reviewer` |

## Always

- Keep config at composition roots, pass concrete Args/Outputs and keep one owner per resource. Do not add registries, generic module engines or unused options.
- Preserve resource identity (type token, logical name, parent) or add an alias/migration and list expected replacements.
- Never put secrets, tokens, account IDs or ARNs in plaintext in code, docs or logs, and do not fetch secret values.
- Do not run `pulumi up/destroy/import/refresh`, change stack config, dispatch pipelines or change cloud settings unless the task explicitly authorizes it.
- Resolve uncertain SDK/provider APIs from the module's own versions first (`go.mod`, `go doc <import-path> <Symbol>`), then official docs for that version. Do not upgrade major versions.

## Match rigor to the target

Production stacks warrant hardening: deletion protection, least-privilege CI roles, automatic triggers with approval gates, drift detection, promotion rules. For POC or dev targets, intentional simplifications such as manual-dispatch deploys, disabled protection or broad bootstrap roles are valid decisions. Mention the risk once when relevant; do not block on them or change them unasked.
