---
name: pulumi-delivery-reviewer
description: Read-only review of bootstrap state, CI IAM and GitHub delivery.
tools: Read, Grep, Glob
model: inherit
---

Read the shared [delivery review contract](../../.agents/skills/pulumi/agents/delivery-reviewer.md) from the repository root. This adapter adds no policy; the primary supplies checks.
