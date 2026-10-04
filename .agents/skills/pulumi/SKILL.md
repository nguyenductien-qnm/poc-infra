---
name: pulumi
description: Adopt or verify Pulumi Go changes in this repository, including components, bootstrap ownership, AWS runtime wiring and GitHub delivery. Use for infrastructure or app-to-infrastructure contract changes; ordinary app-only edits need no infrastructure review.
---

# Pulumi Go pack

Read [current-contract.md](references/current-contract.md) first. Owner decisions in issue comments take precedence over the original Epic checklist. Inspect the relevant implementation before proposing a change; this pack does not authorize cloud operations.

Choose **adopt** to propose configuration/composition: read [adopt.md](workflows/adopt.md). Choose **verify** to review a diff and report checks: read [verify.md](workflows/verify.md). Load only the references needed:

| Change | Reference | Review when risk warrants it |
| --- | --- | --- |
| Args/Outputs, composition, naming, ownership or resource identity | [components.md](references/components.md) | `pulumi-component-reviewer` |
| Backend, secrets, OIDC/IAM or workflow | [delivery.md](references/delivery.md) | `pulumi-delivery-reviewer` |
| ECS task, app env/health, RDS or image contract | [runtime.md](references/runtime.md) | Component reviewer for cross-component changes |
| SDK field, version or AWS/API value not established by code | [sdk-verification.md](references/sdk-verification.md) | Usually local verification |

State the chosen mode, references and reviewer briefly. Default to one matched reviewer for a consequential change. Use a second only for an independent component/delivery risk or when requested; small documentation changes can use self-review. When subagents are unavailable or unauthorized, report self-review explicitly.

Keep Go composition direct, with concrete Args/Outputs and one owner per resource. Do not add a registry, generic module engine, mandatory component wrapper for every helper, or support for services without a consumer. Configuration stays at the composition roots. Do not turn deferred production requirements into PoC release gates or silently undo owner decisions.

Hooks and the manual checker detect local pack problems and stale checks. They do not establish IAM enforcement, approve deployment or prove a live workload healthy.
