"""Read-only, AWS-free pack verification; Python 3.11+, standard library only."""

import argparse
import json
from pathlib import Path
import re
import sys
import time
import tomllib

PACK = Path(".agents/skills/pulumi")
ROLES = ("component", "delivery")
REFERENCES = ("current-contract", "components", "delivery", "runtime", "sdk-verification")
MAX_BYTES = 65536
LINK = re.compile(r"\[[^\]]+\]\(([^\s)]+)\)")
CODEX_COMMAND = 'python3 "$(git rev-parse --show-toplevel)/scripts/verify-pulumi-pack.py" --hook codex'
CODEX_WINDOWS = 'for /f "delims=" %r in (\'git rev-parse --show-toplevel\') do @python "%r/scripts/verify-pulumi-pack.py" --hook codex'


def read(path):
    if path.stat().st_size > MAX_BYTES:
        raise ValueError("file exceeds 64 KiB check limit")
    return path.read_text(encoding="utf-8-sig")


def frontmatter(text):
    match = re.match(r"\A---\r?\n(.*?)\r?\n---(?:\r?\n|$)", text, re.S)
    if not match:
        raise ValueError("missing frontmatter")
    fields = {}
    for line in match[1].splitlines():
        key, separator, value = line.partition(":")
        if not separator or key in fields:
            raise ValueError("expected unique scalar frontmatter fields")
        fields[key] = value.strip()
    if not fields.get("name") or not fields.get("description"):
        raise ValueError("name and description are required")
    return fields


def verify(root, budget=2.0):
    """Return bounded diagnostics; no subprocess, network or filesystem writes."""
    started = time.monotonic()
    findings = []

    def check_time():
        if time.monotonic() - started > budget:
            raise TimeoutError("pack check timed out; verification incomplete")

    def note(path, message):
        if len(findings) < 8:
            findings.append(f"{path}: {message}"[:500])

    def checked(path, callback):
        check_time()
        try:
            callback(read(root / path))
        except (OSError, ValueError, KeyError, TypeError) as error:
            note(path, str(error))

    required = [PACK / "SKILL.md", Path("AGENTS.md"), Path("CLAUDE.md")]
    required += [PACK / "references" / f"{name}.md" for name in REFERENCES]
    required += [PACK / "workflows" / f"{name}.md" for name in ("adopt", "verify")]
    required += [PACK / "agents" / f"{role}-reviewer.md" for role in ROLES]
    required += [Path(".claude/skills/pulumi/SKILL.md")]
    required += [Path(f".claude/agents/pulumi-{role}-reviewer.md") for role in ROLES]

    def links(path, text):
        for target in LINK.findall(text):
            if re.match(r"(?:https?://|#)", target):
                continue
            resolved = (root / path).parent.joinpath(target.split("#", 1)[0]).resolve()
            if not resolved.is_relative_to(root) or not resolved.exists():
                note(path, f"broken or external local link: {target}")

    for path in required:
        checked(path, lambda text, path=path: links(path, text))

    def skill(text):
        fields = frontmatter(text)
        if fields["name"] != "pulumi":
            raise ValueError("skill must be named pulumi")
        return fields

    checked(PACK / "SKILL.md", skill)

    def claude_skill(text):
        if skill(text) != skill(read(root / PACK / "SKILL.md")):
            raise ValueError("Claude router metadata drift")
        if "../../../.agents/skills/pulumi/SKILL.md" not in text:
            raise ValueError("Claude router must point at shared source")

    checked(Path(".claude/skills/pulumi/SKILL.md"), claude_skill)
    for role in ROLES:
        name = f"pulumi-{role}-reviewer"
        shared = f"{PACK.as_posix()}/agents/{role}-reviewer.md"

        def codex_agent(text, name=name, shared=shared):
            fields = tomllib.loads(text)
            if fields.get("name") != name or not fields.get("description"):
                raise ValueError("invalid Codex agent identity")
            if fields.get("sandbox_mode") != "read-only":
                raise ValueError("Codex reviewer must use read-only sandbox")
            if fields.get("web_search") != "disabled":
                raise ValueError("Codex reviewer must disable web search")
            if shared not in fields.get("developer_instructions", ""):
                raise ValueError("Codex reviewer must point at shared source")

        def claude_agent(text, name=name, shared=shared):
            fields = frontmatter(text)
            if fields["name"] != name or fields.get("model") != "inherit":
                raise ValueError("invalid Claude agent identity/model")
            if set(fields.get("tools", "").replace(" ", "").split(",")) != {"Read", "Grep", "Glob"}:
                raise ValueError("Claude reviewer tools must be Read, Grep, Glob only")
            if f"../../{shared}" not in text:
                raise ValueError("Claude reviewer must point at shared source")
            codex = tomllib.loads(read(root / f".codex/agents/{name}.toml"))
            if fields["description"] != codex["description"]:
                raise ValueError("reviewer description drift between runtimes")

        checked(Path(f".codex/agents/{name}.toml"), codex_agent)
        checked(Path(f".claude/agents/{name}.md"), claude_agent)

    def hooks(text, runtime):
        events = json.loads(text)["hooks"]
        if set(events) != {"SessionStart", "PostToolUse"}:
            raise ValueError("expected only SessionStart and PostToolUse")
        for event, groups in events.items():
            if len(groups) != 1 or len(groups[0]["hooks"]) != 1:
                raise ValueError("expected one lightweight handler per event")
            re.compile(groups[0]["matcher"])
            expected = "startup|resume|clear|compact" if event == "SessionStart" else (
                "^(Edit|Write)$" if runtime == "claude" else "^(apply_patch|Edit|Write)$"
            )
            if groups[0]["matcher"] != expected:
                raise ValueError("hook matcher drift")
            handler = groups[0]["hooks"][0]
            if handler.get("type") != "command" or handler.get("timeout") != 5:
                raise ValueError("expected command handler with five-second timeout")
            if runtime == "claude":
                if handler.get("command") != "python" or handler.get("args") != [
                    "${CLAUDE_PROJECT_DIR}/scripts/verify-pulumi-pack.py", "--hook", "claude"
                ]:
                    raise ValueError("Claude hook command/args drift")
            else:
                if handler.get("command") != CODEX_COMMAND or handler.get("commandWindows") != CODEX_WINDOWS:
                    raise ValueError("Codex commands drift from root-resolving shared checker")

    for runtime, path in (("codex", ".codex/hooks.json"), ("claude", ".claude/settings.json")):
        checked(Path(path), lambda text, runtime=runtime: hooks(text, runtime))
    for path in (root / ".github/workflows").glob("*.y*ml"):
        check_time()
        for line, text in enumerate(read(path).splitlines(), 1):
            match = re.match(r"\s*(?:-\s*)?uses:\s*([^\s#]+)", text)
            if match and not match[1].startswith(("./", "docker://")) and not re.fullmatch(r"[^@]+@[0-9a-fA-F]{40}", match[1]):
                note(f"{path.relative_to(root)}:{line}", "remote action must pin a full commit SHA")
    for old in ("pulumi-go-aws", "gitops-github-actions", "go-ecs-service", "verify-before-code"):
        if (root / ".claude/skills" / old / "SKILL.md").exists():
            note(f".claude/skills/{old}/SKILL.md", "retired entry point conflicts with shared pack")
    if not (root / "scripts/verify-pulumi-pack.py").is_file():
        note("scripts/verify-pulumi-pack.py", "hook executable is missing")
    check_time()
    return findings


