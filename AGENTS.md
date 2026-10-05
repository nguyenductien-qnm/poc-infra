# Infrastructure work

## Task routing

Use the standalone [Pulumi Go skill](.agents/skills/pulumi/SKILL.md) for infrastructure and its GitOps delivery. Discover this project's actual roots, contracts and decisions; the skill supplies no repository-specific architecture. Application implementation and builds are outside the skill. Codex uses `$pulumi adopt …` / `$pulumi verify …`; Claude uses `/pulumi adopt …` / `/pulumi verify …`.

Follow the existing code style. Keep config at composition roots, concrete Args/Outputs, direct composition and one owner per resource; do not add a framework/registry. Preserve resource identity or provide a reviewed migration/alias. Do not put plaintext internal secrets/tokens/account IDs/ARNs in new code/docs/logs or fetch secret values to inspect schemas.

Follow the [verify workflow](.agents/skills/pulumi/workflows/verify.md); report actual results and missing checks. Respect applicable project decisions and authorization. Do not automatically perform cloud/state writes, change external settings or enable deployment/reconciliation.

Use scoped, lowercase English Conventional Commits. Write this skill pack's instructions, documentation and code comments in English. Preserve existing conventions elsewhere: unaccented Vietnamese code comments and accented Vietnamese guides. Update relevant guidance when behavior changes; use existing documentation rather than adding another report.
