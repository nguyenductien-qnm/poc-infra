# infra-poc

Pulumi Go PoC: `infra/bootstrap` and `infra/workload` share a module; the Go service lives in `app/`. An admin runs bootstrap manually; CI deploys the workload. Newer owner decisions in issues take precedence over the Epic checklist; the pack separates the PoC from deferred production work.

## Task routing

Use the [Pulumi pack](.agents/skills/pulumi/SKILL.md) for adopting/reviewing infrastructure, bootstrap IAM/state, CI or app-to-infrastructure wiring. Load references by scope. App-only changes need no infrastructure reviewer. Codex uses `$pulumi adopt …` / `$pulumi verify …`; Claude uses `/pulumi adopt …` / `/pulumi verify …`.

Follow the existing code style. Keep config at composition roots, concrete Args/Outputs, direct composition and one owner per resource; do not add a framework/registry. Preserve resource identity or provide a reviewed migration/alias. Do not put plaintext internal secrets/tokens/account IDs/ARNs in new code/docs/logs or fetch secret values to inspect schemas.

Follow the [verify workflow](.agents/skills/pulumi/workflows/verify.md); report actual results and missing checks. Preview only in an authorized context. Do not automatically run up/destroy/import, change GitHub settings or enable workflows. Bootstrap operators have separate permissions under `docs/BOOTSTRAP.md`; do not act for the admin without task authorization.

Use scoped, lowercase English Conventional Commits. Write this skill pack's instructions, documentation and code comments in English. Preserve existing conventions elsewhere: unaccented Vietnamese code comments and accented Vietnamese guides. Update relevant guidance when behavior changes; use existing documentation rather than adding another report.
