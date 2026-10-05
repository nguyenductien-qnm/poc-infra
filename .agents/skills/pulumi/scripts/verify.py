"""Portable, read-only Pulumi Go pack verification for Python 3.11+."""

import argparse
import json
from pathlib import Path
import re
import subprocess
import sys
import time
import tomllib

MAX_BYTES = 65536
MAX_FINDINGS = 8
ROLES = ("component", "delivery")
CORE_FILES = (
    "SKILL.md",
    "README.md",
    "references/components.md",
    "references/delivery.md",
    "references/runtime.md",
    "references/sdk-verification.md",
    "workflows/adopt.md",
    "workflows/verify.md",
    "agents/component-reviewer.md",
    "agents/delivery-reviewer.md",
)
LINK = re.compile(r"\[[^\]]+\]\(([^\s)]+)\)")


def _read(path):
    if path.stat().st_size > MAX_BYTES:
        raise ValueError("file exceeds 64 KiB check limit")
    return path.read_text(encoding="utf-8-sig")


def _frontmatter(text):
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


def _templates(pack_root, runtime):
    base = pack_root / "adapters" / runtime
    names = (["hooks.json", *[f"agents/pulumi-{role}-reviewer.toml" for role in ROLES]]
             if runtime == "codex" else
             ["settings.json", "skills/pulumi/SKILL.md", *[f"agents/pulumi-{role}-reviewer.md" for role in ROLES]])
    return [(name, base / name) for name in names]


def verify(pack_root, budget=2.0):
    """Check the self-contained pack and bundled adapters without repository assumptions."""
    pack_root = Path(pack_root).resolve()
    started, findings = time.monotonic(), []

    def check_time():
        if time.monotonic() - started > budget:
            raise TimeoutError("pack check timed out; verification incomplete")

    def note(path, message):
        if len(findings) < MAX_FINDINGS:
            findings.append(f"{path}: {message}"[:500])

    def checked(relative, callback=lambda text: None):
        check_time()
        try:
            callback(_read(pack_root / relative))
        except (OSError, ValueError, KeyError, TypeError, re.error, tomllib.TOMLDecodeError) as error:
            note(relative, str(error))

    def links(relative, text):
        for target in LINK.findall(text):
            local = target.split("#", 1)[0]
            if not local or re.match(r"[a-zA-Z][a-zA-Z0-9+.-]*:", local):
                continue
            resolved = (pack_root / relative).parent.joinpath(local).resolve()
            if not resolved.is_relative_to(pack_root) or not resolved.exists():
                note(relative, f"broken or escaping local link: {target}")

    for relative in CORE_FILES:
        checked(relative, lambda text, relative=relative: links(relative, text))
    for runtime in ("codex", "claude"):
        for name, source in _templates(pack_root, runtime):
            relative = source.relative_to(pack_root).as_posix()
            checked(relative)

    def skill(text):
        if _frontmatter(text)["name"] != "pulumi":
            raise ValueError("skill must be named pulumi")

    checked("SKILL.md", skill)
    try:
        core_fields = _frontmatter(_read(pack_root / "SKILL.md"))
    except (OSError, ValueError):
        core_fields = {}
    for runtime in ("codex", "claude"):
        hook_file = "hooks.json" if runtime == "codex" else "settings.json"
        def adapter_hooks(text, runtime=runtime):
            events = json.loads(text)["hooks"]
            for event, matcher in (("SessionStart", "startup|resume|clear|compact"),
                                   ("PostToolUse", "^(apply_patch|Edit|Write)$" if runtime == "codex" else "^(Edit|Write)$")):
                group = events[event][0]
                handler = group["hooks"][0]
                if group.get("matcher") != matcher or handler.get("type") != "command" or handler.get("timeout") != 5:
                    raise ValueError("adapter hook matcher or timeout drift")
                launcher = handler.get("commandWindows", "") if runtime == "codex" else " ".join(handler.get("args", []))
                command = handler.get("command", "")
                if not isinstance(command, str) or not command.startswith("python"):
                    raise ValueError("adapter hook must use Python")
                if runtime == "codex" and (".agents/skills/pulumi/scripts/verify.py" not in command or
                                           f"--hook {runtime}" not in command or "--repo-root" not in command):
                    raise ValueError("adapter hook must launch the canonical verifier")
                if ".agents/skills/pulumi/scripts/verify.py" not in launcher or f"--hook','{runtime}'" not in launcher or "--repo-root" not in launcher:
                    raise ValueError("adapter hook must launch the canonical verifier")
        checked(f"adapters/{runtime}/{hook_file}", adapter_hooks)
    def claude_router(text):
        if _frontmatter(text) != core_fields or "../../../.agents/skills/pulumi/SKILL.md" not in text:
            raise ValueError("Claude router metadata or source pointer drift")
    checked("adapters/claude/skills/pulumi/SKILL.md", claude_router)
    for role in ROLES:
        def codex_agent(text, role=role):
            fields = tomllib.loads(text)
            if fields.get("name") != f"pulumi-{role}-reviewer" or not fields.get("description"):
                raise ValueError("invalid Codex reviewer identity")
            if fields.get("sandbox_mode") != "read-only" or fields.get("web_search") != "disabled":
                raise ValueError("Codex reviewer must be read-only with web search disabled")
            if f"agents/{role}-reviewer.md" not in fields.get("developer_instructions", ""):
                raise ValueError("Codex reviewer must point at its shared contract")

        def claude_agent(text, role=role):
            fields = _frontmatter(text)
            if fields.get("name") != f"pulumi-{role}-reviewer" or fields.get("model") != "inherit":
                raise ValueError("invalid Claude reviewer identity/model")
            if set(fields.get("tools", "").replace(" ", "").split(",")) != {"Read", "Grep", "Glob"}:
                raise ValueError("Claude reviewer tools must be Read, Grep, Glob only")
            if f"agents/{role}-reviewer.md" not in text:
                raise ValueError("Claude reviewer must point at its shared contract")

        checked(f"adapters/codex/agents/pulumi-{role}-reviewer.toml", codex_agent)
        checked(f"adapters/claude/agents/pulumi-{role}-reviewer.md", claude_agent)
    check_time()
    return findings


