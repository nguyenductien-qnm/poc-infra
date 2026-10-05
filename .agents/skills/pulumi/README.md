# Pulumi Go skill pack

Portable pack: a router ([SKILL.md](SKILL.md)), two workflows, two references, two read-only reviewer contracts, a static verifier with tests, and Claude/Codex adapter templates. It does not depend on the host repository.

## Install

1. Copy this folder to `.agents/skills/pulumi` in the target Git repository.
2. Copy [adapters/claude](adapters/claude) into `.claude/` and/or [adapters/codex](adapters/codex) into `.codex/`. Merge hook entries into existing settings; do not overwrite other hooks or a conflicting agent/skill.
3. Check the installation: `python -B scripts/verify.py --repo-root <repo> --runtime claude` (or `codex`).
4. Start a new session and trust the hooks through the runtime's normal controls.

Requires Python 3.11+. Claude started below the repository root may need `--settings` pointing at the root `.claude/settings.json`.

Invoke `/pulumi adopt|verify` in Claude or `$pulumi adopt|verify` in Codex, and the reviewers by name. Reviewers inherit the model and get read-only tools.

## Standalone checks

```sh
python -B scripts/verify.py
python -B -m unittest discover -s scripts -p test_verify.py
```

Static only: no Go, no cloud, no Pulumi project needed. The tests need Git.

## Hooks

- SessionStart checks pack integrity.
- PostToolUse (Edit/Write, plus Codex `apply_patch`): edits to Pulumi projects, Pulumi Go modules or CI config that references Pulumi add a short "checks are stale" reminder. Edits to the pack or its adapters also rerun the pack checks. Application modules stay silent.

Hooks are advisory: 5 s timeout, 2 s budget, 64 KiB input. A failed, missing or untrusted hook is not a pass, and a reminder does not mean checks ran. Shell/MCP writes and external editors are not seen. For removed Pulumi references in CI files, hooks use the supplied pre-edit text or a bounded `git show HEAD`.

Runtime docs: [Claude hooks](https://code.claude.com/docs/en/hooks), [Claude subagents](https://code.claude.com/docs/en/sub-agents), [Codex skills](https://learn.chatgpt.com/docs/build-skills), [Codex subagents](https://learn.chatgpt.com/docs/agent-configuration/subagents), [Codex hooks](https://learn.chatgpt.com/docs/hooks).
