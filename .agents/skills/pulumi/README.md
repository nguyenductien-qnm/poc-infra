# Pulumi Go skill setup

This folder is the complete portable pack: instructions, two read-only reviewer contracts, verifier/tests and Codex/Claude adapter templates. It needs no host repository documentation or particular infrastructure layout. Application builds/tests are outside its scope.

## Standalone checks

Requires Python 3.11+ available as `python`. From this folder, run:

```sh
python -B scripts/verify.py
python -B -m unittest discover -s scripts -p test_verify.py
```

The static verifier needs no Git, cloud credentials, Pulumi project or installed runtime adapters. Mechanism tests require Git and create isolated temporary repositories for optional integration. Checks do not run Go, contact cloud services or establish production readiness.

## Optional project integration

Copy this complete folder to `.agents/skills/pulumi` in the target Git repository. This is the installation location expected by the bundled native adapters; the target's Pulumi projects and Go modules may be anywhere.

Copy the chosen runtime's agent/skill adapter files from [Codex templates](adapters/codex) or [Claude templates](adapters/claude) into the corresponding `.codex` or `.claude` paths. Merge the template's hook entries into existing configuration, preserving other hooks/settings. Do not overwrite a conflicting agent or skill silently. Installing one runtime does not require the other.

From this folder, check the selected installation explicitly:

```sh
python -B scripts/verify.py --repo-root "/path/to/target" --runtime codex
python -B scripts/verify.py --repo-root "/path/to/target" --runtime claude
```

Start a new session after configuration changes. Review/trust project configuration and exact hook definitions through the runtime's normal controls; never persist trust or bypass approvals automatically. For Claude launched below the target root, pass `--settings` pointing to its root `.claude/settings.json` if that runtime does not load it automatically. All commands use `python`, including CMD/PowerShell launchers. No global configuration, symlinks or installer are required.

Use `$pulumi adopt` / `$pulumi verify` in Codex or `/pulumi adopt` / `/pulumi verify` in Claude. Invoke `pulumi-component-reviewer` or `pulumi-delivery-reviewer` by name when their risk is relevant. Runtime adapters inherit model choices; the primary supplies checks, and reviewers inspect source only. A read-only filesystem sandbox does not make other exposed tools read-only; follow reviewer tool restrictions.

## Hooks and limits

SessionStart checks pack integrity. PostToolUse handles Edit/Write and Codex apply_patch, then uses target Pulumi configuration and Go module dependencies to identify infrastructure changes. It also covers pack and Pulumi delivery configuration edits. Application modules are outside the filter. Mixed modules, external editors, shell/MCP writes and unsupported payloads may require manual verification; a reminder never proves that Go, preview or review ran.

For delivery files whose Pulumi references were removed, hooks use supplied pre-edit text or bounded read-only Git history. Deleted untracked files without a supplied preimage, large/unavailable historical content and ambiguous mixed modules require manual verification.

Hooks are advisory, with a 5-second timeout, 2-second verification budget and 64 KiB input limit. Failures, missing tools, disabled/untrusted hooks and unsupported payloads are not passes. Diagnostics do not echo payloads. Native runtime support must be verified separately for the actual CLI/version/OS; static/configured-command tests do not prove native feedback. Changed definitions require review/trust again.