def _same_text(left, right):
    return _read(left) == _read(right)


def _owns_expected_hooks(template, installed):
    """Require each bundled handler while allowing unrelated user hook groups."""
    expected, actual = json.loads(_read(template)).get("hooks", {}), json.loads(_read(installed)).get("hooks", {})
    for event, groups in expected.items():
        available = actual.get(event, [])
        for group in groups:
            matcher = group.get("matcher")
            expected_hooks = group.get("hooks")
            if not any(candidate.get("matcher") == matcher and all(hook in candidate.get("hooks", []) for hook in expected_hooks)
                       for candidate in available if isinstance(candidate, dict)):
                return False
    return True


def verify_installation(pack_root, repo_root, runtime, budget=2.0):
    """Check one runtime's installed files against its bundled adapter templates."""
    pack_root, repo_root = Path(pack_root).resolve(), Path(repo_root).resolve()
    if runtime not in ("codex", "claude"):
        raise ValueError("runtime must be codex or claude")
    started, findings = time.monotonic(), []

    def note(path, message):
        if len(findings) < MAX_FINDINGS:
            findings.append(f"{path}: {message}"[:500])

    for name, source in _templates(pack_root, runtime):
        if time.monotonic() - started > budget:
            raise TimeoutError("installation check timed out; verification incomplete")
        target = repo_root / (".codex" if runtime == "codex" else ".claude") / name
        try:
            if not target.is_file():
                raise ValueError("installed adapter is missing")
            if not _same_text(source, target):
                if target.suffix != ".json" or not _owns_expected_hooks(source, target):
                    raise ValueError("installed adapter differs from bundled template")
            for link in LINK.findall(_read(target)):
                local = link.split("#", 1)[0]
                if local and not re.match(r"[a-zA-Z][a-zA-Z0-9+.-]*:", local):
                    resolved = target.parent.joinpath(local).resolve()
                    if not resolved.is_relative_to(repo_root) or not resolved.exists():
                        raise ValueError(f"adapter link escapes installation: {link}")
        except (OSError, ValueError) as error:
            note(target.relative_to(repo_root).as_posix(), str(error))
    return findings