def changed_paths(payload, runtime):
    tool = payload.get("tool_name")
    inputs = payload.get("tool_input")
    if tool in ("Edit", "Write") and isinstance(inputs, dict) and isinstance(inputs.get("file_path"), str):
        return [inputs["file_path"]]
    if runtime == "codex" and tool == "apply_patch":
        patch = inputs if isinstance(inputs, str) else next(
            (inputs.get(key) for key in ("input", "patch", "command") if isinstance(inputs.get(key), str)), ""
        ) if isinstance(inputs, dict) else ""
        paths = re.findall(r"^\*\*\* (?:Add File:|Update File:|Delete File:|Move to:) (.+)$", patch, re.M)
        if paths:
            return paths
    raise ValueError("edit payload not supported; checks not verified, run manual verifier")


def hook(root, runtime, payload):
    if not isinstance(payload, dict) or payload.get("hook_event_name") not in ("SessionStart", "PostToolUse"):
        raise ValueError("unsupported hook payload; checks not verified, run manual verifier")
    event = payload["hook_event_name"]
    cwd = payload.get("cwd")
    if not isinstance(cwd, str) or not Path(cwd).resolve().is_relative_to(root):
        raise ValueError("hook cwd outside this repository; checks not verified")
    suffix = ""
    if event == "PostToolUse":
        relevant = []
        for path in changed_paths(payload, runtime):
            resolved = (Path(cwd) / path).resolve()
            if resolved.is_relative_to(root):
                relative = resolved.relative_to(root).as_posix()
                if relative.startswith((".agents/", ".claude/", ".codex/", "infra/", "app/", ".github/workflows/")) or relative in ("AGENTS.md", "CLAUDE.md", "README.md", "scripts/verify-pulumi-pack.py", "scripts/test_pulumi_pack.py"):
                    relevant.append(relative)
        if not relevant:
            return None
        suffix = " Relevant files changed; previous checks/review may be stale. Primary: rerun checks from the verify workflow for this diff."
    findings = verify(root)
    message = ("Pulumi pack checks failed: " + " | ".join(findings)) if findings else "Pulumi pack local static checks passed; no Go, cloud or independent review was run."
    return {"hookSpecificOutput": {"hookEventName": event, "additionalContext": message + suffix}}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--hook", choices=("codex", "claude"))
    args = parser.parse_args()
    root = Path(__file__).resolve().parents[1]
    try:
        if args.hook:
            raw = sys.stdin.buffer.read(MAX_BYTES + 1)
            if len(raw) > MAX_BYTES:
                raise ValueError("hook payload exceeds 64 KiB; verification incomplete")
            output = hook(root, args.hook, json.loads(raw))
            if output:
                print(json.dumps(output, ensure_ascii=True))
            return 0
        findings = verify(root)
        print("\n".join(findings) if findings else "Pulumi pack local static checks passed (no cloud operations).")
        return int(bool(findings))
    except (OSError, ValueError, KeyError, TypeError, TimeoutError) as error:
        # Khong in payload, code hay secret tu tool_input.
        message = f"Pulumi pack verification incomplete: {type(error).__name__}; run manual verifier and inspect local config."
        if args.hook:
            print(json.dumps({"systemMessage": message}))
            return 0  # Advisory hook; never block unrelated work or imply a pass.
        print(message, file=sys.stderr)
        return 1


if __name__ == "__main__":
    sys.exit(main())
