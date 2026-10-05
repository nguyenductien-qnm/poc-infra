# Infrastructure work

Use the [Pulumi Go skill](.agents/skills/pulumi/SKILL.md) for infrastructure and its delivery: `/pulumi adopt …` / `/pulumi verify …` in Claude, `$pulumi adopt …` / `$pulumi verify …` in Codex. The skill is generic; discover this repository's roots, stacks and decisions from source and docs. Application implementation and builds are outside it.

Even without the skill: never put secrets, tokens, account IDs or ARNs in plaintext, and do not run cloud/state writes or dispatch deployments unless the task authorizes it.

## Conventions

- Follow the existing code style.
- Use scoped, lowercase English Conventional Commits.
- Write the skill pack's instructions, docs and code comments in English. Elsewhere keep unaccented Vietnamese code comments and accented Vietnamese guides.
- Update relevant docs when behavior changes; do not add separate report files.