def changed_paths(payload, runtime):
    tool, inputs = payload.get("tool_name"), payload.get("tool_input")
    if tool in ("Edit", "Write") and isinstance(inputs, dict) and isinstance(inputs.get("file_path"), str):
        return [inputs["file_path"]]
    if runtime == "codex" and tool == "apply_patch":
        if isinstance(inputs, dict):
            inputs = next((inputs.get(key) for key in ("input", "patch", "command") if isinstance(inputs.get(key), str)), "")
        if isinstance(inputs, str):
            return re.findall(r"^\*\*\* (?:Add File:|Update File:|Delete File:|Move to:) ([^\r\n]+)\r?$", inputs, re.M)
    raise ValueError("edit payload not supported; run manual verify")


def _nearest_pulumi_module(path, repo_root):
    current = path.parent
    for _ in range(20):
        if not current.is_relative_to(repo_root):
            return False
        manifest = current / "go.mod"
        try:
            if manifest.is_file():
                return "github.com/pulumi/pulumi/sdk" in _read(manifest)
        except (OSError, ValueError):
            return False
        if current == repo_root:
            return False
        current = current.parent
    return False


def _gitops_config(path):
    return path.suffix.lower() in (".yml", ".yaml", ".json", ".toml")


def _payload_preimage_has_pulumi(payload, runtime, changed):
    """Use supplied pre-edit text without retaining or reporting it."""
    inputs = payload.get("tool_input")
    if isinstance(inputs, dict):
        for key in ("old_string", "oldText", "old_content"):
            value = inputs.get(key)
            if isinstance(value, str) and "pulumi" in value.lower():
                return True
        if runtime == "codex":
            inputs = next((inputs.get(key) for key in ("input", "patch", "command") if isinstance(inputs.get(key), str)), "")
    if runtime == "codex" and isinstance(inputs, str):
        escaped = re.escape(changed)
        section = re.search(rf"^\*\*\* (?:Update|Delete) File: {escaped}\r?$([\s\S]*?)(?=^\*\*\* (?:Add|Update|Delete|Move to:) |\Z)", inputs, re.M)
        return bool(section and re.search(r"^-.*pulumi", section.group(1), re.I | re.M))
    return False


def _git_preimage_has_pulumi(path, repo_root, started):
    """Bounded, optional hook-only lookup of a tracked file's HEAD content."""
    if time.monotonic() - started > 0.5:
        return False
    relative = path.relative_to(repo_root).as_posix()
    try:
        size = subprocess.run(["git", "cat-file", "-s", f"HEAD:{relative}"], cwd=repo_root,
                              capture_output=True, text=True, timeout=0.15, check=True).stdout.strip()
        if not size.isdecimal() or int(size) > MAX_BYTES:
            return False
        content = subprocess.run(["git", "show", f"HEAD:{relative}"], cwd=repo_root,
                                 capture_output=True, timeout=0.15, check=True).stdout
        return b"pulumi" in content.lower()
    except (OSError, subprocess.SubprocessError, ValueError):
        return False


