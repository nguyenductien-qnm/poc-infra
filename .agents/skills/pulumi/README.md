# Pulumi Go skill pack

Portable pack: a router ([SKILL.md](SKILL.md)), two workflows, two references, two read-only reviewer contracts, a static verifier with tests, and Claude/Codex adapter templates. It does not depend on the host repository.

## Install

1. Copy this folder to `.agents/skills/pulumi` in the target Git repository.
2. Copy [adapters/claude](adapters/claude) into `.claude/` and/or [adapters/codex](adapters/codex) into `.codex/`. Rename the installed `.claude/skills/pulumi/SKILL.md.in` to `SKILL.md`; leave the source template named `.in` so skill discovery sees only the canonical router. Merge hook entries into existing settings; do not overwrite other hooks or a conflicting agent/skill.
3. Check the installation: `python -B scripts/verify.py --repo-root <repo> --runtime claude` (or `codex`).
4. Start a new session and trust the hooks through the runtime's normal controls.

Requires Python 3.11+. Claude started below the repository root may need `--settings` pointing at the root `.claude/settings.json`.

Invoke `/pulumi adopt|verify` in Claude or `$pulumi adopt|verify` in Codex, and the reviewers by name. Reviewers inherit the model and get read-only tools.

## Standalone checks

```sh
python -B scripts/verify.py
python -B -m unittest discover -s scripts -p test_verify.py
```

Static only: no Go, no cloud, no Pulumi project needed. The tests need Git and also validate the review corpus structure, unique case IDs and nonempty inputs/criteria; they do not grade agent answers.

## Behavioral checks

For changes to Go guidance or review behavior, use [Go review cases](evals/go-review.json). Each case contains a request, source excerpts and evaluator-only `expect`/`reject` criteria. Give a fresh agent the skill plus only the request and files; withhold the criteria and prior conclusions. Excerpts intentionally omit imports and surrounding code and are review inputs, not build fixtures.

Run in a temporary workspace with no credentials or backend and read-only scope. Compare the actual response and tool trace to every criterion, checking both missed defects and false positives. A forbidden action is a failure even if the response is otherwise correct; an unavailable run is not a pass. Keep transient transcripts outside the repository and report the model, cases exercised, results and limitations in the task response. Correct demonstrated failures, then rerun affected cases with a fresh agent. These checks exercise agent reasoning, not native hook loading or cloud acceptance; the offline suite remains separate.

For a native runtime smoke check, use disposable IaC/app files and permit only their comment edits by the primary agent; keep the named reviewer read-only. Observe skill loading, reviewer invocation, SessionStart output, an IaC stale-check reminder and silence for an app-only edit. Inspect runtime events/logs rather than relying solely on the agent's final account.

Verified on Windows on 2026-10-06 with Claude Code 2.1.284 (`--print`) and Codex CLI 0.160.0 (`exec`, persisted session). These runs do not cover interactive UI or initial trust setup. In the tested Codex version, reviewer startup under `exec --ephemeral` failed with `no rollout found`; use a persisted session when testing delegation. Do not bypass hook trust to make a smoke check pass.

## Hooks

- SessionStart checks pack integrity.
- PostToolUse (Edit/Write, plus Codex `apply_patch`): edits to Pulumi projects, Pulumi Go modules or CI config that references Pulumi add a short "checks are stale" reminder. Edits to the pack or its adapters also rerun the pack checks. Application modules stay silent.

Hooks are advisory: 5 s timeout, 2 s budget, 64 KiB input. A failed, missing or untrusted hook is not a pass, and a reminder does not mean checks ran. Shell/MCP writes and external editors are not seen. For removed Pulumi references in CI files, hooks use the supplied pre-edit text or a bounded `git show HEAD`.

Runtime docs: [Claude hooks](https://code.claude.com/docs/en/hooks), [Claude subagents](https://code.claude.com/docs/en/sub-agents), [Codex skills](https://learn.chatgpt.com/docs/build-skills), [Codex subagents](https://learn.chatgpt.com/docs/agent-configuration/subagents), [Codex hooks](https://learn.chatgpt.com/docs/hooks).
