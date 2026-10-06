"""Standalone tests for the portable Pulumi skill-pack verifier.

The fixture deliberately contains only a copied canonical pack, its installed
runtime adapter, and generic IaC source.  It must never depend on the host.
"""

import importlib.util
import json
import os
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile
import unittest


CANONICAL = Path(__file__).resolve().parents[1]


class PortablePackTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory(prefix="pulumi portable pack with spaces ")
        self.addCleanup(self.temp.cleanup)
        self.repo = Path(self.temp.name).resolve() / "fresh repository with spaces"
        self.repo.mkdir()
        self.pack = self.repo / ".agents" / "skills" / "pulumi"
        shutil.copytree(CANONICAL, self.pack, ignore=shutil.ignore_patterns("__pycache__"))
        self._seed_repository()
        spec = importlib.util.spec_from_file_location("portable_pack", self.pack / "scripts" / "verify.py")
        self.module = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(self.module)

    def _seed_repository(self):
        (self.repo / "platform" / "live").mkdir(parents=True)
        (self.repo / "platform" / "shared").mkdir(parents=True)
        (self.repo / "platform" / "tools").mkdir(parents=True)
        (self.repo / "service").mkdir()
        (self.repo / "service" / "nested").mkdir()
        (self.repo / "operations").mkdir()
        (self.repo / "platform" / "go.mod").write_text(
            "module example.invalid/iac\n\ngo 1.23\n\nrequire github.com/pulumi/pulumi/sdk/v3 v3.0.0\n",
            encoding="utf-8",
        )
        for relative in ("live/main.go", "shared/network.go"):
            (self.repo / "platform" / relative).write_text("package placeholder\n", encoding="utf-8")
        (self.repo / "platform" / "tools" / "go.mod").write_text("module example.invalid/tools\n\ngo 1.23\n", encoding="utf-8")
        (self.repo / "platform" / "tools" / "helper.go").write_text("package tools\n", encoding="utf-8")
        (self.repo / "platform" / "live" / "Pulumi.yaml").write_text(
            "name: generic-stack\nruntime: go\n", encoding="utf-8"
        )
        (self.repo / "service" / "go.mod").write_text("module example.invalid/service\n\ngo 1.23\n", encoding="utf-8")
        (self.repo / "service" / "main.go").write_text("package main\n", encoding="utf-8")
        (self.repo / "service" / "nested" / "go.mod").write_text("module example.invalid/nested\n\ngo 1.23\n", encoding="utf-8")
        (self.repo / "service" / "nested" / "worker.go").write_text("package nested\n", encoding="utf-8")
        (self.repo / "operations" / "reconcile.yaml").write_text("command: pulumi preview\n", encoding="utf-8")
        (self.repo / "operations" / "application-pipeline.yaml").write_text("command: go test ./...\n", encoding="utf-8")
        subprocess.run(["git", "init", "-q", str(self.repo)], check=True, capture_output=True)

    def install(self, runtime):
        adapter = self.pack / "adapters" / runtime
        self.assertTrue(adapter.is_dir(), adapter)
        for source in adapter.rglob("*"):
            if source.is_dir():
                continue
            target = self.repo / f".{runtime}" / source.relative_to(adapter)
            if target.name == "SKILL.md.in":
                target = target.with_suffix("")
            target.parent.mkdir(parents=True, exist_ok=True)
            shutil.copy2(source, target)

    def payload(self, event="SessionStart", **extra):
        return {"hook_event_name": event, "cwd": str(self.repo), **extra}

    def assert_finding(self, findings, expected):
        self.assertTrue(any(expected in finding for finding in findings), findings)

    def test_canonical_pack_is_self_contained_and_cli_runs_outside_host(self):
        self.assertEqual([], self.module.verify(self.pack))
        result = subprocess.run(
            [sys.executable, "-B", str(self.pack / "scripts" / "verify.py")],
            cwd=self.repo,
            capture_output=True,
            text=True,
            timeout=5,
        )
        self.assertEqual(0, result.returncode, result.stderr)
        self.assertNotIn(str(CANONICAL.parents[3]), result.stdout + result.stderr)

    def test_required_content_link_and_frontmatter_drift_are_findings(self):
        component = self.pack / "references" / "components.md"
        original_component = component.read_text(encoding="utf-8")
        component.unlink()
        self.assert_finding(self.module.verify(self.pack), "components")

        # Recreate the canonical fixture before exercising independent metadata drift.
        component.write_text(original_component, encoding="utf-8")
        skill = self.pack / "SKILL.md"
        original_skill = skill.read_text(encoding="utf-8")
        skill.write_text(original_skill.replace("name: pulumi", "name: unrelated", 1), encoding="utf-8")
        self.assert_finding(self.module.verify(self.pack), "pulumi")
        skill.write_text(original_skill.replace("references/components.md", "references/missing.md", 1), encoding="utf-8")
        self.assert_finding(self.module.verify(self.pack), "missing.md")

    def test_each_runtime_installation_passes_independently(self):
        for runtime in ("codex", "claude"):
            with self.subTest(runtime=runtime):
                self.install(runtime)
                self.assertEqual([], self.module.verify_installation(self.pack, self.repo, runtime))

    def test_adapter_templates_do_not_add_discoverable_skills(self):
        self.assertEqual([self.pack / "SKILL.md"], list(self.pack.rglob("SKILL.md")))
        nested = self.pack / "adapters" / "claude" / "skills" / "pulumi" / "SKILL.md"
        shutil.copy2(self.pack / "SKILL.md", nested)
        self.assert_finding(self.module.verify(self.pack), "nested SKILL.md")

    def test_claude_installation_requires_discoverable_filename(self):
        self.install("claude")
        installed = self.repo / ".claude" / "skills" / "pulumi" / "SKILL.md"
        self.assertTrue(installed.is_file())
        installed.rename(installed.with_suffix(".md.in"))
        self.assert_finding(self.module.verify_installation(self.pack, self.repo, "claude"), "missing")

    def test_behavioral_cases_have_usable_inputs_and_criteria(self):
        corpus = json.loads((self.pack / "evals" / "go-review.json").read_text(encoding="utf-8"))
        self.assertEqual(1, corpus["format"])
        self.assertIsInstance(corpus["cases"], list)
        self.assertTrue(corpus["cases"])
        seen = set()
        for case in corpus["cases"]:
            for key in ("id", "request"):
                self.assertIsInstance(case[key], str)
                self.assertTrue(case[key].strip(), key)
            self.assertNotIn(case["id"], seen, "duplicate case id")
            seen.add(case["id"])
            self.assertIsInstance(case["files"], dict)
            self.assertTrue(case["files"], case["id"])
            for path, source in case["files"].items():
                self.assertIsInstance(path, str)
                self.assertTrue(path.strip(), case["id"])
                self.assertIsInstance(source, str)
                self.assertTrue(source.strip(), path)
            for key in ("expect", "reject"):
                self.assertIsInstance(case[key], list)
                self.assertTrue(case[key], (case["id"], key))
                for criterion in case[key]:
                    self.assertIsInstance(criterion, str)
                    self.assertTrue(criterion.strip(), (case["id"], key))

    def test_selected_runtime_tolerates_unrelated_configuration_but_rejects_owned_drift(self):
        self.install("codex")
        path = self.repo / ".codex" / "hooks.json"
        config = json.loads(path.read_text(encoding="utf-8"))
        config["hooks"]["PostToolUse"].append({"matcher": "^unrelated$", "hooks": [{"type": "command", "command": "echo unrelated"}]})
        path.write_text(json.dumps(config), encoding="utf-8")
        self.assertEqual([], self.module.verify_installation(self.pack, self.repo, "codex"))

        owned = config["hooks"]["SessionStart"][0]["hooks"][0]
        key = "commandWindows" if os.name == "nt" and "commandWindows" in owned else "command"
        owned[key] = "python missing-owned-handler.py"
        path.write_text(json.dumps(config), encoding="utf-8")
        self.assertTrue(self.module.verify_installation(self.pack, self.repo, "codex"))

    def test_generic_pulumi_layout_and_shared_module_are_relevant(self):
        self.install("claude")
        for relative in ("platform/live/main.go", "platform/live/Pulumi.yaml", "platform/shared/network.go"):
            with self.subTest(relative=relative):
                output = self.module.hook(
                    self.pack, "claude",
                    self.payload("PostToolUse", tool_name="Edit", tool_input={"file_path": relative}), self.repo,
                )
                self.assertIsNotNone(output)
                self.assertIn("stale", output["hookSpecificOutput"]["additionalContext"])

    def test_only_pack_edits_rerun_pack_checks(self):
        self.install("claude")
        (self.pack / "references" / "delivery.md").unlink()
        infra = self.module.hook(
            self.pack, "claude",
            self.payload("PostToolUse", tool_name="Edit", tool_input={"file_path": "platform/live/main.go"}), self.repo,
        )
        self.assertNotIn("failed", infra["hookSpecificOutput"]["additionalContext"])
        self.install("codex")
        for relative in (".agents/skills/pulumi/SKILL.md", ".claude/settings.json",
                         ".claude/agents/pulumi-component-reviewer.md", ".codex/agents/pulumi-delivery-reviewer.toml"):
            with self.subTest(relative=relative):
                pack = self.module.hook(
                    self.pack, "claude",
                    self.payload("PostToolUse", tool_name="Edit", tool_input={"file_path": relative}), self.repo,
                )
                self.assertIn("failed", pack["hookSpecificOutput"]["additionalContext"])

    def test_unrelated_runtime_files_are_silent(self):
        self.install("claude")
        self.install("codex")
        unrelated = {
            ".claude/commands/release.md": "# Release\n",
            ".claude/skills/other/SKILL.md": "---\nname: other\ndescription: unrelated\n---\n",
            ".codex/agents/other.toml": "name = \"other\"\n",
        }
        for relative, text in unrelated.items():
            path = self.repo / relative
            path.parent.mkdir(parents=True, exist_ok=True)
            path.write_text(text, encoding="utf-8")
            with self.subTest(relative=relative):
                self.assertIsNone(self.module.hook(
                    self.pack, "claude",
                    self.payload("PostToolUse", tool_name="Edit", tool_input={"file_path": relative}), self.repo,
                ))

    def test_app_module_is_silent_and_mixed_patch_is_relevant(self):
        self.install("codex")
        for relative in ("service/main.go", "service/go.mod"):
            output = self.module.hook(
                self.pack, "codex",
                self.payload("PostToolUse", tool_name="Edit", tool_input={"file_path": relative}), self.repo,
            )
            self.assertIsNone(output)
        patch = "\r\n".join(("*** Begin Patch", "*** Update File: service/main.go", "*** Update File: platform/live/main.go", "*** End Patch"))
        output = self.module.hook(
            self.pack, "codex", self.payload("PostToolUse", tool_name="apply_patch", tool_input={"patch": patch}), self.repo,
        )
        self.assertIn("stale", output["hookSpecificOutput"]["additionalContext"])

    def test_nearest_go_module_and_generic_gitops_boundaries(self):
        self.install("claude")
        silent = ("service/nested/worker.go", "platform/tools/helper.go", "operations/application-pipeline.yaml")
        relevant = ("platform/go.mod", "platform/go.sum", "operations/reconcile.yaml")
        (self.repo / "platform" / "go.sum").write_text("example checksum\n", encoding="utf-8")
        for relative in silent:
            with self.subTest(silent=relative):
                self.assertIsNone(self.module.hook(
                    self.pack, "claude", self.payload("PostToolUse", tool_name="Edit", tool_input={"file_path": relative}), self.repo,
                ))
        for relative in relevant:
            with self.subTest(relevant=relative):
                self.assertIsNotNone(self.module.hook(
                    self.pack, "claude", self.payload("PostToolUse", tool_name="Edit", tool_input={"file_path": relative}), self.repo,
                ))

    def test_deleted_tracked_gitops_marker_uses_bounded_history(self):
        self.install("claude")
        self._commit_fixture_files("operations")
        pulumi_config = self.repo / "operations" / "reconcile.yaml"
        application_config = self.repo / "operations" / "application-pipeline.yaml"
        pulumi_config.write_text("command: reconcile\n", encoding="utf-8")
        retained_marker_removed = self.module.hook(
            self.pack, "claude",
            self.payload("PostToolUse", tool_name="Edit", tool_input={
                "file_path": "operations/reconcile.yaml", "old_string": "command: pulumi preview",
            }), self.repo,
        )
        self.assertIsNotNone(retained_marker_removed)
        self.assertIn("stale", retained_marker_removed["hookSpecificOutput"]["additionalContext"])
        pulumi_config.unlink()
        application_config.unlink()
        deleted_pulumi = self.module.hook(
            self.pack, "claude",
            self.payload("PostToolUse", tool_name="Edit", tool_input={"file_path": "operations/reconcile.yaml"}), self.repo,
        )
        self.assertIsNotNone(deleted_pulumi)
        self.assertIn("stale", deleted_pulumi["hookSpecificOutput"]["additionalContext"])
        self.assertIsNone(self.module.hook(
            self.pack, "claude",
            self.payload("PostToolUse", tool_name="Edit", tool_input={"file_path": "operations/application-pipeline.yaml"}), self.repo,
        ))

    def test_untracked_removed_gitops_preimages_respect_absolute_and_subdirectory_paths(self):
        self.install("codex")
        subdir = self.repo / "platform" / "live"
        for name, changed in (
            ("absolute-removal.yaml", None),
            ("relative-removal.yaml", "../../operations/relative-removal.yaml"),
        ):
            path = self.repo / "operations" / name
            path.write_text("command: pulumi preview\n", encoding="utf-8")
            path.unlink()
            changed = changed or str(path)
            patch = "\r\n".join(("*** Begin Patch", f"*** Delete File: {changed}", "-command: pulumi preview", "*** End Patch"))
            with self.subTest(changed=changed):
                output = self.module.hook(
                    self.pack, "codex",
                    self.payload("PostToolUse", cwd=str(subdir), tool_name="apply_patch", tool_input={"patch": patch}), self.repo,
                )
                self.assertIsNotNone(output)
                self.assertIn("stale", output["hookSpecificOutput"]["additionalContext"])
        old_string = self.module.hook(
            self.pack, "claude",
            self.payload("PostToolUse", cwd=str(subdir), tool_name="Edit", tool_input={
                "file_path": "../../operations/edit-removal.yaml", "old_string": "command: pulumi preview",
            }), self.repo,
        )
        self.assertIsNotNone(old_string)

    def _commit_fixture_files(self, path):
        common = ["git", "-C", str(self.repo), "-c", "user.name=portable-harness", "-c", "user.email=portable-harness@example.invalid"]
        subprocess.run([*common, "add", "--", path], check=True, capture_output=True)
        subprocess.run([*common, "commit", "-q", "-m", "fixture marker"], check=True, capture_output=True)

    def test_template_wiring_drift_is_rejected_before_installation(self):
        path = self.pack / "adapters" / "codex" / "hooks.json"
        original = path.read_text(encoding="utf-8")
        for old, new in (("startup|resume|clear|compact", "wrong-event"), ("\"timeout\": 5", "\"timeout\": 99"), ("python", "not-python"),
                         (".agents/skills/pulumi/scripts/verify.py", ".agents/skills/pulumi/scripts/wrong.py")):
            with self.subTest(old=old):
                path.write_text(original.replace(old, new), encoding="utf-8")
                self.assertTrue(self.module.verify(self.pack))
        path.write_text(original, encoding="utf-8")
        claude = self.pack / "adapters" / "claude" / "agents" / "pulumi-component-reviewer.md"
        original_claude = claude.read_text(encoding="utf-8")
        claude.write_text(original_claude.replace("model: inherit", "model: unrelated", 1), encoding="utf-8")
        self.assertTrue(self.module.verify(self.pack))

    def test_patch_delete_move_and_containment(self):
        self.install("codex")
        patch = "\r\n".join(("*** Begin Patch", "*** Delete File: notes.txt", "*** Update File: platform/a.go", "*** Move to: platform/b.go", "*** End Patch"))
        output = self.module.hook(
            self.pack, "codex", self.payload("PostToolUse", tool_name="apply_patch", tool_input=patch), self.repo,
        )
        self.assertIsNotNone(output)
        external = self.module.hook(
            self.pack, "codex", self.payload("PostToolUse", tool_name="Edit", tool_input={"file_path": "../outside.go"}), self.repo,
        )
        self.assertIsNone(external)

    def test_invalid_payloads_timeouts_and_secrets_do_not_pass_or_echo(self):
        self.install("claude")
        for payload in (None, [], {}, self.payload("Stop"), self.payload("PostToolUse", tool_name="Edit", tool_input={})):
            with self.subTest(payload=repr(payload)):
                with self.assertRaises(ValueError):
                    self.module.hook(self.pack, "claude", payload, self.repo)
        with self.assertRaises(TimeoutError):
            self.module.verify(self.pack, budget=-1)
        result = subprocess.run(
            [sys.executable, "-B", str(self.pack / "scripts" / "verify.py"), "--hook", "claude", "--repo-root", str(self.repo)],
            input=b"secret-value-must-not-be-echoed",
            capture_output=True,
            timeout=5,
        )
        self.assertNotIn(b"secret-value", result.stdout + result.stderr)
        oversized = subprocess.run(
            [sys.executable, "-B", str(self.pack / "scripts" / "verify.py"), "--hook", "claude", "--repo-root", str(self.repo)],
            input=b"x" * 65537,
            capture_output=True,
            timeout=5,
        )
        self.assertNotIn(b"x" * 64, oversized.stdout + oversized.stderr)

    def test_configured_hook_commands_execute_from_spaced_subdirectory(self):
        for runtime in ("codex", "claude"):
            with self.subTest(runtime=runtime):
                self.install(runtime)
                subdir = self.repo / "platform" / "live"
                command = self._configured_command(runtime)
                result = subprocess.run(
                    command, cwd=subdir, input=json.dumps(self.payload(cwd=str(subdir))).encode(), capture_output=True, timeout=5,
                    **self._command_options(command),
                )
                self.assertEqual(0, result.returncode, result.stderr)
                self.assertIn("hookSpecificOutput", json.loads(result.stdout))

    @unittest.skipUnless(os.name == "nt", "Windows command variants are Windows-only")
    def test_codex_windows_handler_runs_in_cmd_and_powershell_and_fails_without_python(self):
        self.install("codex")
        subdir = self.repo / "platform" / "live"
        command = self._configured_command("codex")
        self.assertIsInstance(command, str)
        stdin = json.dumps(self.payload(cwd=str(subdir))).encode()
        # `commandWindows` is CMD syntax; `command` is the PowerShell form.
        result = subprocess.run(command, shell=True, cwd=subdir, input=stdin, capture_output=True, timeout=5)
        self.assertEqual(0, result.returncode, result.stderr)
        self.assertIn("hookSpecificOutput", json.loads(result.stdout))
        handler = json.loads((self.repo / ".codex" / "hooks.json").read_text(encoding="utf-8"))["hooks"]["SessionStart"][0]["hooks"][0]
        result = subprocess.run(["powershell.exe", "-NoProfile", "-NonInteractive", "-Command", handler["command"]], cwd=subdir, input=stdin, capture_output=True, timeout=5)
        self.assertEqual(0, result.returncode, result.stderr)
        self.assertIn("hookSpecificOutput", json.loads(result.stdout))
        missing = command.replace("python -c", "pulumi-pack-python-not-installed -c", 1)
        result = subprocess.run(missing, shell=True, cwd=subdir, input=stdin, capture_output=True, timeout=5)
        self.assertNotEqual(0, result.returncode)
        self.assertNotIn(b"hookSpecificOutput", result.stdout)

    def _configured_command(self, runtime):
        if runtime == "codex":
            handler = json.loads((self.repo / ".codex" / "hooks.json").read_text(encoding="utf-8"))["hooks"]["SessionStart"][0]["hooks"][0]
            return handler["commandWindows"] if os.name == "nt" and "commandWindows" in handler else handler["command"]
        handler = json.loads((self.repo / ".claude" / "settings.json").read_text(encoding="utf-8"))["hooks"]["SessionStart"][0]["hooks"][0]
        return [handler["command"], *handler.get("args", [])]

    @staticmethod
    def _command_options(command):
        if isinstance(command, str):
            return {"shell": True}
        return {"env": {**os.environ, "CLAUDE_PROJECT_DIR": str(Path.cwd())}}


if __name__ == "__main__":
    unittest.main()
