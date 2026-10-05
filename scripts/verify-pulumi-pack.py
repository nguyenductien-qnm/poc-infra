"""Keep the existing CI command while the portable verifier lives in the skill."""

from pathlib import Path
import runpy


if __name__ == "__main__":
    runpy.run_path(
        str(Path(__file__).resolve().parents[1] / ".agents/skills/pulumi/scripts/verify.py"),
        run_name="__main__",
    )
