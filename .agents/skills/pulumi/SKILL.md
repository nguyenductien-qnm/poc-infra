---
name: pulumi
description: Adopt, implement or verify Pulumi Go infrastructure and its GitOps delivery. Use for resource composition, ownership, state, secrets, identity, workload configuration or infrastructure pipelines; application implementation and builds are outside this skill.
---

# Pulumi Go infrastructure

Inspect the target project's Pulumi configuration, Go modules and applicable repository guidance before editing. Determine the requested outcome, affected stacks, resource owners and delivery model. Existing project decisions are task context; this skill requires no particular repository, topology, backend, cloud or application.

For adoption or implementation, read [adopt.md](workflows/adopt.md). For validation or review, read [verify.md](workflows/verify.md). Load the smallest relevant set:

| Change | Reference | Review when risk warrants it |
| --- | --- | --- |
| Go composition, ownership, resource identity or migration | [components.md](references/components.md) | `pulumi-component-reviewer` |
| State, secrets, deployment identities or GitOps delivery | [delivery.md](references/delivery.md) | `pulumi-delivery-reviewer` |
| IaC-managed workload configuration | [runtime.md](references/runtime.md) | Component reviewer when contracts cross resources |
| Uncertain SDK, provider or API behavior | [sdk-verification.md](references/sdk-verification.md) | Focused local or official-source verification |

State the route briefly. Use one matched reviewer for a consequential change; use both for independent composition and delivery risks. Small documentation changes can use self-review. Disclose self-review when subagents are unavailable.

Keep configuration at composition roots, concrete Args/Outputs and one owner per resource. Preserve resource identity or explain a reviewed migration. Do not add a registry, generic module engine or services without a consumer.

Application contracts and supplied artifact references are read-only inputs for IaC wiring. Application implementation, build/tests and container builds are outside this skill. Cloud/state writes and deployment require task authorization; the skill does not grant it.

The [pack setup](README.md) describes optional Codex/Claude adapters, advisory hooks and standalone checks. Hooks validate the pack and remind about stale evidence; they do not run Go checks, enforce IAM, approve deployment or prove live health.
