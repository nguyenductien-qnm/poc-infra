"""Small pack mechanism tests, not Pulumi mocks or live deployment evidence."""

import importlib.util
import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[1]
spec = importlib.util.spec_from_file_location("pack", ROOT / "scripts/verify-pulumi-pack.py")
pack = importlib.util.module_from_spec(spec)
spec.loader.exec_module(pack)


class PackTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory(prefix="pulumi pack with spaces ")
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name).resolve()
        for directory in (".agents", ".claude", ".codex", ".github/workflows", "docs", "scripts"):
            shutil.copytree(ROOT / directory, self.root / directory, ignore=shutil.ignore_patterns("__pycache__"))
        for file in ("AGENTS.md", "CLAUDE.md"):
            shutil.copy2(ROOT / file, self.root / file)

    def rewrite(self, path, old, new):
        target = self.root / path
        text = target.read_text(encoding="utf-8")
        self.assertIn(old, text)
        target.write_text(text.replace(old, new), encoding="utf-8")

    def payload(self, event="SessionStart", **extra):
        return {"hook_event_name": event, "cwd": str(self.root), **extra}

    def test_checkout_links_and_adapters(self):
        self.assertEqual([], pack.verify(self.root))

    def test_broken_link_and_missing_adapter(self):
        target = self.root / ".agents/skills/pulumi/references/components.md"
        target.unlink()
        (self.root / ".codex/agents/pulumi-component-reviewer.toml").unlink()
        findings = " ".join(pack.verify(self.root))
        self.assertIn("broken or external local link", findings)
        self.assertIn("pulumi-component-reviewer.toml", findings)

    def test_metadata_and_agent_tool_drift(self):
        self.rewrite(".claude/skills/pulumi/SKILL.md", "name: pulumi", "name: unrelated")
        self.rewrite(".claude/agents/pulumi-delivery-reviewer.md", "Read, Grep, Glob", "Read, Grep, Glob, Bash")
        self.rewrite(".codex/agents/pulumi-component-reviewer.toml", 'sandbox_mode = "read-only"', 'sandbox_mode = "workspace-write"')
        findings = " ".join(pack.verify(self.root))
        self.assertIn("skill must be named pulumi", findings)
        self.assertIn("tools must be", findings)
        self.assertIn("read-only sandbox", findings)

    def test_hook_wiring_drift(self):
        path = self.root / ".codex/hooks.json"
        config = json.loads(path.read_text())
        config["hooks"]["PostToolUse"][0]["matcher"] = "Bash"
        path.write_text(json.dumps(config))
        self.assertIn("hook matcher drift", " ".join(pack.verify(self.root)))

    def test_unpinned_action_is_a_static_finding(self):
        path = self.root / ".github/workflows/unpinned.yml"
        path.write_text("jobs:\n  check:\n    steps:\n      - uses: actions/checkout@v4\n")
        self.assertIn("unpinned.yml:4", " ".join(pack.verify(self.root)))

    def test_two_runtime_session_payloads(self):
        for runtime in ("codex", "claude"):
            output = pack.hook(self.root, runtime, self.payload())
            self.assertEqual("SessionStart", output["hookSpecificOutput"]["hookEventName"])

    def test_edit_from_subdirectory_and_unrelated_edit(self):
        subdir = self.root / "infra/workload"
        subdir.mkdir(parents=True)
        relevant = self.payload("PostToolUse", cwd=str(subdir), tool_name="Edit", tool_input={"file_path": "main.go"})
        self.assertIsNotNone(pack.hook(self.root, "claude", relevant))
        unrelated = self.payload("PostToolUse", tool_name="Write", tool_input={"file_path": "notes.txt"})
        self.assertIsNone(pack.hook(self.root, "claude", unrelated))
        external = self.payload("PostToolUse", tool_name="Write", tool_input={"file_path": "../outside.txt"})
        self.assertIsNone(pack.hook(self.root, "claude", external))

    def test_codex_patch_delete_and_move_paths(self):
        patch = "*** Begin Patch\n*** Delete File: notes.txt\n*** Update File: infra/a.go\n*** Move to: infra/b.go\n*** End Patch"
        for inputs in (patch, {"input": patch}, {"patch": patch}, {"command": patch}):
            payload = self.payload("PostToolUse", tool_name="apply_patch", tool_input=inputs)
            self.assertEqual(["notes.txt", "infra/a.go", "infra/b.go"], pack.changed_paths(payload, "codex"))
            self.assertIsNotNone(pack.hook(self.root, "codex", payload))

    def test_unsupported_payload_and_budget_never_pass(self):
        for payload in (None, [], {}, self.payload("Stop"), self.payload("PostToolUse", tool_name="Edit", tool_input={})):
            with self.assertRaises(ValueError):
                pack.hook(self.root, "codex", payload)
        with self.assertRaises(TimeoutError):
            pack.verify(self.root, budget=-1)

    def test_malformed_input_does_not_echo_payload(self):
        result = subprocess.run(
            [os.sys.executable, "-B", str(self.root / "scripts/verify-pulumi-pack.py"), "--hook", "claude"],
            input=b"secret-value-must-not-be-echoed", capture_output=True, timeout=5,
        )
        output = json.loads(result.stdout)
        self.assertIn("incomplete", output["systemMessage"])
        self.assertNotIn(b"secret-value", result.stdout + result.stderr)

    def test_configured_commands_with_spaces_and_subdirectory(self):
        # Execute the actual configured command, not just a reconstructed argv.
        subprocess.run(["git", "init", "-q", str(self.root)], check=True, capture_output=True)
        subdir = self.root / "infra/workload"
        subdir.mkdir(parents=True)
        stdin = json.dumps(self.payload(cwd=str(subdir))).encode()
        codex = json.loads((self.root / ".codex/hooks.json").read_text())["hooks"]["SessionStart"][0]["hooks"][0]
        command = codex["commandWindows" if os.name == "nt" else "command"]
        result = subprocess.run(command, shell=True, cwd=subdir, input=stdin, capture_output=True, timeout=5)
        self.assertEqual(0, result.returncode, result.stderr)
        self.assertIn("hookSpecificOutput", json.loads(result.stdout))
        claude = json.loads((self.root / ".claude/settings.json").read_text())["hooks"]["SessionStart"][0]["hooks"][0]
        argv = [claude["command"], *claude["args"]]
        # Claude sets CLAUDE_PROJECT_DIR to the launch directory, even below root.
        result = subprocess.run(argv, cwd=subdir, env={**os.environ, "CLAUDE_PROJECT_DIR": str(subdir)}, input=stdin, capture_output=True, timeout=5)
        self.assertEqual(0, result.returncode, result.stderr)
        self.assertIn("hookSpecificOutput", json.loads(result.stdout))

    def test_missing_interpreter_in_configured_command(self):
        subprocess.run(["git", "init", "-q", str(self.root)], check=True, capture_output=True)
        config = json.loads((self.root / ".codex/hooks.json").read_text())
        handler = config["hooks"]["SessionStart"][0]["hooks"][0]
        command = handler["commandWindows" if os.name == "nt" else "command"]
        command = command.replace("@python ", "@pulumi-pack-python-not-installed ") if os.name == "nt" else command.replace("python3 ", "pulumi-pack-python-not-installed ")
        result = subprocess.run(command, shell=True, cwd=self.root, input=json.dumps(self.payload()).encode(), capture_output=True, timeout=5)
        self.assertNotEqual(0, result.returncode)
        self.assertNotIn(b"hookSpecificOutput", result.stdout)

    def test_related_edit_reports_failure_but_unrelated_is_silent(self):
        (self.root / ".codex/agents/pulumi-component-reviewer.toml").unlink()
        payload = self.payload("PostToolUse", tool_name="Edit", tool_input={"file_path": "infra/workload/main.go"})
        output = pack.hook(self.root, "claude", payload)["hookSpecificOutput"]["additionalContext"]
        self.assertIn("checks failed", output)
        payload["tool_input"]["file_path"] = "notes.txt"
        self.assertIsNone(pack.hook(self.root, "claude", payload))


if __name__ == "__main__":
    unittest.main()