def _relevant(path, repo_root, payload=None, runtime=None, started=None, changed=None):
    relative = path.relative_to(repo_root)
    parts, name = relative.parts, relative.name
    if len(parts) >= 3 and parts[:3] == (".agents", "skills", "pulumi"):
        return True
    if name == "Pulumi.yaml" or (name.startswith("Pulumi.") and path.suffix.lower() in (".yaml", ".yml", ".json")):
        return True
    if path.suffix == ".go" or name in ("go.mod", "go.sum"):
        if _nearest_pulumi_module(path, repo_root):
            return True
    if _gitops_config(path):
        try:
            if "pulumi" in _read(path).lower():
                return True
        except (OSError, ValueError):
            pass
        if payload is not None and runtime is not None and _payload_preimage_has_pulumi(payload, runtime, changed or str(relative)):
            return True
        if started is not None:
            return _git_preimage_has_pulumi(path, repo_root, started)
    return False


def hook(pack_root, runtime, payload, repo_root):
    """Return advisory hook JSON only for pack, Pulumi, or Pulumi GitOps edits."""
    pack_root, repo_root = Path(pack_root).resolve(), Path(repo_root).resolve()
    if not isinstance(payload, dict) or payload.get("hook_event_name") not in ("SessionStart", "PostToolUse"):
        raise ValueError("unsupported hook payload; run manual verify")
    if runtime not in ("codex", "claude"):
        raise ValueError("runtime must be codex or claude")
    cwd = payload.get("cwd")
    if not isinstance(cwd, str) or not Path(cwd).resolve().is_relative_to(repo_root):
        raise ValueError("hook cwd outside repository; run manual verify")
    event = payload["hook_event_name"]
    started = time.monotonic()
    stale = False
    if event == "PostToolUse":
        for changed in changed_paths(payload, runtime):
            candidate = (Path(cwd).resolve() / changed).resolve()
            if candidate.is_relative_to(repo_root) and _relevant(candidate, repo_root, payload, runtime, started, changed):
                stale = True
                break
        if not stale:
            return None
    remaining = 2.0 - (time.monotonic() - started)
    if remaining <= 0:
        raise TimeoutError("hook check timed out; verification incomplete")
    findings = verify(pack_root, budget=remaining)
    remaining = 2.0 - (time.monotonic() - started)
    if remaining <= 0:
        raise TimeoutError("hook check timed out; verification incomplete")
    findings.extend(verify_installation(pack_root, repo_root, runtime, budget=remaining))
    findings = findings[:MAX_FINDINGS]
    message = ("Pulumi pack checks failed: " + " | ".join(findings) if findings else
               "Pulumi pack local static checks passed; no Go, cloud, or independent review was run.")
    if stale:
        message += " Relevant files changed; previous checks/review may be stale. Run manual verify for ambiguous or uncovered edits."
    return {"hookSpecificOutput": {"hookEventName": event, "additionalContext": message}}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--repo-root", type=Path)
    parser.add_argument("--runtime", choices=("codex", "claude"))
    parser.add_argument("--hook", choices=("codex", "claude"))
    args = parser.parse_args()
    if bool(args.repo_root) != bool(args.runtime) and not args.hook:
        parser.error("--repo-root and --runtime must be used together")
    if args.hook and not args.repo_root:
        parser.error("--hook requires --repo-root")
    pack_root = Path(__file__).resolve().parents[1]
    try:
        if args.hook:
            raw = sys.stdin.buffer.read(MAX_BYTES + 1)
            if len(raw) > MAX_BYTES:
                raise ValueError("hook payload exceeds 64 KiB")
            output = hook(pack_root, args.hook, json.loads(raw), args.repo_root)
            if output:
                print(json.dumps(output, ensure_ascii=True))
            return 0
        findings = verify(pack_root)
        if args.repo_root:
            findings.extend(verify_installation(pack_root, args.repo_root, args.runtime))
        print("\n".join(findings) if findings else "Pulumi pack local static checks passed (no cloud operations).")
        return int(bool(findings))
    except (OSError, ValueError, KeyError, TypeError, TimeoutError, re.error, tomllib.TOMLDecodeError):
        message = "Pulumi pack verification incomplete; run manual verify and inspect local configuration."
        if args.hook:
            print(json.dumps({"systemMessage": message}))
            return 0
        print(message, file=sys.stderr)
        return 1


if __name__ == "__main__":
    sys.exit(main())
