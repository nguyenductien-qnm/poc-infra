"""Expose the portable tests to the existing CI discovery command."""

from pathlib import Path
import runpy


PortablePackTests = runpy.run_path(
    str(Path(__file__).resolve().parents[1] / ".agents/skills/pulumi/scripts/test_verify.py")
)["PortablePackTests"]
