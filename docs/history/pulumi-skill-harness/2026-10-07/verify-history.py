import argparse
import gzip
import hashlib
import json
from pathlib import Path, PurePosixPath
import re
import tarfile

ROOT = Path(__file__).resolve().parent
PATTERNS = {
    "aws-key": r"\b(?:AKIA|ASIA)[A-Z0-9]{16}\b",
    "github-token": r"\b(?:gh[pousr]_[A-Za-z0-9]{20,}|github_pat_[A-Za-z0-9_]{20,})\b",
    "model-token": r"\bsk-(?:ant-)?[A-Za-z0-9_-]{16,}\b",
    "bearer-token": r"(?i)Bearer\s+[A-Za-z0-9._-]{20,}",
    "private-key": r"-----BEGIN (?:RSA |EC |OPENSSH )?PRIVATE KEY-----",
    "arn": r"\barn:(?:aws(?:-[a-z0-9-]+)?):[^\s\"'<>\\]+",
    "account-number": r"\b\d{12}\b",
    "jira-user": r"712020:[a-f0-9-]{36}",
}


def digest(raw):
    return hashlib.sha256(raw).hexdigest()


def check(condition, message):
    if not condition:
        raise ValueError(message)


def local_path(relative):
    parts = PurePosixPath(relative)
    check(not parts.is_absolute() and ".." not in parts.parts and "\\" not in relative, "Unsafe archive path")
    path = (ROOT / relative).resolve()
    check(path.is_relative_to(ROOT), "Path outside historical record")
    return path


def check_text(raw, label):
    text = raw.decode("utf-8")
    for name, pattern in PATTERNS.items():
        check(re.search(pattern, text) is None, f"Sensitive pattern {name}: {label}")
    if label.endswith(".json"):
        json.loads(text)
    return text


def load_bundle(record):
    path = local_path(record["archive"])
    raw = path.read_bytes()
    check(digest(raw) == record["archive_sha256"], f"Archive checksum mismatch: {record['run']}")
    check(len(raw) == record["archive_bytes"], f"Archive size mismatch: {record['run']}")
    members = {}
    with tarfile.open(path, "r:gz") as archive:
        for member in archive:
            name = member.name
            local_path(name)
            check(member.isfile() and name not in members, f"Unsupported or duplicate member: {name}")
            raw_member = archive.extractfile(member).read()
            check_text(raw_member, record["run"] + "/" + name)
            members[name] = raw_member
    hashes = {name: digest(raw_member) for name, raw_member in members.items()}
    inventory = digest(json.dumps(hashes, sort_keys=True, separators=(",", ":")).encode())
    check(inventory == record["source_file_inventory_sha256"], f"Source inventory mismatch: {record['run']}")
    check(len(members) == record["source_file_count"], f"Source file count mismatch: {record['run']}")
    for name, expected in record["key_file_sha256"].items():
        check(hashes.get(name) == expected, f"Key record mismatch: {record['run']}/{name}")
    return members


def verify(index):
    check(index["schema"] == 1, "Unknown historical record schema")
    run_names = [record["run"] for record in index["runs"]]
    check(len(set(run_names)) == len(run_names) == index["bundle_count"], "Run count mismatch")
    final = json.loads((ROOT / "decisions/offline-loop-final-decision.json").read_text(encoding="utf-8"))
    check(index["baseline_commit"] == final["canonical_commit"], "Baseline commit mismatch")
    chains = final["grade_integrity"]
    chain_hashes = {run: hashes for runs in chains.values() for run, hashes in runs.items()}
    source_count, graded_count, final_count = 0, 0, 0
    for record in index["records"]:
        raw = local_path(record["path"]).read_bytes()
        check(digest(raw) == record["sha256"] and len(raw) == record["bytes"], f"Receipt mismatch: {record['path']}")
        if record["path"].endswith(".gz"):
            raw = gzip.decompress(raw)
        check(digest(raw) == record["source_sha256"], f"Source receipt mismatch: {record['path']}")
        check_text(raw, record["path"].removesuffix(".gz"))
    for record in index["runs"]:
        members = load_bundle(record)
        source_count += len(members)
        grade = json.loads(members["grade-frozen.json"]) if "grade-frozen.json" in members else None
        check(record["grade_available"] == (grade is not None), "Grade availability mismatch")
        if grade is not None:
            graded_count += 1
            check(record["score"] == grade.get("score"), f"Score mismatch: {record['run']}")
            for key, filename in {"frozen_event_sha256": "events.jsonl", "independent_checks_sha256": "independent-checks.json", "source_freeze_receipt_sha256": "source-freeze.json"}.items():
                if key in grade:
                    check(digest(members[filename]) == grade[key], f"Frozen grade reference mismatch: {record['run']}/{filename}")
        if record["run"] in chain_hashes:
            final_count += 1
            for name, expected in chain_hashes[record["run"]].items():
                check(digest(members[name]) == expected, f"Final chain mismatch: {record['run']}/{name}")
    check(source_count == index["source_file_count"], "Total source file count mismatch")
    check(graded_count == index["graded_bundle_count"], "Graded run count mismatch")
    check(final_count == index["final_chain_grade_count"] == len(chain_hashes), "Final chain coverage mismatch")
    print(json.dumps({"verified": True, "bundles": len(run_names), "captured_files": source_count, "graded_runs": graded_count, "final_chain_grades": final_count, "cloud_operations": 0}))


def main():
    parser = argparse.ArgumentParser(description="Inspect captured historical evidence without extracting or executing it.")
    parser.add_argument("--show", nargs=2, metavar=("RUN", "MEMBER"))
    args = parser.parse_args()
    index = json.loads((ROOT / "index.json").read_text(encoding="utf-8"))
    if args.show:
        run, member = args.show
        record = next((r for r in index["runs"] if r["run"] == run), None)
        check(record is not None, "Unknown run")
        members = load_bundle(record)
        check(member in members, "Unknown member")
        print(members[member].decode("utf-8"), end="")
    else:
        verify(index)


if __name__ == "__main__":
    main()
