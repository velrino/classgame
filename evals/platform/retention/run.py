#!/usr/bin/env python3
"""Opt-in retention roundtrip in an existing private loopback Firestore emulator."""
import argparse
import datetime as dt
import hashlib
import ipaddress
import json
import os
from pathlib import Path
import secrets
import subprocess
import tempfile
from urllib.parse import urlsplit

ROOT = Path(__file__).resolve().parents[3]

def main():
    parser = argparse.ArgumentParser(description=__doc__, allow_abbrev=False)
    parser.add_argument("--run", action="store_true")
    parser.add_argument("--project", default="demo-classgame-retention")
    options = parser.parse_args()
    endpoint = os.environ.get("FIRESTORE_EMULATOR_HOST", "")
    try:
        address = urlsplit("//" + endpoint)
        port = address.port
        local = ipaddress.ip_address(address.hostname or "").is_loopback and isinstance(port, int) and 0 < port < 65536
    except ValueError:
        local = False
    if not options.run or not options.project.startswith("demo-") or not local or address.username is not None or address.path or address.query or address.fragment:
        parser.error("requires --run, a demo- project and numeric loopback FIRESTORE_EMULATOR_HOST")
    with tempfile.TemporaryDirectory(prefix="classgame-retention-commands-") as temporary:
        directory = Path(temporary)
        for command in ["classbackup", "retentionsmoke"]:
            result = subprocess.run(["go", "build", "-o", str(directory / command), "./cmd/" + command], cwd=ROOT / "api",
                                    env=dict(os.environ, GOPROXY="off"), text=True, capture_output=True, timeout=180)
            if result.returncode:
                raise RuntimeError(result.stderr)
        environment = {"PATH": os.environ.get("PATH", os.defpath), "TZ": "UTC", "FIRESTORE_EMULATOR_HOST": endpoint,
                       "RETENTION_BACKUP_COMMAND": str(directory / "classbackup")}
        result = subprocess.run([str(directory / "retentionsmoke"), "-run", "-project", options.project], env=environment,
                                text=True, capture_output=True, timeout=210)
        if not result.stdout:
            raise RuntimeError(result.stderr or "missing retention report")
        report = json.loads(result.stdout)
        if report.get("mode") != "firestore-emulator-retention" or report.get("project") != options.project or report.get("real_provider_calls") != 0:
            raise RuntimeError("invalid retention report")
    now = dt.datetime.now(dt.timezone.utc)
    output = ROOT / "evals/output" / (now.strftime("%Y%m%dT%H%M%S.%fZ") + "-retention-" + secrets.token_hex(6))
    output.mkdir(parents=True, mode=0o700)
    manifest = {"suite": "retention", "mode": "firestore-emulator", "project": options.project, "created_at": now.isoformat(),
                "revision": subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=ROOT, text=True).strip(),
                "working_tree_dirty": bool(subprocess.check_output(["git", "status", "--porcelain"], cwd=ROOT, text=True)),
                "harness_sha256": hashlib.sha256(Path(__file__).read_bytes()).hexdigest(), "real_provider_calls": 0,
                "passed": report["passed"], "total": report["total"], "cleanup_complete": report["cleanup_complete"]}
    for name, value in [("manifest.json", manifest), ("report.json", report)]:
        with (output / name).open("x") as target:
            json.dump(value, target, indent=2)
            target.write("\n")
    passed = result.returncode == 0 and report["passed"] == report["total"] and report["cleanup_complete"]
    print(("PASS" if passed else "FAIL") + " {}/{} real local retention cases; provider calls: 0".format(report["passed"], report["total"]))
    print("Run:", output)
    if not passed:
        print(result.stderr)
    return 0 if passed else 1

if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except (OSError, ValueError, RuntimeError, subprocess.SubprocessError) as error:
        print("retention:", error)
        raise SystemExit(2)
